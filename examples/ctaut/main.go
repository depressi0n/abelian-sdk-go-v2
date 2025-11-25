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
	identifier, err := abelian.NewAutId("0c0f58a00e08c6b2efb849dca83f59e22be4986f26adc4a1a4f4b921eb0abe25")
	if err != nil {
		panic(err)
	}
	fmt.Println("identifier: ", identifier.String())

	// 2. reregister
	//reRegisterCTAUT(identifier, 1)

	// 3. mint
	//mintCTAUT(identifier, 1)

	// 4. transfer
	//transferCTAUT(identifier)

	// 5. burn
	//burnCTAUT(identifier)
}
