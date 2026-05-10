package abelian

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/abesuite/abec/abecryptox/abecryptoxkey"
	"github.com/abesuite/abec/abecryptox/abecryptoxparam"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

func MessageSignatureSign(account *RootSeedAccount, abelAddress []byte, message []byte) ([]byte, error) {
	if account.privacyLevel == crypto.PrivacyLevelFullPrivacyPre || account.privacyLevel == crypto.PrivacyLevelFullPrivacyRand {
		return nil, errors.New("unsupported operation for full privacy account")
	}
	if len(message) == 0 {
		return nil, fmt.Errorf("MessageSignatureSign: the input message is nil/empty")
	}

	// parse public rand
	address, err := NewAbelAddress(abelAddress)
	if err != nil {
		return nil, err
	}
	cryptoAddress := address.GetCryptoAddress()
	if cryptoAddress.GetCryptoScheme() != crypto.CryptoSchemePQRingCTX {
		return nil, fmt.Errorf("unsupported operation for private crypto")
	}
	privacyLevel := cryptoAddress.GetPrivacyLevel()
	if account.privacyLevel != privacyLevel {
		return nil, fmt.Errorf("mismatched privacy level between account and address")
	}

	publicRand, err := crypto.ExtractPublicRandFromCryptoAddress(cryptoAddress)
	if err != nil {
		panic(fmt.Errorf("fail to extract public rand from crypto address: %v", err))
	}

	// derive crypto address and spend secret key
	pp := abecryptoxparam.PQRingCTXPP
	reGenCryptoAddress, cryptoSpendSecretKey, _, _, _, err :=
		abecryptoxkey.CryptoAddressKeyReGenByRootSeedsFromPublicRand(account.cryptoScheme, account.privacyLevel,
			account.coinSpendKeySeed, account.coinSerialNumberKeySeed, account.coinValueKeySeed,
			account.coinDetectorKey, publicRand)
	if err != nil {
		return nil, err
	}
	// check the ownership
	if !bytes.Equal(cryptoAddress.Data(), reGenCryptoAddress) {
		return nil, errors.New("invalid address")
	}

	// message
	message = append(message, crypto.SerializeCryptoScheme(account.cryptoScheme)...)
	message = append(message, byte(account.privacyLevel))

	// make message signature
	messageSig, err := pp.MessageSignatureSign(message[:], cryptoSpendSecretKey)
	if err != nil {
		return nil, err
	}

	signature, err := pp.SerializeMessageSignature(messageSig)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func MessageSignatureVerify(abelAddress []byte, message []byte, signature []byte) error {
	if len(abelAddress) == 0 {
		return fmt.Errorf("MessageSignatureVerify: the input abelAddress is nil/empty")
	}
	if len(message) == 0 {
		return fmt.Errorf("MessageSignatureVerify: the input message is nil/empty")
	}
	if len(signature) == 0 {
		return fmt.Errorf("MessageSignatureVerify: the input signature is nil/empty")
	}

	address, err := NewAbelAddress(abelAddress)
	if err != nil {
		return errors.New("MessageSignatureVerify: invalid address")
	}
	cryptoAddress := address.GetCryptoAddress()
	coinAddress := cryptoAddress.GetCoinAddress()

	pp := abecryptoxparam.PQRingCTXPP
	messageSignature, err := pp.DeserializeMessageSignature(signature)
	if err != nil {
		return fmt.Errorf("MessageSignatureVerify: invalid signature %v", err)
	}

	err = pp.MessageSignatureMatch(messageSignature, coinAddress.Data())
	if err != nil {
		return fmt.Errorf("MessageSignatureVerify: mismatched signature and address %v", err)
	}

	message = append(message, crypto.SerializeCryptoScheme(cryptoAddress.GetCryptoScheme())...)
	message = append(message, byte(cryptoAddress.GetPrivacyLevel()))
	return pp.MessageSignatureVerify(message, messageSignature)
}
