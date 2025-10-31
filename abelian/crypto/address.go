package crypto

import (
	"crypto/sha256"
	"fmt"

	api "github.com/abesuite/abec/sdkapi/v2"
	"golang.org/x/crypto/sha3"
)

// AddressType enumerate all possible address types
type AddressType int

const (
	ADDRESS_TYPE_COIN_ADDRESS   AddressType = 1
	ADDRESS_TYPE_CRYPTO_ADDRESS AddressType = 2
)

func (addressType AddressType) String() string {
	switch addressType {
	case ADDRESS_TYPE_COIN_ADDRESS:
		return "CoinAddress"
	case ADDRESS_TYPE_CRYPTO_ADDRESS:
		return "CryptoAddress"
	default:
		return "UnknownAddress"
	}
}

type Address interface {
	AddressType() AddressType
	Data() []byte
	Validate() error
}

// Define the length of different type of coin address
const (
	COIN_ADDRESS_LENGTH_FULL_PRIVACY_PRE  = 9504
	COIN_ADDRESS_LENGTH_FULL_PRIVACY_RAND = 9633
	COIN_ADDRESS_LENGTH_PSEUDONYM         = 193
)

// CoinAddressType enumerate all possible coin address types
type CoinAddressType int

const (
	COIN_ADDRESS_TYPE_FULL_PRIVACY_PRE  CoinAddressType = 0
	COIN_ADDRESS_TYPE_FULL_PRIVACY_RAND CoinAddressType = 1
	COIN_ADDRESS_TYPE_PSEUDONYM         CoinAddressType = 2
	COIN_ADDRESS_TYPE_PSEUDONYMCT       CoinAddressType = 3
)

// CoinAddress abstract application layer’s functional requirements for
// underlying addresses
//
// Currently, method CoinAddressType is used to distinguish different address type
// Other methods are mainly added for convenience of use, display, etc.
type CoinAddress interface {
	Address

	CoinAddressType() CoinAddressType
	PrivacyLevel() PrivacyLevel
	Fingerprint() []byte
}

func NewCoinAddress(data []byte) (CoinAddress, error) {
	var coinAddress CoinAddress

	switch len(data) {
	case COIN_ADDRESS_LENGTH_FULL_PRIVACY_PRE:
		coinAddress = &CoinAddressFullPrivacyPre{
			data: data,
		}
	case COIN_ADDRESS_LENGTH_FULL_PRIVACY_RAND:
		coinAddress = &CoinAddressFullPrivacy{
			data: data,
		}
	case COIN_ADDRESS_LENGTH_PSEUDONYM:
		if data[0] == uint8(COIN_ADDRESS_TYPE_PSEUDONYM) {
			coinAddress = &CoinAddressPseudonym{
				data: data,
			}
		} else if data[0] == uint8(COIN_ADDRESS_TYPE_PSEUDONYMCT) {
			coinAddress = &CoinAddressPseudonym{
				data: data,
			}
		} else {
			return nil, ErrInvalidAddress
		}

	default:
		log.Errorf("invalid coin address %v with length %d", data, len(data))
		return nil, ErrInvalidAddress
	}

	return coinAddress, nil
}

var _ CoinAddress = &CoinAddressFullPrivacyPre{}
var _ CoinAddress = &CoinAddressFullPrivacy{}
var _ CoinAddress = &CoinAddressPseudonym{}

type CoinAddressFullPrivacyPre struct {
	data []byte
}

func (address *CoinAddressFullPrivacyPre) AddressType() AddressType {
	return ADDRESS_TYPE_COIN_ADDRESS
}

func (address *CoinAddressFullPrivacyPre) CoinAddressType() CoinAddressType {
	return COIN_ADDRESS_TYPE_FULL_PRIVACY_PRE
}

func (address *CoinAddressFullPrivacyPre) Data() []byte {
	return address.data
}

func (address *CoinAddressFullPrivacyPre) PrivacyLevel() PrivacyLevel {
	return PrivacyLevelFullPrivacyPre
}

func (address *CoinAddressFullPrivacyPre) Fingerprint() []byte {
	hash := sha256.Sum256(address.data)
	return hash[:]
}

func (address *CoinAddressFullPrivacyPre) Validate() error {
	if len(address.data) != COIN_ADDRESS_LENGTH_FULL_PRIVACY_PRE {
		log.Errorf("invalid coin address %v with length %d for privacy level %d", address.data, len(address.data), COIN_ADDRESS_LENGTH_FULL_PRIVACY_PRE)
		return fmt.Errorf("coin address data length is not %d", COIN_ADDRESS_LENGTH_FULL_PRIVACY_PRE)
	}
	return nil
}

