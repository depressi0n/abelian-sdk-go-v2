package main

import (
	"encoding/hex"
	"fmt"
	"slices"
	"sort"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func reRegisterCTAUT(identifier abelian.AutId) {
	// here get aut medata for threshold
	metadata, err := database.LoadCTAUTMetadata(identifier.String())
	if err != nil {
		panic(err)
	}

	pseudonymCTAccount, err := database.LoadAccountByID(5)
	if err != nil {
		panic("fail to load account with id 5")
	}

	// And then generated address
	charlieAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}
	diorAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}
	emilyAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}

	changeAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}

	issuerAbelAddresses := [][]byte{
		charlieAbelAddress,
		diorAbelAddress,
		emilyAbelAddress,
	}
	rootTokenNum := 3

	// Load coins of specified account
	selectAccountIDs := []int64{5}
	rootTokens, err := database.LoadCTAUTTokenByAccountID(selectAccountIDs[0], identifier.String(), true)
	if err != nil {
		panic(err)
	}
	if len(rootTokens) == 0 {
		panic("no root token found")
	}

	// Note that here the coin which hosts the CT-AUT token must be meet the CT-AUT instance claimed threshold
	// filter as your self-defined rules
	// for simplicity, we just select the first reRegisterThreshold root tokens
	filterRootTokens := make([]*database.Token, 0)
	for i := 0; i < int(metadata.ReregistrationThreshold); i++ {
		// select root token from different issuerCoinAddresses
		filterRootTokens = append(filterRootTokens, rootTokens[i])
	}
	// CT-AUT script: emily and fiona would be the issuer of CT-AUT instance
	// it would generate 4 root tokens,
	// two of them would be planned to mint
	// two of them would be planned to re-register
	issuerCoinAddresses := make([][]byte, 0, len(issuerAbelAddresses))
	for _, abelAddress := range issuerAbelAddresses {
		address, _ := abelian.NewAbelAddress(abelAddress)
		issuerCoinAddresses = append(issuerCoinAddresses, address.GetCryptoAddress().GetCoinAddress().Data())
	}

	// set the host outpoint for each root token
	targetAmount := int64(0)
	// Note that there are 3 types of txOutDescs
	txOutDescsForFully := []*abelian.TxOutDesc{}
	txOutDescsForPseudo := []*abelian.TxOutDesc{}
	txOutDescsForPseudoCT := []*abelian.TxOutDesc{}
	// AUT (root) tokens would be hosted on pseudoCT-privacy
	txOutDescsForAUT := make([]*abelian.TxOutDesc, 0, len(issuerAbelAddresses)*rootTokenNum)
	for _, abelAddress := range issuerAbelAddresses {
		address, _ := abelian.NewAbelAddress(abelAddress)

		for j := 0; j < rootTokenNum; j++ {
			txOutDescsForAUT = append(txOutDescsForAUT, &abelian.TxOutDesc{
				AbelAddress: address,
				CoinValue:   1,
			})
			targetAmount += 1
		}
	}

	selectAmount := int64(0)
	selectedCoinsForAUT := []*database.Coin{}

	// load coins for consumed root tokens
	for _, token := range filterRootTokens {
		coin, err := database.LoadCoinByPoint(token.AccountID, token.TxID, token.Index)
		if err != nil {
			panic(err)
		}
		selectedCoinsForAUT = append(selectedCoinsForAUT, coin)
		selectAmount += coin.Value
	}

	availableCoins := []*database.Coin{}
	for _, accountID := range selectAccountIDs {
		coins, err := database.LoadCoinByAccountID(accountID)
		if err != nil {
			return
		}
		for _, coin := range coins {
			if coin.Value == 1 {
				continue
			}
			availableCoins = append(availableCoins, coin)
		}
	}
	if len(availableCoins) == 0 {
		panic(fmt.Errorf("find no coin to spend"))
	}

	// Customized filters to select coins
	// Note that the coin which hosts the CT-AUT token must appear in the front
	sort.SliceStable(availableCoins, func(i, j int) bool {
		// Some custom logic would be applied, such as store the coins of parasitic CT-AUT separately
		return availableCoins[i].Value > availableCoins[j].Value
	})

	// Select coins for paying transaction fee
	selectedCoinsForABEL := []*database.Coin{}
	for i := 0; i < len(availableCoins); i++ {
		if selectAmount > targetAmount {
			break
		}
		// skip the special value to avoid unconscious destruction
		if availableCoins[i].Value == 1 {
			continue
		}

		selectedCoinsForABEL = append(selectedCoinsForABEL, availableCoins[i])
		selectAmount += availableCoins[i].Value
	}

	// Build TxInDesc
	txInDescsForAUT := []*abelian.TxInDescWithRing{}
	coin2AccountID := map[string]int64{}
	for _, coin := range selectedCoinsForAUT {
		ring, err := database.LoadRing(coin.RingID)
		if err != nil {
			panic(err)
		}
		coinIDs := make([]*abelian.CoinID, len(ring.Coins))
		serializedTxOuts := make([][]byte, len(ring.Coins))
		for i := 0; i < len(ring.Coins); i++ {
			coinIDs[i] = ring.Coins[i].ID()
			serializedTxOuts[i] = ring.Coins[i].TxVoutData
		}
		ringDetail, err := abelian.NewCoinRing(ring.RingVersion, ring.RingHeight, ring.RingBlockIDs, coinIDs, serializedTxOuts, ring.IsCoinbase)
		if err != nil {
			panic(err)
		}

		txIndesc := &abelian.TxInDescWithRing{
			BlockHeight: coin.BlockHeight,
			BlockID:     coin.BlockHash,
			TxVersion:   coin.TxVersion,
			TxID:        coin.TxID,
			TxOutIndex:  coin.Index,
			TxOutData:   coin.TxVoutData,
			CoinValue:   coin.Value,
			TxoRing:     ringDetail,
		}

		//fmt.Printf("%#+v", txIndesc)
		txInDescsForAUT = append(txInDescsForAUT, txIndesc)
		coin2AccountID[coin.Coin.ID().String()] = coin.AccountID
	}

	// Build TxInDesc
	txInDescsForFully := []*abelian.TxInDescWithRing{}
	txInDescsForPseudoCT := []*abelian.TxInDescWithRing{}
	txInDescsForPseudo := []*abelian.TxInDescWithRing{}
	for _, coin := range selectedCoinsForABEL {
		ring, err := database.LoadRing(coin.RingID)
		if err != nil {
			panic(err)
		}
		coinIDs := make([]*abelian.CoinID, len(ring.Coins))
		serializedTxOuts := make([][]byte, len(ring.Coins))
		for i := 0; i < len(ring.Coins); i++ {
			coinIDs[i] = ring.Coins[i].ID()
			serializedTxOuts[i] = ring.Coins[i].TxVoutData
		}
		ringDetail, err := abelian.NewCoinRing(ring.RingVersion, ring.RingHeight, ring.RingBlockIDs, coinIDs, serializedTxOuts, ring.IsCoinbase)
		if err != nil {
			panic(err)
		}

		txIndesc := &abelian.TxInDescWithRing{
			BlockHeight: coin.BlockHeight,
			BlockID:     coin.BlockHash,
			TxVersion:   coin.TxVersion,
			TxID:        coin.TxID,
			TxOutIndex:  coin.Index,
			TxOutData:   coin.TxVoutData,
			CoinValue:   coin.Value,
			TxoRing:     ringDetail,
		}

		// get the privacy level of the coin
		coinAddress, err := crypto.DecodeCoinAddressFromSerializedTxOutData(coin.TxVersion, coin.TxVoutData)
		if err != nil {
			panic(err)
		}
		privacyLevel := coinAddress.PrivacyLevel()

		switch privacyLevel {
		case crypto.PrivacyLevelFullPrivacyPre, crypto.PrivacyLevelFullPrivacyRand:
			txInDescsForFully = append(txInDescsForFully, txIndesc)
			break
		case crypto.PrivacyLevelPseudonymCT:
			txInDescsForPseudoCT = append(txInDescsForPseudoCT, txIndesc)
			break
		case crypto.PrivacyLevelPseudonym:
			txInDescsForPseudo = append(txInDescsForPseudo, txIndesc)
			break
		default:
			panic(fmt.Errorf("unsupported privacy level %d", privacyLevel))
		}

		//fmt.Printf("%#+v", txIndesc)
		coin2AccountID[coin.Coin.ID().String()] = coin.AccountID
	}

	// [IMPORTANT] Sort TxInDesc
	txInDescs := make([]*abelian.TxInDescWithRing, 0, len(txInDescsForFully)+len(txInDescsForPseudoCT)+len(txInDescsForPseudo))
	txInDescs = append(txInDescs, txInDescsForFully...)
	txInDescs = append(txInDescs, txInDescsForPseudoCT...)
	txInDescs = append(txInDescs, txInDescsForPseudo...)
	// Note that the inner order is kept while sorting
	err = abelian.SortTxInDescWithRing(txInDescs)
	if err != nil {
		panic(err)
	}

	// Then find the index for txOutDescsForAUT
	inStartIndex := uint8(len(txInDescsForFully))
	txInDescs = slices.Insert(txInDescs, len(txInDescsForFully), txInDescsForAUT...)

	// set sender account ID for signing
	senderAccountIDs := make([]int64, 0, len(txInDescs))
	for _, desc := range txInDescs {
		senderAccountIDs = append(senderAccountIDs, coin2AccountID[abelian.NewCoinID(desc.TxID, desc.TxOutIndex).String()])
	}

	// Estimated fee
	tmpTxOutDescs := make([]*abelian.TxOutDesc, 0, len(txOutDescsForFully)+len(txOutDescsForPseudoCT)+len(txOutDescsForPseudo)+len(txOutDescsForAUT))
	tmpTxOutDescs = append(tmpTxOutDescs, txOutDescsForFully...)
	tmpTxOutDescs = append(tmpTxOutDescs, txOutDescsForPseudoCT...)
	tmpTxOutDescs = append(tmpTxOutDescs, txOutDescsForAUT...)
	tmpTxOutDescs = append(tmpTxOutDescs, txOutDescsForPseudo...)
	estimatedTxFee := abelian.EstimateTxFee(txInDescs, tmpTxOutDescs)

	// change if needed
	if selectAmount-targetAmount-estimatedTxFee > 0 {
		changeAbelAddress, err := abelian.NewAbelAddress(changeAddress)
		if err != nil {
			panic("invalid abel address")
		}
		if changeAbelAddress.GetNetID() != common.GetNetworkID() {
			panic("change address with unmatched network id")
		}

		// Note that the change will be a fully-privacy output according to the above configuration
		txOutDescsForPseudoCT = append(txOutDescsForPseudoCT, &abelian.TxOutDesc{
			AbelAddress: changeAbelAddress,
			CoinValue:   selectAmount - targetAmount - estimatedTxFee,
		})
	}

	// [IMPORTANT] sort txOutDescs
	txOutDescs := make([]*abelian.TxOutDesc, 0, len(txOutDescsForFully)+len(txOutDescsForPseudoCT)+len(txOutDescsForPseudo))
	txOutDescs = append(txOutDescs, txOutDescsForFully...)
	txOutDescs = append(txOutDescs, txOutDescsForPseudoCT...)
	txOutDescs = append(txOutDescs, txOutDescsForPseudo...)
	// [IMPORTANT] sort txOutDescs, note that the order should be
	// 1. fully-privacy / 2. pseudoCT-privacy / 3. pseudo-privacy
	// 2. the inner order would be kept while sorting
	err = abelian.SortTxOutDesc(txOutDescs)
	if err != nil {
		panic(err)
	}

	// Find the outStartIndex for txOutDescsForAUT
	outStartIndex := uint8(len(txOutDescsForFully))
	txOutDescs = slices.Insert(txOutDescs, len(txOutDescsForFully), txOutDescsForAUT...)

	// change the privacy type
	// 0 - unlimited, 1 - limited public, 2 - limited hidden
	privacyType := abelian.AutPrivacyTypeLimitedHidden

	txMemo, autWitness, err := abelian.CreateCTAUTReRegisterScript(
		abelian.AutScriptVersion,
		identifier,
		[]byte("Post-Quantum USD on the world"), uint64(1)<<51-1,
		issuerCoinAddresses, 800_000, // update the expired block
		1,
		1,
		privacyType,
		inStartIndex, uint8(len(txInDescsForAUT)),
		outStartIndex, uint8(len(txOutDescsForAUT)),
		[]byte("The first re-register of the first Post-Quantum USD on the world"),
	)
	if err != nil {
		panic(err)
	}

	//  Make an unsigned transaction
	txDesc := abelian.NewTxDescWithRingWithMemo(txInDescs, txOutDescs, estimatedTxFee, txMemo)
	unsignedRawTx, err := abelian.GenerateUnsignedRawTxWithRing(txDesc)
	if err != nil {
		panic(fmt.Errorf("fail to generate unsigned raw tx: %v", err))
	}
	unsignedRawTx.AutWitness = autWitness
	//fmt.Println(unsignedRawTx)

	// Sign transaction
	signedRawTx, err := SignRawTransactionForCTAUT(unsignedRawTx, senderAccountIDs)
	if err != nil {
		panic(err)
	}
	// Broadcast signed transaction
	returnedTxHash, err := client.SendRawTx(hex.EncodeToString(signedRawTx.Data))
	if err != nil {
		panic(fmt.Errorf("fail to send raw tx: %v", err))
	}

	// assert equal
	if returnedTxHash != signedRawTx.TxID {
		panic(fmt.Errorf("unmatched tx id"))
	}
	fmt.Println("Submit transaction: ", returnedTxHash)

	_, err = database.InsertTx(returnedTxHash, 0, hex.EncodeToString(unsignedRawTx.Data), senderAccountIDs, hex.EncodeToString(signedRawTx.Data))
	if err != nil {
		panic(err)
	}

	// mark coin spent
	for _, coin := range selectedCoinsForABEL {
		err = database.SpendCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mark coin spent: %v", err))
		}
	}
	for _, coin := range selectedCoinsForAUT {
		err = database.SpendCoin(coin.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mark coin spent: %v", err))
		}
	}
	// mark token spent
	for _, token := range filterRootTokens {
		err = database.SpendToken(token.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mark token spent: %v", err))
		}
	}
}
