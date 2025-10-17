package abelian

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/abesuite/abec/ctaut"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

type AccountPrivacyLevel int

const (
	AccountPrivacyLevelFullPrivacyOld AccountPrivacyLevel = 0
	AccountPrivacyLevelFullPrivacy    AccountPrivacyLevel = 1
	AccountPrivacyLevelPseudonym      AccountPrivacyLevel = 2
	AccountPrivacyLevelPseudonymCT    AccountPrivacyLevel = 3
)

func NewAccount(networkID NetworkID, accountPrivacyLevel AccountPrivacyLevel) (Account, error) {
	cryptoScheme, privacyLevel := getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel)
	switch accountPrivacyLevel {
	case AccountPrivacyLevelFullPrivacyOld:
		randSeeds, err := crypto.GenerateSeed(cryptoScheme, privacyLevel)
		if err != nil {
			return nil, fmt.Errorf("fail to genereate seed for account")
		}
		keysAndAddressByRandSeeds, err := crypto.GenerateCryptoKeysAndAddressByRandSeeds(randSeeds)
		if err != nil {
			return nil, fmt.Errorf("fail to genereate keys and address")
		}
		account := NewCryptoKeysAccount(
			networkID,
			accountPrivacyLevel,
			keysAndAddressByRandSeeds.SpendSecretKey,
			keysAndAddressByRandSeeds.SerialNoSecretKey,
			keysAndAddressByRandSeeds.ViewSecretKey,
			keysAndAddressByRandSeeds.DetectorKey,
			keysAndAddressByRandSeeds.CryptoAddress,
		)
		return account, err
	case AccountPrivacyLevelFullPrivacy:
		// nothing to do
	case AccountPrivacyLevelPseudonym:
		// nothing to do
	case AccountPrivacyLevelPseudonymCT:
		// nothing to do
	default:
		return nil, fmt.Errorf("invalid privacy level for account")
	}
	// AccountPrivacyLevelFullPrivacy or AccountPrivacyLevelPseudonym
	rootSeeds, err := crypto.GenerateSeed(cryptoScheme, privacyLevel)
	if err != nil {
		return nil, fmt.Errorf("fail to genereate seed for account")
	}

	account := NewRootSeedAccount(
		networkID,
		accountPrivacyLevel,
		rootSeeds.CoinSpendKeySeed(),
		rootSeeds.CoinSerialNumberKeySeed(),
		rootSeeds.CoinValueKeySeed(),
		rootSeeds.CoinDetectorKey(),
	)
	return account, err
}

func getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel AccountPrivacyLevel) (crypto.CryptoScheme, crypto.PrivacyLevel) {
	cryptoScheme := crypto.CryptoSchemePQRingCTX
	privacyLevel := crypto.PrivacyLevelFullPrivacyRand
	switch accountPrivacyLevel {
	case AccountPrivacyLevelFullPrivacyOld:
		cryptoScheme = crypto.CryptoSchemePQRingCT
		privacyLevel = crypto.PrivacyLevelFullPrivacyPre
		break
	case AccountPrivacyLevelFullPrivacy:
		cryptoScheme = crypto.CryptoSchemePQRingCTX
		privacyLevel = crypto.PrivacyLevelFullPrivacyRand
		break
	case AccountPrivacyLevelPseudonym:
		cryptoScheme = crypto.CryptoSchemePQRingCTX
		privacyLevel = crypto.PrivacyLevelPseudonym
		break
	case AccountPrivacyLevelPseudonymCT:
		cryptoScheme = crypto.CryptoSchemePQRingCTX
		privacyLevel = crypto.PrivacyLevelPseudonymCT
		break
	default:
		panic("unsupported privacy level of account")
	}
	return cryptoScheme, privacyLevel
}

type AccountType int

const (
	AccountTypeSeeds AccountType = iota
	AccountTypeKeys
)

// ViewAccount encapsulates the ability to
// - determine whether the coin belongs to the corresponding account
// - generate serial number for specified coins
type ViewAccount interface {
	ReceiveCTAUTToken(version uint32, scriptType ctaut.CTAUTScriptType, valueScript []byte,
		txVersion uint32, txOutData []byte) (success bool, v uint64, tokenType crypto.AutTokenType,
		coinValuePK []byte, coinValueSK []byte, err error)

	ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error)
	GenerateSerialNumberWithBlocks(coinID *CoinID, serializedBlocksForRingGroup [][]byte) (coinSerialNumbers []byte, err error)
	GenerateSerialNumbersWithBlocks(coinIDs []*CoinID, serializedBlocksForRingGroup [][]byte) (coinSerialNumbers [][]byte, err error)

	GenerateSerialNumberWithRing(coinID *CoinID, serializedRing []byte) (coinSerialNumbers []byte, err error)

	ViewKeyMaterial() ([]byte, []byte, []byte)

	AccountType() AccountType
}

