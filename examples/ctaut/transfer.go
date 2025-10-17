package main

import (
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func transferCTAUT(identifier [abelian.CTAUTIdentifierLength]byte) {
	pseudonymCTAccount, err := database.LoadAccountByID(5)
	if err != nil {
		panic("fail to load account with id 5")
	}

	// And then generated address
	haloAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}
	innaAbelAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}
	changeAddress, err := pseudonymCTAccount.GenerateAbelAddress()
	if err != nil {
		panic("fail to generated change address for account")
	}

	receiverAddresses := [][]byte{
		haloAbelAddress,
		innaAbelAddress,
	}
	recipientValues := []uint64{2000, 500}

	targetTokenValue := uint64(0)
	recipients := make([]*abelian.Recipient, 0, len(receiverAddresses))
	for i := 0; i < len(receiverAddresses); i++ {
		abelAddress, _ := abelian.NewAbelAddress(receiverAddresses[i])

		recipients = append(recipients, &abelian.Recipient{
			CryptoAddress: *abelAddress.GetCryptoAddress(),
			Value:         recipientValues[i],
			HideValue:     true,
		})
		targetTokenValue += recipientValues[i]
	}

	// Load CT-AUT Tokens of specified account
	selectAccountIDs := []int64{5}
	tokens, err := database.LoadCTAUTTokenByAccountID(selectAccountIDs[0], hex.EncodeToString(identifier[:]), false)
	if err != nil {
		panic(err)
	}

	selectedTokens := make([]*database.Token, 0, len(tokens))
	selectedValue := uint64(0)
	for _, token := range tokens {
		if selectedValue >= targetTokenValue {
			break
		}

		selectedTokens = append(selectedTokens, token)
		selectedValue += token.Value
	}
	if selectedValue < targetTokenValue {
		panic("CT-Token token value is not enough for transfer")
	}
	if selectedValue-targetTokenValue != 0 {
		change, _ := abelian.NewAbelAddress(changeAddress)

		receiverAddresses = append(receiverAddresses, changeAddress)
		recipients = append(recipients, &abelian.Recipient{
			CryptoAddress: *change.GetCryptoAddress(),
			Value:         selectedValue - targetTokenValue,
			HideValue:     false,
		})
	}

	// [IMPORTANT] sort the CT-AUT input/output tokens: CT-Token > Plain Token
	// For simplicity, we make it already for output
	sort.SliceStable(selectedTokens, func(i, j int) bool {
		if selectedTokens[i].TokenType == selectedTokens[j].TokenType {
			return false
		}

		if selectedTokens[i].TokenType == 0 { // hidden
			return true
		}
		if selectedTokens[j].TokenType == 0 { // hidden
			return false
		}
		return false
	})

	consumedTokens := make([]*abelian.InputTokenDesc, len(selectedTokens))
	for i, token := range selectedTokens {
		consumedTokens[i] = abelian.NewInputTokenDesc(
			token.Version,
			token.ValueScript,
			token.CryptoValuePK,
			token.CryptoValueSK,
			token.Value,
		)
	}

	txMemo, autWitness, err := abelian.CreateCTAUTTransferScript(
		identifier,
		consumedTokens,
		recipients,
		[]byte("first transfer to for halo and inna"),
	)
	if err != nil {
		panic(err)
	}

	// Abelian Layer

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
	// load coins for selected CT-AUT tokens
	for _, token := range selectedTokens {
		coin, err := database.LoadCoinByPoint(token.AccountID, token.TxID, token.Index)
		if err != nil {
			panic(err)
		}
		selectedCoins = append(selectedCoins, coin)
		selectAmount += coin.Value
	}
	// provide transaction fee from Abelian Coins
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
		if selectAmount > targetAmount {
			break
		}

		selectedCoins = append(selectedCoins, availableCoins[i])
		selectAmount += availableCoins[i].Value
	}
	if selectAmount < targetAmount {
		panic("not enough coin to pay for transaction")
	}

	// Build TxInDesc with selectedCoins
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

		fmt.Printf("%#+v", txIndesc)
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
	fmt.Println(unsignedRawTx)

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
	for _, token := range selectedTokens {
		err = database.SpendToken(token.ID)
		if err != nil {
			panic(fmt.Errorf("fail to mark token spent: %v", err))
		}
	}
}