type CoinAddressFullPrivacy struct {
	data []byte
}

func (address *CoinAddressFullPrivacy) AddressType() AddressType {
	return ADDRESS_TYPE_COIN_ADDRESS
}

func (address *CoinAddressFullPrivacy) CoinAddressType() CoinAddressType {
	return COIN_ADDRESS_TYPE_FULL_PRIVACY_RAND
}

func (address *CoinAddressFullPrivacy) Data() []byte {
	return address.data
}

func (address *CoinAddressFullPrivacy) PrivacyLevel() PrivacyLevel {
	return PrivacyLevelFullPrivacyRand
}

func (address *CoinAddressFullPrivacy) Fingerprint() []byte {
	hash := sha3.Sum256(address.data)
	return hash[:]
}

func (address *CoinAddressFullPrivacy) Validate() error {
	if len(address.data) != COIN_ADDRESS_LENGTH_FULL_PRIVACY_RAND {
		log.Errorf("invalid coin address %v with length %d for privacy level %d", address.data, len(address.data), COIN_ADDRESS_LENGTH_FULL_PRIVACY_RAND)
		return fmt.Errorf("coin address data length is not %d", COIN_ADDRESS_LENGTH_FULL_PRIVACY_RAND)
	}

	return nil
}

type CoinAddressPseudonym struct {
	data []byte
}

func (address *CoinAddressPseudonym) AddressType() AddressType {
	return ADDRESS_TYPE_COIN_ADDRESS
}

func (address *CoinAddressPseudonym) CoinAddressType() CoinAddressType {
	return COIN_ADDRESS_TYPE_PSEUDONYM
}

func (address *CoinAddressPseudonym) Data() []byte {
	return address.data
}

func (address *CoinAddressPseudonym) PrivacyLevel() PrivacyLevel {
	return PrivacyLevelPseudonym
}

func (address *CoinAddressPseudonym) Fingerprint() []byte {
	hash := sha3.Sum256(address.data)
	return hash[:]
}

func (address *CoinAddressPseudonym) Validate() error {
	if len(address.data) != COIN_ADDRESS_LENGTH_PSEUDONYM {
		log.Errorf("invalid coin address %v with length %d for privacy level %d", address.data, len(address.data), COIN_ADDRESS_LENGTH_PSEUDONYM)
		return fmt.Errorf("coin address data length is not %d", COIN_ADDRESS_LENGTH_PSEUDONYM)
	}
	if address.data[0] != uint8(COIN_ADDRESS_LENGTH_PSEUDONYM) {
		return fmt.Errorf("coin address data type is not %d", COIN_ADDRESS_TYPE_PSEUDONYM)
	}

	return nil
}

type CoinAddressPseudonymCT struct {
	data []byte
}

func (address *CoinAddressPseudonymCT) AddressType() AddressType {
	return ADDRESS_TYPE_COIN_ADDRESS
}

func (address *CoinAddressPseudonymCT) CoinAddressType() CoinAddressType {
	return COIN_ADDRESS_TYPE_PSEUDONYMCT
}

func (address *CoinAddressPseudonymCT) Data() []byte {
	return address.data
}

func (address *CoinAddressPseudonymCT) PrivacyLevel() PrivacyLevel {
	return PrivacyLevelPseudonymCT
}

func (address *CoinAddressPseudonymCT) Fingerprint() []byte {
	hash := sha3.Sum256(address.data)
	return hash[:]
}

func (address *CoinAddressPseudonymCT) Validate() error {
	if len(address.data) != COIN_ADDRESS_LENGTH_PSEUDONYM {
		return fmt.Errorf("coin address data length is not %d", COIN_ADDRESS_LENGTH_PSEUDONYM)
	}
	if address.data[0] != uint8(COIN_ADDRESS_TYPE_PSEUDONYMCT) {
		return fmt.Errorf("coin address data type is not %d", COIN_ADDRESS_TYPE_PSEUDONYMCT)
	}

	return nil
}

// Define the length of different type of crypto address
const (
	CRYPTO_ADDRESS_LENGTH_FULL_PRIVACT_PRE  = 10696
	CRYPTO_ADDRESS_LENGTH_FULL_PRIVACY_RAND = 10826
	CRYPTO_ADDRESS_LENGTH_PSEUDONYM         = 198
	CRYPTO_ADDRESS_LENGTH_PSEUDONYMCT       = 1391
)

// CryptoAddress encapsulated coin address for upper layer
type CryptoAddress struct {
	data         []byte
	cryptoScheme CryptoScheme
	privacyLevel PrivacyLevel
	coinAddress  CoinAddress
}