var _ ViewAccount = &RootSeedViewAccount{}
var _ ViewAccount = &CryptoKeysViewAccount{}

type RootSeedViewAccount struct {
	networkID               NetworkID
	cryptoScheme            crypto.CryptoScheme
	privacyLevel            crypto.PrivacyLevel
	coinSerialNumberKeySeed []byte
	coinValueKeySeed        []byte
	coinDetectorKey         []byte
}

func NewRootSeedViewAccount(
	netID NetworkID,
	accountPrivacyLevel AccountPrivacyLevel,
	coinSerialNumberKeySeed []byte,
	coinValueKeySeed []byte,
	coinDetectorKey []byte,
) *RootSeedViewAccount {
	cryptoScheme, privacyLevel := getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel)
	return &RootSeedViewAccount{
		networkID:    netID,
		cryptoScheme: cryptoScheme, privacyLevel: privacyLevel, coinSerialNumberKeySeed: coinSerialNumberKeySeed, coinValueKeySeed: coinValueKeySeed, coinDetectorKey: coinDetectorKey}
}

func (account *RootSeedViewAccount) AccountType() AccountType {
	return AccountTypeSeeds
}

func (account *RootSeedViewAccount) GenerateSerialNumberWithBlocks(coinID *CoinID, serializedBlocksForRingGroup [][]byte) ([]byte, error) {
	if coinID == nil {
		return nil, nil
	}
	outPoint, err := crypto.NewOutPointFromTxId(coinID.TxID, coinID.Index)
	if err != nil {
		return nil, err
	}
	outPoints := []*crypto.OutPoint{
		outPoint,
	}
	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByRootSeeds(outPoints, serializedBlocksForRingGroup, account.coinSerialNumberKeySeed)
	if err != nil {
		return nil, err
	}
	if len(serialNumbers) != 1 {
		return nil, fmt.Errorf("fail to generate serial number with one coin id")
	}
	return serialNumbers[0], nil
}
func (account *RootSeedViewAccount) GenerateSerialNumberWithRing(coinID *CoinID, serializedRing []byte) ([]byte, error) {
	if coinID == nil {
		return nil, nil
	}
	outPoint, err := crypto.NewOutPointFromTxId(coinID.TxID, coinID.Index)
	if err != nil {
		return nil, err
	}
	outPoints := []*crypto.OutPoint{
		outPoint,
	}
	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByRootSeedsWithRing(outPoints, serializedRing, account.coinSerialNumberKeySeed)
	if err != nil {
		return nil, err
	}
	if len(serialNumbers) != 1 {
		return nil, fmt.Errorf("fail to generate serial number with one coin id")
	}
	return serialNumbers[0], nil
}

func (account *RootSeedViewAccount) GenerateSerialNumbersWithBlocks(coinIDs []*CoinID, serializedBlocksForRingGroup [][]byte) ([][]byte, error) {
	if len(coinIDs) == 0 {
		return nil, nil
	}
	// Prepare outPoints.
	outPoints := make([]*crypto.OutPoint, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		outPoint, err := crypto.NewOutPointFromTxId(coinIDs[i].TxID, coinIDs[i].Index)
		if err != nil {
			return nil, err
		}
		outPoints[i] = outPoint
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByRootSeeds(outPoints, serializedBlocksForRingGroup, account.coinSerialNumberKeySeed)
	if err != nil {
		return nil, err
	}

	// Convert serial numbers to []byte] type and return them.
	coinSerialNumbers := make([][]byte, len(coinIDs))
	for i := 0; i < len(serialNumbers); i++ {
		coinSerialNumbers[i] = serialNumbers[i]
	}

	return coinSerialNumbers, nil
}
func (account *RootSeedViewAccount) ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error) {
	privacyLevel, err := crypto.GetTxoPrivacyLevel(txVersion, txOutData)
	if err != nil {
		return false, 0, err
	}
	if account.privacyLevel != privacyLevel {
		return false, 0, nil
	}

	success, err = crypto.TxoCoinDetectByCoinDetectorRootKey(txVersion, txOutData, account.coinDetectorKey)
	if err != nil {
		return false, 0, err
	}
	if !success {
		return false, 0, nil
	}

	success, v, err = crypto.TxoCoinReceiveByRootSeeds(txVersion, txOutData, account.coinValueKeySeed, account.coinDetectorKey)
	if err != nil {
		return false, 0, err
	}
	return success, v, nil
}

