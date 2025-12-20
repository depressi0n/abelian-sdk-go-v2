package main

import (
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
	registerCTAUT()

	// fill in the txid
	//identifier, err := abelian.NewAutId("a65d71c793b27a5de0f884d446d1a5443b60a77e81d4d037815bea16a87288a5")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("identifier: ", identifier.String())

	// 2. reregister
	//reRegisterCTAUT(identifier)

	// 3. mint
	//mintCTAUT(identifier)

	// 4. transfer
	//transferCTAUT(identifier)

	// 5. burn
	//burnCTAUT(identifier)
}
