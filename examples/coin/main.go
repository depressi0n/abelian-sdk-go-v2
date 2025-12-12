package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func ScanCoins(viewAccounts []*database.ViewAccount, tx *abelian.Tx, isCoinbaseTx bool, blockID string, blockHeight int64) error {
	serializedTxOuts := make([][]byte, len(tx.Vout))
	for index := 0; index < len(tx.Vout); index++ {
		txOutData, err := hex.DecodeString(tx.Vout[index].Script)
		if err != nil {
			return fmt.Errorf("fail to decode output of transaction: %v", err)
		}
		serializedTxOuts[index] = txOutData
	}

	serializedTx, err := hex.DecodeString(tx.Hex)
	if err != nil {
		return err
	}
	extAutScript, err := abelian.ExtractAutScriptFromHostTx(serializedTx)
	if err != nil {
		return err
	}
	var generatedTokens []*abelian.CTAUTToken
	if extAutScript != nil {
		generatedTokens, err = abelian.GetGeneratedCTAUTTokens(extAutScript)
		if err != nil {
			return err
		}

		switch extAutScript.Type() {
		case abelian.CTAUTTypeRegistration:
			metadata, err := abelian.RegisteredCTAUTMetadata(extAutScript)
			if err != nil {
				return err
			}
			fmt.Println("Register CT-AUT instance: ")
			fmt.Println("\t Version: ", metadata.Version)
			fmt.Println("\t Identifier: ", metadata.AutIdentifier)
			fmt.Println("\t Name: ", metadata.AutName)
			fmt.Println("\t Symbol: ", metadata.AutSymbol)
			fmt.Println("\t Base Unit Name: ", metadata.BaseUnitName)
			fmt.Println("\t Sub Unit Name: ", metadata.SubUnitName)
			fmt.Println("\t Unit Scale: ", metadata.UnitScale)
			fmt.Println("\t Memo: ", metadata.AutMemo)

			fmt.Println("\t Planned Total Supply: ", metadata.PlannedTotalSupply)
			fmt.Printf("\t %d Issuers: \n", len(metadata.Issuers))
			for i := 0; i < len(metadata.Issuers); i++ {
				fmt.Println("\t\t", metadata.Issuers[i])
			}
			fmt.Println("\t Privacy Type: ", metadata.PrivacyType)
			fmt.Println("\t Re-Registration Expire Height: ", metadata.ReregistrationExpireHeight)
			fmt.Println("\t Re-Registration Threshold: ", metadata.ReregistrationThreshold)
			fmt.Println("\t Mint Threshold: ", metadata.MintThreshold)

			fmt.Printf("\t %d Root Tokens: \n", len(generatedTokens))
			for i := 0; i < len(generatedTokens); i++ {
				fmt.Printf("\t\t (%s,%d)\n", generatedTokens[i].HostOutpoint.TxHash, generatedTokens[i].HostOutpoint.Index)
			}

			fmt.Println("\t Update Script Versions: ", metadata.UpdateScriptVersions)

			dbMetadata := &database.Metadata{
				RegisteredTxID:             tx.TxID,
				Version:                    metadata.Version,
				Identifier:                 metadata.AutIdentifier.String(),
				Name:                       metadata.AutName,
				Symbol:                     metadata.AutSymbol,
				BaseUnitName:               metadata.BaseUnitName,
				SubUnitName:                metadata.SubUnitName,
				UnitScale:                  metadata.UnitScale,
				Memo:                       metadata.AutMemo,
				PlannedTotalAmount:         metadata.PlannedTotalSupply,
				Issuers:                    strings.Join(metadata.Issuers, ";"),
				ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
				ReregistrationThreshold:    metadata.ReregistrationThreshold,
				MintThreshold:              metadata.MintThreshold,
				MintedAmount:               0,
				BurnedAmount:               0,
				UpdateScriptVersions:       metadata.UpdateScriptVersions,
			}
			_, err = database.InsertCTAUTInstance(dbMetadata)
			if err != nil {
				return err
			}
		case abelian.CTAUTTypeReRegistration:
			fallthrough
		case abelian.CTAUTTypeMint:
			fallthrough
		case abelian.CTAUTTypeTransfer:
			fallthrough
		case abelian.CTAUTTypeBurn:
			// get instance from database
			identifier := extAutScript.AutIdentifier()
			metadata, err := database.LoadCTAUTMetadata(identifier.String())
			if err != nil {
				fmt.Errorf("fail to load CT-AUT instance: %v", err)
				return err
			}
			if metadata == nil {
				return fmt.Errorf("no such CT-AUT instance found")
			}
			activeRootTokens, err := database.LoadActiveRootTokens(identifier.String())
			if err != nil {
				return err
			}

			activeRootTokenSet := make(map[abelian.OutPoint]struct{})
			for _, token := range activeRootTokens {
				activeRootTokenSet[abelian.OutPoint{
					TxHash: token.TxID,
					Index:  token.Index,
				}] = struct{}{}
			}

			ctAUTMetadata := &abelian.CTAUTMetadata{
				Version:                    metadata.Version,
				AutIdentifier:              identifier,
				AutName:                    metadata.Name,
				AutSymbol:                  metadata.Symbol,
				BaseUnitName:               metadata.BaseUnitName,
				SubUnitName:                metadata.SubUnitName,
				UnitScale:                  metadata.UnitScale,
				AutMemo:                    metadata.Memo,
				PlannedTotalSupply:         metadata.PlannedTotalAmount,
				Issuers:                    strings.Split(metadata.Issuers, ";"),
				ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
				ReregistrationThreshold:    metadata.ReregistrationThreshold,
				MintThreshold:              metadata.MintThreshold,
				PrivacyType:                metadata.PrivacyType,
				MintedAmount:               metadata.MintedAmount,
				BurnedAmount:               metadata.BurnedAmount,
				ActiveRootTokenSet:         activeRootTokenSet,
				UpdateScriptVersions:       metadata.UpdateScriptVersions,
			}
			updatedAUTMetadata, err := abelian.UpdateMetadataFromCTAUTScript(extAutScript, ctAUTMetadata)
			if err != nil {
				fmt.Errorf("fail to update instance: %s", err)
				return err
			}

			metadata.Version = updatedAUTMetadata.Version
			metadata.Memo = updatedAUTMetadata.AutMemo
			metadata.PlannedTotalAmount = updatedAUTMetadata.PlannedTotalSupply
			metadata.Issuers = strings.Join(updatedAUTMetadata.Issuers, ";")
			metadata.PrivacyType = uint8(updatedAUTMetadata.PrivacyType)
			metadata.ReregistrationExpireHeight = updatedAUTMetadata.ReregistrationExpireHeight
			metadata.ReregistrationThreshold = updatedAUTMetadata.ReregistrationThreshold
			metadata.MintThreshold = updatedAUTMetadata.MintThreshold

			metadata.MintedAmount = updatedAUTMetadata.MintedAmount
			metadata.BurnedAmount = updatedAUTMetadata.BurnedAmount
			metadata.UpdateScriptVersions = updatedAUTMetadata.UpdateScriptVersions

			// update instance to database
			err = database.UpdateCTAUTMetadata(metadata)
			if err != nil {
				return err
			}

			fmt.Println("Updated CT-AUT metadata:")
			fmt.Printf("\t Identifier : %v -> %v \n", ctAUTMetadata.AutIdentifier, metadata.Identifier)
			fmt.Printf("\t Version : %v -> %v  \n", ctAUTMetadata.Version, metadata.Version)
			fmt.Printf("\t Memo : %v -> %v \n", ctAUTMetadata.AutMemo, metadata.Memo)

			issuers := strings.Split(metadata.Issuers, ";")
			fmt.Printf("\t %d Issuers: \n", len(issuers))
			for i := 0; i < len(issuers); i++ {
				fmt.Printf("\t\t %s \n", issuers[i])
			}

			fmt.Printf("\t %d Tokens: \n", len(generatedTokens))
			for i := 0; i < len(generatedTokens); i++ {
				fmt.Printf("\t\t (%s,%d)\n", generatedTokens[i].HostOutpoint.TxHash, generatedTokens[i].HostOutpoint.Index)
			}

			fmt.Println("\t Planned Total Supply: ", metadata.PlannedTotalAmount)
			fmt.Println("\t Privacy Type: ", metadata.PrivacyType)
			fmt.Println("\t Re-Registration Height: ", metadata.ReregistrationExpireHeight)
			fmt.Println("\t Re-Registration Threshold: ", metadata.ReregistrationThreshold)
			fmt.Println("\t Mint Threshold: ", metadata.MintThreshold)

			fmt.Println("\t Minted Amount: ", metadata.MintedAmount)
			fmt.Println("\t Burned Amount: ", metadata.BurnedAmount)
			fmt.Println("\t Update Script Versions: ", metadata.UpdateScriptVersions)
		default:
			return fmt.Errorf("unsupported CT-AUT script type: %d", extAutScript.Type())
		}
	}

	for index := 0; index < len(serializedTxOuts); index++ {
		txOutData := serializedTxOuts[index]
		for _, viewAccount := range viewAccounts {
			success, value, err := viewAccount.ReceiveCoin(uint32(tx.Version), txOutData)
			if err != nil {
				return fmt.Errorf("fail to decode output of transaction: %v", err)
			}
			if !success {
				continue
			}
			coinAddress, err := crypto.DecodeCoinAddressFromSerializedTxOutData(uint32(tx.Version), txOutData)
			if err != nil {
				return fmt.Errorf("fail to decode coin address from transaction output")
			}

			fmt.Printf("💰 Find coin of account with account id %d: block id %s, block height %d, transacion id %s, index %d, value %v ABELs to address fingerprint %s\n",
				viewAccount.ID, blockID, blockHeight, tx.TxHash, index, abelian.NeutrinoToAbel(int64(value)), hex.EncodeToString(coinAddress.Fingerprint()))

			_, err = database.InsertCoin(viewAccount.ID, tx.Version, tx.TxID, uint8(index), blockID, blockHeight, int64(value), isCoinbaseTx, txOutData)
			if err != nil {
				return fmt.Errorf("fail to store coin into database: %v", err)
			}

			if extAutScript != nil {
				identifier := extAutScript.AutIdentifier()
				isRootToken := extAutScript.Type() == abelian.CTAUTTypeRegistration || extAutScript.Type() == abelian.CTAUTTypeReRegistration

				var token *abelian.CTAUTToken
				for i := 0; i < len(generatedTokens); i++ {
					if generatedTokens[i].HostOutpoint.Index == uint8(index) {
						token = generatedTokens[i]
						break
					}
				}

				if token != nil {
					success, ctAutValue, tokenType, cryptoValuePK, cryptoValueSK, err := viewAccount.ReceiveCTAUTToken(
						extAutScript.Version(), extAutScript.Type(), token.ValueScript,
						tx.Version, txOutData)
					if err != nil {
						return fmt.Errorf("fail to get the amount of CT-AUT token: %v", err)
					}
					if !success {
						continue
					}

					// note that the last token would be marked burned
					if extAutScript.Type() == abelian.CTAUTTypeBurn {
						if token.HostOutpoint.Index == generatedTokens[len(generatedTokens)-1].HostOutpoint.Index {
							continue
						}
					}

					if isRootToken {
						fmt.Printf("💲 Find CT-AUT Root token for CT-AUT with identifier %s of account with account id %d: block id %s, block height %d, transacion id %s, index %d, value %v to address fingerprint %s\n",
							identifier.String(), viewAccount.ID, blockID, blockHeight, tx.TxHash, index, ctAutValue, hex.EncodeToString(coinAddress.Fingerprint()))
					} else {
						fmt.Printf("💲 Find CT-AUT token for CT-AUT with identifier %s of account with account id %d: block id %s, block height %d, transacion id %s, index %d, value %v to address fingerprint %s\n",
							identifier.String(), viewAccount.ID, blockID, blockHeight, tx.TxHash, index, ctAutValue, hex.EncodeToString(coinAddress.Fingerprint()))
					}

					_, err = database.InsertToken(viewAccount.ID, identifier.String(),
						tx.TxID, uint8(index),
						isRootToken, uint8(tokenType),
						token.ValueScript,
						cryptoValuePK, cryptoValueSK,
						int64(ctAutValue), extAutScript.Version())
					if err != nil {
						return fmt.Errorf("fail to store coin into database: %v", err)
					}
				}
			}
		}
	}
	if extAutScript != nil && extAutScript.Type() == abelian.CTAUTTypeReRegistration {
		identifier := extAutScript.AutIdentifier()
		for _, viewAccount := range viewAccounts {
			// disable all other root tokens of the same CT-AUT
			disabledTokens, err := database.DisableRootToken(viewAccount.ID, identifier.String(), tx.TxID)
			if err != nil {
				return fmt.Errorf("fail to disable root token: %v", err)
			}
			for i := 0; i < len(disabledTokens); i++ {
				if disabledTokens[i].Status == 3 {
					continue
				}
				err = database.InvalidSpentToken(disabledTokens[i].ID)
				if err != nil {
					return fmt.Errorf("fail to confirm spent token: %v", err)
				}
				fmt.Printf("🚫 Disable CT-AUT Root token for CT-AUT with identifier %s of account with account id %d: block id %s, block height %d, transacion id %s, index %d\n",
					identifier.String(), viewAccount.ID, blockID, blockHeight, disabledTokens[i].TxID, disabledTokens[i].Index)
			}
		}
	}

	return nil
}

