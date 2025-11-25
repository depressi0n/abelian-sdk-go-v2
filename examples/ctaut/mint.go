package main

import (
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func mintCTAUT(identifier abelian.AutId, mintThreshold uint8) {
	pseudonymCTAccount, err := database.LoadAccountByID(5)
	if err != nil {
		panic("fail to load account with id 5")
	}

	// And then generated address
	fmt.Println("generating fiona")
	fionaAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}
	fmt.Println("generating galileo")
	galileoAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}

	fmt.Println("generating change")
	changeAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}

	receiverAddresses := [][]byte{
		fionaAbelAddress,
		galileoAbelAddress,
	}
	recipientValues := []uint64{3000, 2000}

	// Load root tokens of specified account
	selectAccountIDs := []int64{5}
	rootTokens, err := database.LoadCTAUTTokenByAccountID(selectAccountIDs[0], identifier.String(), true)
	if err != nil {
		panic(err)
	}
	if len(rootTokens) == 0 {
		panic("no root token found")
	}

	// Note that here the coin which hosts the CT-AUT token must be meet the CT-AUT instance claimed threshold
	// filter as your self-defined rules: choose some issuers' root token
	// for simplicity, we just select the first reRegisterThreshold root tokens
	filterRootTokens := make([]*database.Token, 0)
	for i := 0; i < int(mintThreshold); i++ {
		// select root token from different issuers
		filterRootTokens = append(filterRootTokens, rootTokens[i])
	}

	vin := uint64(0)
	recipients := make([]*abelian.Recipient, 0, len(receiverAddresses))
	for i := 0; i < len(receiverAddresses); i++ {
		abelAddress, _ := abelian.NewAbelAddress(receiverAddresses[i])

		recipients = append(recipients, &abelian.Recipient{
			CryptoAddress: *abelAddress.GetCryptoAddress(),
			Value:         recipientValues[i],
			HideValue:     true,
		})
		vin += recipientValues[i]
	}
	// [IMPORTANT] sort the CT-AUT tokens: CT-Token > Plain Token
	// For simplicity, we make it already for output

	txMemo, autWitness, err := abelian.CreateCTAUTMintScript(
		abelian.AutScriptVersion,
		identifier,
		vin,
		uint8(len(filterRootTokens)),
		recipients,
		[]byte("first mint for fiona and galileo"),
	)
	if err != nil {
		panic(err)
	}

	// set the host outpoint for each root token
	targetAmount := int64(0)
	txOutDescs := make([]*abelian.TxOutDesc, 0, len(receiverAddresses))
	for _, abelAddress := range receiverAddresses {
		address, _ := abelian.NewAbelAddress(abelAddress)
		txOutDescs = append(txOutDescs, &abelian.TxOutDesc{
			AbelAddress: address,
			CoinValue:   1,
		})
		targetAmount += 1
	}

	selectAmount := int64(0)
	selectedCoins := []*database.Coin{}

	// load coins for selected root tokens rootTokens[:]
	for _, token := range filterRootTokens {
		coin, err := database.LoadCoinByPoint(token.AccountID, token.TxID, token.Index)
		if err != nil {
			panic(err)
		}
		selectedCoins = append(selectedCoins, coin)
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

	for i := 0; i < len(availableCoins); i++ {
		// skip the special value
		if availableCoins[i].Value == 1 {
			continue
		}
		if selectAmount >= targetAmount {
			break
		}

		selectedCoins = append(selectedCoins, availableCoins[i])
		selectAmount += availableCoins[i].Value
	}

	// Build TxInDesc
	txInDescs := []*abelian.TxInDescWithRing{}
	coin2AccountID := map[string]int64{}
	for _, coin := range selectedCoins {
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
		txInDescs = append(txInDescs, txIndesc)
		coin2AccountID[coin.Coin.ID().String()] = coin.AccountID
	}

	// [IMPORTANT] Sort TxInDesc
	err = abelian.SortTxInDescWithRing(txInDescs)
	if err != nil {
		panic(err)
	}

	// set sender account ID for signing
	senderAccountIDs := make([]int64, 0, len(txInDescs))
	for _, desc := range txInDescs {
		senderAccountIDs = append(senderAccountIDs, coin2AccountID[abelian.NewCoinID(desc.TxID, desc.TxOutIndex).String()])
	}

	// Estimated fee
	estimatedTxFee := abelian.EstimateTxFee(txInDescs, txOutDescs)

	// change if needed
	if selectAmount-targetAmount-estimatedTxFee > 0 {
		changeAbelAddress, err := abelian.NewAbelAddress(changeAddress)
		if err != nil {
			panic("invalid abel address")
		}
		if changeAbelAddress.GetNetID() != common.GetNetworkID() {
			panic("change address with unmatched network id")
		}

		txOutDescs = append(txOutDescs, &abelian.TxOutDesc{
			AbelAddress: changeAbelAddress,
			CoinValue:   selectAmount - targetAmount - estimatedTxFee,
		})
	}

	// [IMPORTANT] sort txOutDescs
	err = abelian.SortTxOutDesc(txOutDescs)
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
	for _, coin := range selectedCoins {
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
