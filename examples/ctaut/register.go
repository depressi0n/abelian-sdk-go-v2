package main

import (
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func registerCTAUT() {
	accountForPseudonymousCT, err := database.LoadAccountByID(5)
	if err != nil {
		panic("fail to load account with id 5")
	}

	aliceAbelAddress, err := accountForPseudonymousCT.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account with account")
	}
	bobAbelAddress, err := accountForPseudonymousCT.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account with account")
	}

	changeAddress, err := accountForPseudonymousCT.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account with id 5")
	}

	issuerAbelAddresses := [][]byte{
		aliceAbelAddress,
		bobAbelAddress,
	}
	rootTokenNum := 2

	// [IMPORTANT] CT-AUT script: bob and cherry would be the issuer, and the follow script would generate 4 root tokens,
	// Both bob and cherry have 2 root tokens.
	// And the plan after that would be
	// 1. For the next mint script, Bob and Cherry will each use 1 root token.
	// 2. For the next re-register script, Bob and Cherry also will each use 1 root token.
	issuerTokens := make([][]byte, 0, len(issuerAbelAddresses))
	for _, abelAddress := range issuerAbelAddresses {
		address, _ := abelian.NewAbelAddress(abelAddress)
		issuerTokens = append(issuerTokens, address.GetCryptoAddress().GetCoinAddress().Data())
	}

	txMemo, autWitness, err := abelian.CreateCTAUTRegisterScript(
		[]byte("Post-Quantum USD"),
		[]byte("PQUSD"),
		[]byte("USD"),
		[]byte("Cent"),
		100,
		[]byte("Post-Quantum USD on the world"),
		uint64(1)<<51-1, // the maximum amount
		issuerTokens,
		1, // it means that bob and cherry must cooperate to mint
		1, // it means that bob and cherry must cooperate to re-register
		600_000,
		uint8(len(issuerAbelAddresses)*rootTokenNum),
		[]byte("The first registration of Post-Quantum USD on the world"),
	)
	if err != nil {
		panic(err)
	}

	// set the host outpoint for each root token
	targetAmount := int64(0)
	txOutDescs := make([]*abelian.TxOutDesc, 0, len(issuerAbelAddresses)*rootTokenNum)
	for _, abelAddress := range issuerAbelAddresses {
		address, _ := abelian.NewAbelAddress(abelAddress)

		for j := 0; j < rootTokenNum; j++ {
			txOutDescs = append(txOutDescs, &abelian.TxOutDesc{
				AbelAddress: address,
				CoinValue:   1,
			})
			targetAmount += 1
		}
	}
	// Load coins of specified account
	selectAccountIDs := []int64{5}
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
	sort.SliceStable(availableCoins, func(i, j int) bool {
		return availableCoins[i].Value > availableCoins[j].Value
	})

	selectAmount := int64(0)
	selectedCoins := []*database.Coin{}
	for i := 0; i < len(availableCoins); i++ {
		if selectAmount > targetAmount {
			break
		}
		// skip the special value to avoid unconscious destruction
		if availableCoins[i].Value == 1 {
			continue
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

	registeredIdentifier, err := abelian.CTAUTIdentifierKey(signedRawTx.TxID)
	if err != nil {
		panic(err)
	}

	// record the identifier for later re-register/mint/transfer/burn
	fmt.Println("identifier: ", hex.EncodeToString(registeredIdentifier[:]))
}
