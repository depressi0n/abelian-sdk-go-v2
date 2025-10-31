package crypto

import (
	"fmt"

	api "github.com/abesuite/abec/sdkapi/v2"
)

type CryptoScheme = api.CryptoScheme

const (
	CryptoSchemePQRingCT  = api.CryptoSchemePQRingCT
	CryptoSchemePQRingCTX = api.CryptoSchemePQRingCTX
)

func CryptoSchemeSerializeSize() int {
	return api.CryptoSchemeSerializeSize()
}
func SerializeCryptoScheme(cryptoScheme CryptoScheme) []byte {
	return api.SerializeCryptoScheme(cryptoScheme)
}
func DeserializeCryptoScheme(serializedCryptoScheme []byte) (CryptoScheme, error) {
	return api.DeserializeCryptoScheme(serializedCryptoScheme)
}

// PrivacyLevel alias type for api.PrivacyLevel to identifier different privacy level
// Defined by a cryptographic layer to differentiate for different target
//
// currently, two full-privacy addresses and one pseudonym address are supported
type PrivacyLevel = api.PrivacyLevel

const (
	PrivacyLevelFullPrivacyPre  PrivacyLevel = api.PrivacyLevelRINGCTPre // for back-compatibility
	PrivacyLevelFullPrivacyRand              = api.PrivacyLevelRINGCT
	PrivacyLevelPseudonym                    = api.PrivacyLevelPSEUDONYM
	PrivacyLevelPseudonymCT                  = api.PrivacyLevelPSEUDONYMCT
)

func GetCryptoSchemeParamSeedBytesLen(cryptoScheme CryptoScheme) (int, error) {
	return api.GetCryptoSchemeParamSeedBytesLen(cryptoScheme)
}
func GetParamKeyGenPublicRandBytesLen(cryptoScheme CryptoScheme) (int, error) {
	return api.GetParamKeyGenPublicRandBytesLen(cryptoScheme)
}

// GenerateSeed generate a safe seed
// For back-compatibility, hoisting origin implementation here
//
// for CryptoSchemePQRingCT: to keep back-compatibility
//  1. PrivacyLevelFullPrivacyPre
//     cryptoScheme || coinSpendKeyRootSeed || coinValueKeyRootSeed
//
// for CryptoSchemePQRingCTX:
//  1. PrivacyLevelFullPrivacyRand
//     cryptoScheme || privacyLevel || coinSpendKeyRootSeed || coinSerialNumberKeyRootSeed || coinValueKeyRootSeed || coinDetectorRootKey
//  2. PrivacyLevelPseudonym
//     cryptoScheme || privacyLevel || coinSpendKeyRootSeed || coinDetectorRootKey
func GenerateSeed(cryptoScheme CryptoScheme, privacyLevel PrivacyLevel) (*CryptoSeeds, error) {
	underlyingSeedLen, err := api.GetCryptoSchemeParamSeedBytesLen(cryptoScheme)
	if err != nil {
		return nil, err
	}
	switch cryptoScheme {
	case CryptoSchemePQRingCT:
		if privacyLevel != PrivacyLevelFullPrivacyPre {
			return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", privacyLevel, cryptoScheme)
		}
		coinAddressKeySeed := RandomBytes(underlyingSeedLen)
		coinValueKeySeed := RandomBytes(underlyingSeedLen)
		return NewRandSeeds(cryptoScheme, privacyLevel, coinAddressKeySeed, nil, coinValueKeySeed, nil, nil)
	case CryptoSchemePQRingCTX:
		if privacyLevel != PrivacyLevelFullPrivacyRand &&
			privacyLevel != PrivacyLevelPseudonym &&
			privacyLevel != PrivacyLevelPseudonymCT {
			return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", privacyLevel, cryptoScheme)
		}
		coinSpendKeyRootSeed := RandomBytes(underlyingSeedLen)
		var coinSerialNumberKeyRootSeed, coinValueKeyRootSeed []byte
		if privacyLevel == PrivacyLevelFullPrivacyRand {
			coinSerialNumberKeyRootSeed = RandomBytes(underlyingSeedLen)
			coinValueKeyRootSeed = RandomBytes(underlyingSeedLen)
		}
		if privacyLevel == PrivacyLevelPseudonymCT {
			coinValueKeyRootSeed = RandomBytes(underlyingSeedLen)
		}
		coinDetectorRootKey := RandomBytes(underlyingSeedLen)
		return NewRootSeeds(cryptoScheme, privacyLevel, coinSpendKeyRootSeed, coinSerialNumberKeyRootSeed, coinValueKeyRootSeed, coinDetectorRootKey)

	default:
		log.Errorf("error %v, got %v", ErrInvalidCryptoScheme, cryptoScheme)
		return nil, fmt.Errorf("%v, got %v", ErrInvalidCryptoScheme, cryptoScheme)
	}
}