func TrackCoins(tx *abelian.Tx) error {
	for index := 0; index < len(tx.Vin); index++ {
		mayConsumedCoins, err := database.LoadCoinBySerialNumber(tx.Vin[index].SerialNumber)
		if err != nil {
			panic(fmt.Errorf("fail to load coin by serial number from database: %v", err))
		}

		for _, coin := range mayConsumedCoins {
			coinID := coin.Coin.ID()
			for ringIndex := 0; ringIndex < len(tx.Vin[index].TXORing.OutPoints); ringIndex++ {
				if coinID.TxID == tx.Vin[index].TXORing.OutPoints[ringIndex].TxHash &&
					coinID.Index == tx.Vin[index].TXORing.OutPoints[ringIndex].Index {
					err = database.ConfirmSpentCoin(coin.ID)
					if err != nil {
						panic(fmt.Errorf("fail to consume coin: %v", err))
					}

					fmt.Printf("💸 Coin of account with account id %d is consumed: block id %s, block height %d, transacion id %s, index %d, value %v ABELs\n",
						coin.AccountID, coin.BlockHash, coin.BlockHeight, tx.TxHash, index, abelian.NeutrinoToAbel(coin.Value))

					token, err := database.LoadTokenByPoint(coin.AccountID, coin.TxID, coin.Index)
					if err != nil {
						panic(fmt.Errorf("fail to load token by point from database: %v", err))
					}
					if token != nil {
						err = database.ConfirmSpentToken(token.ID)
						if err != nil {
							panic(fmt.Errorf("fail to consume token: %v", err))
						}
						fmt.Printf("💸 CT-AUT token of account with account id %d is consumed: block id %s, block height %d, transacion id %s, index %d, value %v\n",
							coin.AccountID, coin.BlockHash, coin.BlockHeight, tx.TxHash, index, token.Value)
					}
					break
				}
			}

		}
	}
	return nil
}
func BuildCoinRingsWithBlockHeight(height int64) (map[abelian.CoinID]*abelian.CoinRing, error) {
	serializedBlockGroups, err := abelian.GetRingBlockGroupByHeight(client, height)
	if err != nil {
		return nil, fmt.Errorf("fail to get block group with height %d: %v", height, err)
	}

	coinIDRings, err := abelian.BuildCoinRings(serializedBlockGroups)
	if err != nil {
		return nil, fmt.Errorf("fail to build txo rings with height %d: %v", height, err)
	}
	// build mapping: coin id -> ring
	coinID2RingDetail := map[abelian.CoinID]*abelian.CoinRing{}
	for i := 0; i < len(coinIDRings); i++ {
		ring := coinIDRings[i]
		for j := 0; j < len(ring.CoinIDRing.CoinIDs); j++ {
			coinID2RingDetail[*ring.CoinIDRing.CoinIDs[j]] = ring
		}
	}
	return coinID2RingDetail, nil
}
func HandleCoinMaturity(height int64) error {
	// handle coinbase coin maturity
	fmt.Printf("handle coinbase maturity in block with height %d \n", height)
	immatureCoinbaseCoins, err := database.LoadImmatureCoinbaseCoins(height - abelian.GetCoinbaseMaturity())
	if err != nil {
		panic(fmt.Errorf("fail to load immature coinbase coins from database"))
	}
	for _, coin := range immatureCoinbaseCoins {
		fmt.Printf("🎉 coinbase coin (%s,%d) %v ABELs is mature\n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value))
		err = database.MatureCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mature immature coinbase coins"))
		}
	}

	// handle transfer coins maturity
	blockNum := int64(abelian.GetBlockNumPerRingGroupByBlockHeight(height))
	if height%blockNum != blockNum-1 {
		return nil
	}
	fmt.Printf("handle transfer maturity in block with height %d \n", height)

	// NOTE: the height that meets the above conditions should mature some coins, here show for specified height
	// how to handle maturity with ring
	coinID2CoinRing, err := BuildCoinRingsWithBlockHeight(height)
	if err != nil {
		return err
	}

	// in this point, some coins would be mature
	immatureCoins, err := database.LoadImmatureCoins(height)
	if err != nil {
		panic(fmt.Errorf("fail to load immature coins from database"))
	}
	for i := 0; i < len(immatureCoins); i++ {
		coin := immatureCoins[i]
		coinID := coin.Coin.ID()
		// Before mark coin mature, build rings from blocks ith height
		coinRing, ok := coinID2CoinRing[*coinID]
		if !ok {
			return fmt.Errorf("can not found ring detail for coin (%s)", coinID.String())
		}

		ringIndex := 0
		for ; ringIndex < len(coinRing.CoinIDRing.CoinIDs); ringIndex++ {
			if coinRing.CoinIDRing.CoinIDs[ringIndex].TxID == coinID.TxID &&
				coinRing.CoinIDRing.CoinIDs[ringIndex].Index == coinID.Index {
				break
			}
		}

		ringId, err := coinRing.RingId()
		if err != nil {
			panic(err)
		}
		coin.Coin.SetRingInfo(ringId, uint8(ringIndex))

		// generate serial number
		w := bytes.Buffer{}
		err = coinRing.Serialize(&w)
		if err != nil {
			panic(err)
		}
		serializedRing := w.Bytes()
		viewAccount, _ := database.LoadViewAccount(coin.AccountID)
		serialNumber, err := viewAccount.GenerateSerialNumberWithRing(
			&abelian.CoinID{
				TxID:  coin.TxID,
				Index: coin.Index,
			},
			serializedRing)
		if err != nil {
			panic(err)
		}

		// update the coin info
		err = database.UpdateCoinInfo(coin.ID, ringId, uint8(ringIndex), hex.EncodeToString(serialNumber))
		if err != nil {
			panic(fmt.Errorf("fail to update coins"))
		}

		// insert relevant coins which would be used when generating transaction to consume matured coins
		for j := 0; j < len(coinRing.CoinIDRing.CoinIDs); j++ {
			relevantCoinID := coinRing.CoinIDRing.CoinIDs[j]
			relevantSerializedTXO := coinRing.SerializedTxOuts[j]
			_, err = database.InsertRelevantCoin(0, coin.TxVersion, relevantCoinID.TxID, relevantCoinID.Index, "", 0, -1, coinRing.IsCoinbase, relevantSerializedTXO, ringId, uint8(j))
			if err != nil {
				panic(err)
			}
		}

		// insert ring which would be used when generating transaction to consume matured coins
		_, err = database.InsertRing(
			ringId,
			coinRing.Version,
			int64(coinRing.RingBlockHeight),
			coinRing.CoinIDRing.BlockIDs,
			int8(len(coinRing.CoinIDRing.CoinIDs)),
			coinRing.IsCoinbase)
		if err != nil {
			panic(err)
		}
		fmt.Printf("📢 Ring related for coin (%s,%d) %v ABELs is inserted\n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value))

		// lastly mark coin as mature
		err = database.MatureCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mature immature coins"))
		}
		fmt.Printf("🎉 transfer coin (%s,%d) %v ABELs mature with ring info (%s,%d) \n", coin.TxID, coin.Index, abelian.NeutrinoToAbel(coin.Value), ringId, ringIndex)

		token, err := database.LoadTokenByPoint(coin.AccountID, coin.TxID, coin.Index)
		if err != nil {
			panic(fmt.Errorf("fail to load token by point from database: %v", err))
		}
		if token != nil {
			err = database.MatureToken(token.ID)
			if err != nil {
				panic(fmt.Errorf("fail to mature token: %v", err))
			}
			fmt.Printf("🎈CT-AUT token of account with account id %d is matured: block id %s, block height %d, transacion id %s, index %d, value %v\n",
				coin.AccountID, coin.BlockHash, coin.BlockHeight, token.TxID, token.Index, token.Value)
		}
	}

	return nil
}