func (account *RootSeedViewAccount) ReceiveCTAUTToken(version uint32, scriptType ctaut.CTAUTScriptType,
	valueScript []byte, txVersion uint32, txOutData []byte) (success bool,
	v uint64, tokenType crypto.AutTokenType,
	cryptoValuePK []byte, cryptoValueSK []byte, err error) {
	if account.privacyLevel != crypto.PrivacyLevelPseudonymCT {
		return false, 0, tokenType, nil, nil, nil
	}

	success, _, err = account.ReceiveCoin(txVersion, txOutData)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	if !success {
		return false, 0, tokenType, nil, nil, nil
	}

	privacyLevel, err := crypto.GetTxoPrivacyLevel(txVersion, txOutData)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	if privacyLevel != crypto.PrivacyLevelPseudonymCT {
		return false, 0, tokenType, nil, nil, nil
	}

	publicRandFromTxo, err := crypto.ExtractPublicRandFromTxo(txVersion, txOutData)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}

	if scriptType == ctaut.Registration || scriptType == ctaut.ReRegistration {
		return true, 0, tokenType, nil, nil, nil
	}

	cryptoValuePK, cryptoValueSK, err = crypto.CryptoValueKeyGen(account.coinValueKeySeed, publicRandFromTxo)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	v, tokenType, err = crypto.ExtractCTAUTTokenValue(version, valueScript, cryptoValuePK, cryptoValueSK)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	return true, v, tokenType, cryptoValuePK, cryptoValueSK, nil
}

func (account *RootSeedViewAccount) ViewKeyMaterial() ([]byte, []byte, []byte) {
	var coinDetectorKey []byte
	if account.coinDetectorKey != nil {
		coinDetectorKey = account.coinDetectorKey
	}
	return account.coinSerialNumberKeySeed, account.coinValueKeySeed, coinDetectorKey
}

type CryptoKeysViewAccount struct {
	networkID         NetworkID
	cryptoScheme      crypto.CryptoScheme
	privacyLevel      crypto.PrivacyLevel
	serialNoSecretKey []byte
	viewSecretKey     []byte
	detectorKey       []byte
	cryptoAddress     *crypto.CryptoAddress
}

func NewCryptoKeyViewAccount(
	networkID NetworkID,
	accountPrivacyLevel AccountPrivacyLevel,
	serialNoSecretKey []byte, viewSecretKey []byte,
	detectorKey []byte, cryptoAddress *crypto.CryptoAddress) *CryptoKeysViewAccount {
	cryptoScheme, privacyLevel := getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel)
	return &CryptoKeysViewAccount{
		networkID:         networkID,
		cryptoScheme:      cryptoScheme,
		privacyLevel:      privacyLevel,
		serialNoSecretKey: serialNoSecretKey,
		viewSecretKey:     viewSecretKey,
		detectorKey:       detectorKey,
		cryptoAddress:     cryptoAddress,
	}
}
func (account *CryptoKeysViewAccount) AccountType() AccountType {
	return AccountTypeKeys
}