// GenerateCryptoKeysAndAddressBySeedBytes generate an address key pair from specified seeds
// NOTE: when seedsType is root
func GenerateCryptoKeysAndAddressBySeedBytes(seedBytes []byte) (*CryptoKeysAndAddress, error) {
	cryptoSeeds, err := deserializeSeed(seedBytes)
	if err != nil {
		return nil, err
	}
	if cryptoSeeds.seedsType == seedsTypeRoot {
		return GenerateCryptoKeysAndAddressByRootSeeds(cryptoSeeds)
	} else if cryptoSeeds.seedsType == seedsTypeRand {
		return GenerateCryptoKeysAndAddressByRandSeeds(cryptoSeeds)
	} else {
		log.Errorf("call GenerateCryptoKeysAndAddressBySeedBytes with invalid seeds")
		return nil, AssertError("call GenerateCryptoKeysAndAddressBySeedBytes with invalid seeds")
	}
}

// GenerateCryptoKeysAndAddressByRootSeeds generate an address key pair from root seeds
// NOTE:
// 1. Multiple call use will produce DIFFERENT pairs
// 2. SHOULD call ONLY when crypto scheme is CryptoSchemePQRingCTX
//
// Two ways to generate the same address pair:
// 1. use ExtractPublicRandFromCryptoAddress to extract the public rand from the generated address,
// and then call GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand to generate
// 2. use ExtractPublicRandFromCryptoAddress to extract the public rand from the generated address,
// and with that call GenerateRandSeedsByRootSeedsFromPublicRand to generate the rand seed,
// and then call GenerateCryptoKeysAndAddressByRandSeeds to generate
func GenerateCryptoKeysAndAddressByRootSeeds(rootSeeds *CryptoSeeds) (*CryptoKeysAndAddress, error) {
	if rootSeeds.seedsType != seedsTypeRoot {
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRootSeeds with invalid type seeds")
	}
	cryptoAddressBytes, cryptoSpendSecretKey,
		cryptoSerialNoSecretKey, cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyGenByRootSeeds(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey)
	if err != nil {
		return nil, err
	}

	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		log.Errorf("fail to generate crypto address with bytes %v", cryptoAddressBytes)
		return nil, err
	}

	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// ExtractPublicRandFromCryptoAddress extract public rand from crypto address
func ExtractPublicRandFromCryptoAddress(cryptoAddress *CryptoAddress) ([]byte, error) {
	publicRand, err := api.ExtractPublicRandFromCryptoAddress(cryptoAddress.Data())
	if err != nil {
		log.Errorf("fail to extract public rand from crypto address %v", cryptoAddress.Data())
	}
	return publicRand, err
}

// GenerateRandSeedsByRootSeedsFromPublicRand generate rand seed with specified root seed and public rand
// NOTE: SHOULD call ONLY when crypto scheme is CryptoSchemePQRingCTX
func GenerateRandSeedsByRootSeedsFromPublicRand(rootSeedBytes []byte, publicRand []byte) (*CryptoSeeds, error) {
	rootSeeds, err := deserializeSeed(rootSeedBytes)
	if err != nil {
		return nil, err
	}
	if rootSeeds.seedsType != seedsTypeRoot {
		log.Errorf("call GenerateRandSeedsByRootSeedsFromPublicRand with invalid type seeds")
		return nil, AssertError("call GenerateRandSeedsByRootSeedsFromPublicRand with invalid seeds")
	}

	if rootSeeds.cryptoScheme != CryptoSchemePQRingCTX {
		log.Errorf("expected crypto scheme %d, but got %d ", CryptoSchemePQRingCTX, rootSeeds.cryptoScheme)
		return nil, fmt.Errorf("expected crypto scheme %d, but got %d ", CryptoSchemePQRingCTX, rootSeeds.cryptoScheme)
	}
	if rootSeeds.privacyLevel != PrivacyLevelFullPrivacyRand &&
		rootSeeds.privacyLevel != PrivacyLevelPseudonym &&
		rootSeeds.privacyLevel != PrivacyLevelPseudonymCT {
		log.Errorf("invalid privacy level %d for crypto scheme %d", rootSeeds.privacyLevel, rootSeeds.cryptoScheme)
		return nil, fmt.Errorf("invalid privacy level %d for crypto scheme %d", rootSeeds.privacyLevel, rootSeeds.cryptoScheme)
	}

	coinSpendKeyRandSeed, coinSerialNumberKeyRandSeed,
		coinValueKeyRandSeed, coinDetectorKey, err := api.RandSeedsGenByRootSeedsFromPublicRand(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey,
		publicRand)
	if err != nil {
		return nil, fmt.Errorf("fail to generate crypto seed from root seed")
	}

	return NewRandSeeds(rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		coinSpendKeyRandSeed, coinSerialNumberKeyRandSeed,
		coinValueKeyRandSeed, coinDetectorKey,
		publicRand)
}