var client *abelian.Client

func init() {
	client = common.Client
}

func main() {
	// Load view accounts from database
	viewAccounts, err := database.LoadViewAccounts()
	if err != nil {
		panic(err)
	}

	if len(viewAccounts) == 0 {
		panic(fmt.Errorf("no view account found in database"))
	}

	// Scan and store coins for accounts with specified height ranges
	startScanHeight := common.GetCoinScanStartHeight()
	endScanHeight := common.GetCoinScanEndHeight()

	// scan blocks with specified height scope
	for currentHeight := startScanHeight; currentHeight < endScanHeight; currentHeight++ {
		block, err := client.GetBlockByHeight(currentHeight)
		if err != nil {
			panic(fmt.Errorf("fail to query block: %v", err))
		}

		fmt.Printf("Scan and track coins in block with height %d\n", block.Height)
		for i := 0; i < len(block.TxHashes); i++ {
			tx, err := client.GetRawTx(block.TxHashes[i])
			if err != nil {
				panic(fmt.Errorf("fail to query transaction: %v", err))
			}

			// scan coin in transaction
			err = ScanCoins(viewAccounts, tx, i == 0, block.BlockHash, block.Height)
			if err != nil {
				panic(fmt.Errorf("fail to scan tranaction:%v", err))
			}

			// track coin status
			// for coinbase transaction, there no valid consumption
			if i == 0 {
				continue
			}
			// but for transfer transaction, There must be valid consumption
			err = TrackCoins(tx)
			if err != nil {
				panic(fmt.Errorf("fail to track coin status :%v", err))
			}
		}

		err = HandleCoinMaturity(currentHeight)
		if err != nil {
			panic(fmt.Errorf("fail to hanle coin maturity :%v", err))
		}
	}
}
