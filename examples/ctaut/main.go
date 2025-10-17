package main

import (
	"encoding/hex"
	"fmt"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

var client *abelian.Client

func init() {
	client = common.Client
}
func SignRawTransactionForCTAUT(unsignedRawTx *abelian.UnsignedRawTx, senderAccountIDs []int64) (*abelian.SignedRawTx, error) {
	// Load account (NOT view account!!!)
	senderAccounts := make([]abelian.Account, 0, len(senderAccountIDs))
	for _, accountID := range senderAccountIDs {
		account, err := database.LoadAccountByID(accountID)
		if err != nil {
			panic(fmt.Errorf("fail to load account: %v", err))
		}
		senderAccounts = append(senderAccounts, account.Account)
	}

	// Sign the unsigned transaction
	return abelian.GenerateSignedRawTxForCTAUT(unsignedRawTx, senderAccounts)
}
func main() {
	// 1. register
	//registerCTAUT()

	// 2. reregister
	//identifier, err := abelian.CTAUTIdentifierKey("e93b0f0d04415e76fd27f95b49b6b9e394883f8154777f5aa63e84f2ceef51bf")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("identifier: ", hex.EncodeToString(identifier[:]))
	//reRegisterCTAUT(identifier, 1)

	// 3. mint
	//identifier, err := abelian.CTAUTIdentifierKey("e93b0f0d04415e76fd27f95b49b6b9e394883f8154777f5aa63e84f2ceef51bf")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("identifier: ", hex.EncodeToString(identifier[:]))
	//mintCTAUT(identifier, 1)

	// 4. transfer
	//identifier, err := abelian.CTAUTIdentifierKey("e93b0f0d04415e76fd27f95b49b6b9e394883f8154777f5aa63e84f2ceef51bf")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("identifier: ", hex.EncodeToString(identifier[:]))
	//transferCTAUT(identifier)

	// 5. burn
	identifier, err := abelian.CTAUTIdentifierKey("e93b0f0d04415e76fd27f95b49b6b9e394883f8154777f5aa63e84f2ceef51bf")
	if err != nil {
		panic(err)
	}
	fmt.Println("identifier: ", hex.EncodeToString(identifier[:]))
	burnCTAUT(identifier)
}