// GenerateCryptoKeysAndAddressByRandSeeds generate an address key pair from rand seeds
// Different from GenerateCryptoKeysAndAddressByRootSeeds, multiple call use will produce THE SAME pairs
func GenerateCryptoKeysAndAddressByRandSeeds(randSeeds *CryptoSeeds) (*CryptoKeysAndAddress, error) {
	if randSeeds.seedsType != seedsTypeRand {
		log.Errorf("call GenerateCryptoKeysAndAddressByRandSeeds with invalid type seeds")
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRandSeeds invalid seeds")
	}
	var coinDetectorKey []byte
	if randSeeds.coinDetectorKey != nil {
		coinDetectorKey = randSeeds.coinDetectorKey
	}

	cryptoAddressBytes, cryptoSpendSecretKey, cryptoSerialNoSecretKey,
		cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyGenByRandSeeds(
		randSeeds.cryptoScheme, randSeeds.privacyLevel,
		randSeeds.coinSpendKeySeed, randSeeds.coinSerialNumberKeySeed,
		randSeeds.coinValueKeySeed, coinDetectorKey,
		randSeeds.publicRand)
	if err != nil {
		return nil, err
	}

	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		log.Errorf("fail to generate crypto address with bytes %v", cryptoAddressBytes)
		return nil, err
	}
	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand generate the address key pair from root seed
// and public rand which can extracted from crypto address by calling ExtractPublicRandFromCryptoAddress
func GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand(rootSeedBytes []byte, publicRand []byte) (*CryptoKeysAndAddress, error) {
	rootSeeds, err := deserializeSeed(rootSeedBytes)
	if err != nil {
		return nil, err
	}
	if rootSeeds.seedsType != seedsTypeRoot {
		return nil, AssertError("call GenerateCryptoKeysAndAddressByRootSeedsFromPublicRand with invalid seeds")
	}

	cryptoAddressBytes, cryptoSpendSecretKey, cryptoSerialNoSecretKey,
		cryptoViewSecretKey, cryptoDetectorKey, err := api.CryptoAddressKeyReGenByRootSeedsFromPublicRand(
		rootSeeds.cryptoScheme, rootSeeds.privacyLevel,
		rootSeeds.coinSpendKeySeed, rootSeeds.coinSerialNumberKeySeed,
		rootSeeds.coinValueKeySeed, rootSeeds.coinDetectorKey,
		publicRand)
	if err != nil {
		log.Errorf("fail to re-generate crypto address from root seed")
		return nil, err
	}

	cryptoAddress, err := NewCryptoAddress(cryptoAddressBytes)
	if err != nil {
		log.Errorf("fail to generate crypto address from crypto address bytes %v", cryptoAddressBytes)
		return nil, err
	}
	cryptoKeysAndAddress := &CryptoKeysAndAddress{
		SpendSecretKey:    cryptoSpendSecretKey,
		SerialNoSecretKey: cryptoSerialNoSecretKey,
		ViewSecretKey:     cryptoViewSecretKey,
		DetectorKey:       cryptoDetectorKey,
		CryptoAddress:     cryptoAddress,
	}

	return cryptoKeysAndAddress, nil
}

// DecodeCoinAddressFromSerializedTxOutData extract coin address from serialized transaction output
func DecodeCoinAddressFromSerializedTxOutData(txVersion uint32, txOutData []byte) (CoinAddress, error) {
	// potentially use the latest transaction version default
	coinAddressData, err := api.ExtractCoinAddressFromSerializedTxOut(txVersion, txOutData)
	if err != nil {
		log.Errorf("fail to extract coin address from tx version %d and data %v", txVersion, txOutData)
		return nil, err
	}

	return NewCoinAddress(coinAddressData)
}

type OutPoint = api.OutPoint

func NewOutPointFromTxId(txID string, index uint8) (*OutPoint, error) {
	outPoint, err := api.NewOutPointFromTxIdStr(txID, index)
	if err != nil {
		log.Errorf("fail to create outpoint from txid %s and index %d", txID, index)
	}
	return outPoint, err
}

func GetTxoPrivacyLevel(txVersion uint32, txOutData []byte) (PrivacyLevel, error) {
	privacyLevel, err := api.GetTxoPrivacyLevel(txVersion, txOutData)
	if err != nil {
		log.Errorf("fail to get txo privacy level from version %d and data %v", txVersion, txOutData)
	}
	return privacyLevel, err
}

