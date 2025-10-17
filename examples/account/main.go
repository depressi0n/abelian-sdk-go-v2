package main

import (
	"fmt"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/database"
)

func main() {
	networkID := common.GetNetworkID()

	privacyLevels := []abelian.AccountPrivacyLevel{
		abelian.AccountPrivacyLevelFullPrivacy,
		abelian.AccountPrivacyLevelPseudonym,
		abelian.AccountPrivacyLevelPseudonymCT,
	}

	for _, privacyLevel := range privacyLevels {
		account, err := abelian.NewAccount(networkID, privacyLevel)
		if err != nil {
			panic(fmt.Errorf("fail to generate account:%v", err))
		}
		fmt.Printf("%+v\n", account)

		spendKey := account.SpendKeyMaterial()
		snKeySeed, valueKeySeed, detectKey := account.ViewKeyMaterial()
		accountID, err := database.InsertAccount(networkID, privacyLevel, spendKey, snKeySeed, valueKeySeed, detectKey)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%x\n", accountID)
		for i := 0; i < len(spendKey); i++ {
			fmt.Printf("%#02x, ", spendKey[i])
		}
		fmt.Println()

		for i := 0; i < len(snKeySeed); i++ {
			fmt.Printf("%#02x, ", snKeySeed[i])
		}
		fmt.Println()

		for i := 0; i < len(valueKeySeed); i++ {
			fmt.Printf("%#02x, ", valueKeySeed[i])
		}
		fmt.Println()

		for i := 0; i < len(detectKey); i++ {
			fmt.Printf("%#02x, ", detectKey[i])
		}
		fmt.Println()

		loadedAccount, err := database.LoadAccountByID(accountID)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", loadedAccount)

		address, err := loadedAccount.GenerateAbelAddress()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%x\n", address)
	}
}