func (account *CryptoKeysViewAccount) GenerateSerialNumberWithBlocks(coinID *CoinID, serializedBlocksForRingGroup [][]byte) ([]byte, error) {
	if coinID == nil {
		return nil, nil
	}
	outPoint, err := crypto.NewOutPointFromTxId(coinID.TxID, coinID.Index)
	if err != nil {
		return nil, err
	}
	outPoints := []*crypto.OutPoint{
		outPoint,
	}
	cryptoSerialNumberSecretKeys := [][]byte{
		account.serialNoSecretKey,
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByKeys(outPoints, serializedBlocksForRingGroup, cryptoSerialNumberSecretKeys)
	if err != nil {
		return nil, err
	}

	if len(serialNumbers) != 1 {
		return nil, fmt.Errorf("fail to generate serial number with one coin id")
	}
	return serialNumbers[0], nil
}

func (account *CryptoKeysViewAccount) GenerateSerialNumbersWithBlocks(coinIDs []*CoinID, serializedBlocksForRingGroup [][]byte) ([][]byte, error) {
	if len(coinIDs) == 0 {
		return nil, nil
	}
	// Prepare outPoints.
	outPoints := make([]*crypto.OutPoint, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		outPoint, err := crypto.NewOutPointFromTxId(coinIDs[i].TxID, coinIDs[i].Index)
		if err != nil {
			return nil, err
		}
		outPoints[i] = outPoint
	}

	// Prepare cryptoSecretKeys.
	cryptoSerialNumberSecretKeys := make([][]byte, len(coinIDs))
	for i := 0; i < len(coinIDs); i++ {
		cryptoSerialNumberSecretKeys[i] = account.serialNoSecretKey
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByKeys(outPoints, serializedBlocksForRingGroup, cryptoSerialNumberSecretKeys)
	if err != nil {
		return nil, err
	}

	// Convert serial numbers to []byte] type and return them.
	coinSerialNumbers := make([][]byte, len(coinIDs))
	for i := 0; i < len(serialNumbers); i++ {
		coinSerialNumbers[i] = serialNumbers[i]
	}

	return coinSerialNumbers, nil
}
func (account *CryptoKeysViewAccount) GenerateSerialNumberWithRing(coinID *CoinID, serializedRing []byte) (coinSerialNumbers []byte, err error) {
	if coinID == nil {
		return nil, nil
	}
	outPoint, err := crypto.NewOutPointFromTxId(coinID.TxID, coinID.Index)
	if err != nil {
		return nil, err
	}
	outPoints := []*crypto.OutPoint{
		outPoint,
	}
	cryptoSerialNumberSecretKeys := [][]byte{
		account.serialNoSecretKey,
	}

	// Call API to generate coin serial numbers.
	serialNumbers, err := crypto.GenerateCoinSerialNumberByKeysWithRing(outPoints, serializedRing, cryptoSerialNumberSecretKeys)
	if err != nil {
		return nil, err
	}

	if len(serialNumbers) != 1 {
		return nil, fmt.Errorf("fail to generate serial number with one coin id")
	}
	return serialNumbers[0], nil
}

func (account *CryptoKeysViewAccount) ReceiveCoin(txVersion uint32, txOutData []byte) (success bool, v uint64, err error) {
	coinAddressFromSerializedTxOut, err := crypto.DecodeCoinAddressFromSerializedTxOutData(txVersion, txOutData)
	if err != nil {
		return false, 0, err
	}
	coinAddressFromCryptoAddress := account.cryptoAddress.GetCoinAddress()
	if !bytes.Equal(coinAddressFromCryptoAddress.Data(), coinAddressFromSerializedTxOut.Data()) {
		return false, 0, nil
	}

	copiedVsk := make([]byte, len(account.viewSecretKey))
	copy(copiedVsk, account.viewSecretKey)
	success, v, err = crypto.TxoCoinReceiveByKeys(txVersion, txOutData, account.cryptoAddress.Data(), copiedVsk)
	if err != nil {
		return false, 0, err
	}
	return success, v, nil
}
func (account *CryptoKeysViewAccount) ReceiveCTAUTToken(version uint32, scriptType ctaut.CTAUTScriptType,
	valueScript []byte, txVersion uint32, txOutData []byte) (success bool,
	v uint64, tokenType crypto.AutTokenType,
	coinValuePK []byte, coinValueSK []byte, err error) {
	return false, 0, tokenType, nil, nil, errors.New("unimplemented!")
	privacyLevel, err := crypto.GetTxoPrivacyLevel(txVersion, txOutData)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	if account.privacyLevel != privacyLevel {
		return false, 0, tokenType, nil, nil, nil
	}
	if privacyLevel != crypto.PrivacyLevelPseudonymCT {
		return false, 0, tokenType, nil, nil, nil
	}

	v, tokenType, err = crypto.ExtractCTAUTTokenValue(txVersion, txOutData /*TODO value public key*/, nil, account.viewSecretKey)
	if err != nil {
		return false, 0, tokenType, nil, nil, err
	}
	return true, v, tokenType, nil, nil, nil
}
func (account *CryptoKeysViewAccount) ViewKeyMaterial() ([]byte, []byte, []byte) {
	var coinSerialNoSecretKey []byte
	if account.serialNoSecretKey != nil {
		coinSerialNoSecretKey = account.serialNoSecretKey
	}
	var coinViewSecretKey []byte
	if account.viewSecretKey != nil {
		coinViewSecretKey = account.viewSecretKey
	}

	var coinDetectorKey []byte
	if account.detectorKey != nil {
		coinDetectorKey = account.detectorKey
	}
	return coinSerialNoSecretKey, coinViewSecretKey, coinDetectorKey
}

type Account interface {
	ViewAccount
	GenerateAbelAddress() ([]byte, error)
	SpendKeyMaterial() []byte
	ViewAccount() ViewAccount
}

var _ Account = &RootSeedAccount{}
var _ Account = &CryptoKeysAccount{}

type RootSeedAccount struct {
	RootSeedViewAccount
	coinSpendKeySeed []byte
}

func (account *RootSeedAccount) SpendKeyMaterial() []byte {
	return account.coinSpendKeySeed
}
func (account *RootSeedAccount) ViewAccount() ViewAccount {
	return &account.RootSeedViewAccount
}

func (account *RootSeedAccount) GenerateAbelAddress() ([]byte, error) {
	rootSeeds, err := crypto.NewRootSeeds(
		account.cryptoScheme,
		account.privacyLevel,
		account.coinSpendKeySeed,
		account.coinSerialNumberKeySeed,
		account.coinValueKeySeed,
		account.coinDetectorKey,
	)
	if err != nil {
		return nil, err
	}
	cryptoKeysAndAddress, err := crypto.GenerateCryptoKeysAndAddressByRootSeeds(rootSeeds)
	if err != nil {
		return nil, err
	}
	abelAddress := NewAbelAddressFromCryptoAddress(account.networkID, cryptoKeysAndAddress.CryptoAddress)
	return abelAddress.Data(), nil
}
func NewRootSeedAccount(networkID NetworkID, accountPrivacyLevel AccountPrivacyLevel,
	coinSpendKeySeed []byte, coinSerialNumberKeySeed []byte,
	coinValueKeySeed []byte, coinDetectorKey []byte) *RootSeedAccount {
	cryptoScheme, privacyLevel := getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel)
	return &RootSeedAccount{
		RootSeedViewAccount: RootSeedViewAccount{
			networkID:               networkID,
			cryptoScheme:            cryptoScheme,
			privacyLevel:            privacyLevel,
			coinSerialNumberKeySeed: coinSerialNumberKeySeed,
			coinValueKeySeed:        coinValueKeySeed,
			coinDetectorKey:         coinDetectorKey,
		},
		coinSpendKeySeed: coinSpendKeySeed,
	}
}
func NewRootSeedAccountFromViewAccount(viewAccount RootSeedViewAccount, coinSpendKeySeed []byte) *RootSeedAccount {
	return &RootSeedAccount{
		RootSeedViewAccount: viewAccount,
		coinSpendKeySeed:    coinSpendKeySeed,
	}
}

type CryptoKeysAccount struct {
	CryptoKeysViewAccount
	spendSecretKey []byte
}

func (account *CryptoKeysAccount) SpendKeyMaterial() []byte {
	return account.spendSecretKey
}
func (account *CryptoKeysAccount) ViewAccount() ViewAccount {
	return &account.CryptoKeysViewAccount
}

func (account *CryptoKeysAccount) GenerateAbelAddress() ([]byte, error) {
	abelAddress := NewAbelAddressFromCryptoAddress(account.networkID, account.cryptoAddress)
	return abelAddress.Data(), nil
}

func NewCryptoKeysAccount(networkID NetworkID, accountPrivacyLevel AccountPrivacyLevel,
	spendSecretKey []byte, serialNoSecretKey []byte,
	viewSecretKey []byte, detectorKey []byte, cryptoAddress *crypto.CryptoAddress) *CryptoKeysAccount {
	cryptoScheme, privacyLevel := getCryptoSchemeAndPrivacyLevel(accountPrivacyLevel)
	return &CryptoKeysAccount{
		CryptoKeysViewAccount: CryptoKeysViewAccount{
			networkID:         networkID,
			cryptoScheme:      cryptoScheme,
			privacyLevel:      privacyLevel,
			serialNoSecretKey: serialNoSecretKey,
			viewSecretKey:     viewSecretKey,
			detectorKey:       detectorKey,
			cryptoAddress:     cryptoAddress,
		},
		spendSecretKey: spendSecretKey,
	}
}
func NewCryptoKeysAccountFromViewAccount(viewAccount CryptoKeysViewAccount, spendSecretKey []byte) *CryptoKeysAccount {
	return &CryptoKeysAccount{
		CryptoKeysViewAccount: viewAccount,
		spendSecretKey:        spendSecretKey,
	}
}