func TxoCoinDetectByCoinDetectorRootKey(txVersion uint32, serializedTxOut []byte, coinDetectorRootKey []byte) (bool, error) {
	success, err := api.TxoCoinDetectByCoinDetectorRootKey(txVersion, serializedTxOut, coinDetectorRootKey)
	if err != nil {
		log.Errorf("fail to detect txo coin from version %d and data %v with root key", txVersion, serializedTxOut)
	}
	return success, err
}

func TxoCoinReceiveByRootSeeds(txVersion uint32, serializedTxOut []byte, coinValueKeyRootSeed []byte, coinDetectorRootKey []byte) (bool, uint64, error) {
	success, value, err := api.TxoCoinReceiveByRootSeeds(txVersion, serializedTxOut, coinValueKeyRootSeed, coinDetectorRootKey)
	if err != nil {
		log.Errorf("fail to receive txo coin from version %d and data %v with root seeds", txVersion, serializedTxOut)
	}
	return success, value, err
}
func PrivacyLevelPseudonymTxoCoinParse(txVersion uint32, serializedTxOut []byte) (value uint64, err error) {
	return api.PrivacyLevelPseudonymTxoCoinParse(txVersion, serializedTxOut)
}

func GenerateCoinSerialNumberByRootSeeds(
	outPoints []*OutPoint,
	serializedBlocksForRingGroup [][]byte,
	coinSerialNumberKeyRootSeed []byte) (serialNumbers [][]byte, err error) {
	serialNumber, err := api.GenerateCoinSerialNumberByRootSeeds(outPoints, serializedBlocksForRingGroup, coinSerialNumberKeyRootSeed)
	if err != nil {
		log.Errorf("fail to generate serial number using serialized blocks with root seeds")
	}
	return serialNumber, err
}
func GenerateCoinSerialNumberByRootSeedsWithRing(
	outPoints []*OutPoint,
	serializedRing []byte,
	coinSerialNumberKeyRootSeed []byte) (serialNumbers [][]byte, err error) {
	serialNumber, err := api.GenerateCoinSerialNumberByRootSeedsWithRing(outPoints, serializedRing, coinSerialNumberKeyRootSeed)
	if err != nil {
		log.Errorf("fail to generate serial number using serialized ring with root seeds")
	}
	return serialNumber, err
}

func TxoCoinReceiveByKeys(txVersion uint32, serializedTxOut []byte, cryptoAddress []byte, cryptoValueSecretKey []byte) (bool, uint64, error) {
	success, value, err := api.TxoCoinReceiveByKeys(txVersion, serializedTxOut, cryptoAddress, cryptoValueSecretKey)
	if err != nil {
		log.Errorf("fail to receive txo coin from version %d with keys", txVersion)
	}
	return success, value, err
}

func GenerateCoinSerialNumberByKeys(outPoints []*OutPoint,
	serializedBlocksForRingGroup [][]byte,
	cryptoSnsks [][]byte) (serialNumbers [][]byte, err error) {
	serialNumber, err := api.GenerateCoinSerialNumberByKeys(outPoints, serializedBlocksForRingGroup, cryptoSnsks)
	if err != nil {
		log.Errorf("fail to generate serial number using serialized blocks with keys")
	}
	return serialNumber, err
}
func GenerateCoinSerialNumberByKeysWithRing(outPoints []*OutPoint,
	serializedRing []byte,
	cryptoSnsks [][]byte) (serialNumbers [][]byte, err error) {
	serialNumber, err := api.GenerateCoinSerialNumberByKeysWithRing(outPoints, serializedRing, cryptoSnsks)
	if err != nil {
		log.Errorf("fail to generate serial number using serialized ring with keys")
	}
	return serialNumber, err
}

func ExtractPublicRandFromTxo(txVersion uint32, serializedTxOut []byte) ([]byte, error) {
	return api.ExtractPublicRandFromTxo(txVersion, serializedTxOut)
}

type AutTokenType = api.AutTokenType

const (
	AutTokenTypeHidden = api.AutTokenTypeHidden
	AutTokenTypePublic = api.AutTokenTypePublic
)

func ExtractCTAUTTokenValue(version uint32, valueScript []byte, cryptoVpk []byte, cryptoVsk []byte) (uint64, AutTokenType, error) {

	value, tokenType, err := api.ExtractAutTokenValue(version, valueScript, cryptoVpk, cryptoVsk)
	if err != nil {
		return 0, AutTokenTypeHidden, err
	}
	return value, tokenType, err
}
func CryptoValueKeyGen(coinValueKeySeed []byte, publicRand []byte) ([]byte, []byte, error) {
	return api.CryptoValueKeyReGenByRootSeedsFromPublicRand(api.CryptoSchemePQRingCTX, api.PrivacyLevelPSEUDONYMCT, coinValueKeySeed, publicRand)
}