func (a *CryptoAddress) AddressType() AddressType {
	return ADDRESS_TYPE_CRYPTO_ADDRESS
}
func (a *CryptoAddress) Data() []byte {
	return a.data
}
func (a *CryptoAddress) Validate() error {
	switch a.cryptoScheme {
	case CryptoSchemePQRingCT:
		if a.privacyLevel != PrivacyLevelFullPrivacyPre {
			log.Errorf("mismatched privacy level %d and crypto scheme %d", a.privacyLevel, a.cryptoScheme)
			return fmt.Errorf("mismatched crypto scheme %d and privacy level %d", a.cryptoScheme, a.privacyLevel)
		}
		if len(a.data) != CRYPTO_ADDRESS_LENGTH_FULL_PRIVACT_PRE {
			log.Errorf("invalid crypto address %v with length %d for privacy level %d", a.data, len(a.data), CRYPTO_ADDRESS_LENGTH_FULL_PRIVACT_PRE)
			return fmt.Errorf("crypto address data length is not %d, but got %d", CRYPTO_ADDRESS_LENGTH_FULL_PRIVACT_PRE, len(a.data))
		}
	case CryptoSchemePQRingCTX:
		if a.privacyLevel == PrivacyLevelFullPrivacyRand {
			if len(a.data) != CRYPTO_ADDRESS_LENGTH_FULL_PRIVACY_RAND {
				log.Errorf("invalid crypto address %v with length %d for privacy level %d", a.data, len(a.data), CRYPTO_ADDRESS_LENGTH_FULL_PRIVACY_RAND)
				return fmt.Errorf("crypto address data length should be %d, but got %d", CRYPTO_ADDRESS_LENGTH_FULL_PRIVACY_RAND, len(a.data))
			}
		} else if a.privacyLevel == PrivacyLevelPseudonym {
			if len(a.data) != CRYPTO_ADDRESS_LENGTH_PSEUDONYM {
				log.Errorf("invalid crypto address %v with length %d for privacy level %d", a.data, len(a.data), CRYPTO_ADDRESS_LENGTH_PSEUDONYM)
				return fmt.Errorf("crypto address data length should be %d, but got %d", CRYPTO_ADDRESS_LENGTH_PSEUDONYM, len(a.data))
			}
		} else if a.privacyLevel == PrivacyLevelPseudonymCT {
			if len(a.data) != CRYPTO_ADDRESS_LENGTH_PSEUDONYMCT {
				log.Errorf("invalid crypto address %v with length %d for privacy level %d", a.data, len(a.data), CRYPTO_ADDRESS_LENGTH_PSEUDONYMCT)
				return fmt.Errorf("crypto address data length should be %d, but got %d", CRYPTO_ADDRESS_LENGTH_PSEUDONYMCT, len(a.data))
			}
		} else {
			return fmt.Errorf("mismatched crypto scheme %d and privacy level %d", a.cryptoScheme, a.privacyLevel)
		}

	default:
		return ErrInvalidCryptoScheme
	}

	return nil
}

func (a *CryptoAddress) GetCryptoScheme() CryptoScheme {
	return a.cryptoScheme
}
func (a *CryptoAddress) GetPrivacyLevel() PrivacyLevel {
	return a.privacyLevel
}
func (a *CryptoAddress) GetCoinAddress() CoinAddress {
	return a.coinAddress
}

func NewCryptoAddress(data []byte) (*CryptoAddress, error) {
	if len(data) < 4 {
		log.Errorf("invalid crypto address length %d", len(data))
		return nil, ErrInvalidAddress
	}
	cryptoScheme, err := api.DeserializeCryptoScheme(data[:4])
	if err != nil {
		log.Errorf("invalid crypto scheme %v in crypto address", data[:4])
		return nil, fmt.Errorf("%v:%s", ErrInvalidCryptoScheme, err)
	}

	privacyLevel, coinAddrData, err := api.ExtractCoinAddressFromCryptoAddress(data)
	if err != nil {
		log.Errorf("fail to extract coin address from crypto address %v", data)
		return nil, fmt.Errorf("%v:%s", ErrInvalidAddress, err)
	}

	cryptoAddress := &CryptoAddress{
		data:         data,
		cryptoScheme: cryptoScheme,
		privacyLevel: privacyLevel,
	}
	cryptoAddress.coinAddress, err = NewCoinAddress(coinAddrData)
	if err != nil {
		log.Errorf("fail to extract coin address from crypto address %v", data)
		return nil, fmt.Errorf("%v:%s", ErrInvalidAddress, err)
	}

	return cryptoAddress, nil
}
