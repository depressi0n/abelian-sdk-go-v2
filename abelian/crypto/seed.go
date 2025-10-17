package crypto

import (
	"fmt"
)

type seedsType int

const (
	seedsTypeRand seedsType = 0
	seedsTypeRoot           = 1
)

func (seedType seedsType) String() string {
	switch seedType {
	case seedsTypeRand:
		return "RandSeeds"
	case seedsTypeRoot:
		return "RootSeeds"
	default:
		return "UnknownSeeds"
	}
}

// CryptoSeeds encapsulate cryptography seeds for different purposes, details as follow:
// - seedsTypeRoot: allow to generate multiple address, and supports scanning coins of all generated addresses through seed
// (cryptoScheme,privacyLevel)                              coinSpendKeySeed coinSerialNumberKeySeed  coinValueKeySeed coinDetectorKey publicRand
// (CryptoSchemePQRingCT, PrivacyLevelFullPrivacyPre)           -                  -                      -               -            -
// (CryptoSchemePQRingCTX, PrivacyLevelFullPrivacyRand)        yes              yes                     yes               yes          no
// (CryptoSchemePQRingCTX, PrivacyLevelPseudonym)                yes               no                      no               yes          no
//
// - seedsTypeRand: only one address can be generated, and supports scanning coins through seed or generated keys
// (cryptoScheme,privacyLevel)                              coinSpendKeySeed coinSerialNumberKeySeed  coinValueKeySeed coinDetectorKey publicRand
// (CryptoSchemePQRingCT, PrivacyLevelFullPrivacyPre)           yes               no                     yes                no           no   // for back-compatibility
// (CryptoSchemePQRingCTX, PrivacyLevelFullPrivacyRand)         yes              yes                     yes               yes          yes
// (CryptoSchemePQRingCTX, PrivacyLevelPseudonym)                 yes               no                      no               yes          yes
type CryptoSeeds struct {
	seedsType               seedsType
	cryptoScheme            CryptoScheme
	privacyLevel            PrivacyLevel
	coinSpendKeySeed        []byte
	coinSerialNumberKeySeed []byte
	coinValueKeySeed        []byte
	coinDetectorKey         []byte
	publicRand              []byte
}

func (s *CryptoSeeds) Type() string {
	return s.seedsType.String()
}
func (s *CryptoSeeds) CryptoScheme() CryptoScheme {
	return s.cryptoScheme
}

func (s *CryptoSeeds) PrivacyLevel() PrivacyLevel {
	return s.privacyLevel
}

func (s *CryptoSeeds) CoinSpendKeySeed() []byte {
	return s.coinSpendKeySeed
}

func (s *CryptoSeeds) CoinSerialNumberKeySeed() []byte {
	return s.coinSerialNumberKeySeed
}

func (s *CryptoSeeds) CoinValueKeySeed() []byte {
	return s.coinValueKeySeed
}

func (s *CryptoSeeds) CoinDetectorKey() []byte {
	return s.coinDetectorKey
}

func (s *CryptoSeeds) PublicRand() []byte {
	return s.publicRand
}

func (s *CryptoSeeds) String() string {
	return fmt.Sprintf("%s{%d|%d}{%x|%x|%x|%x|%x}",
		s.seedsType.String(), s.cryptoScheme, s.privacyLevel,
		s.coinSpendKeySeed, s.coinValueKeySeed,
		s.coinValueKeySeed, s.coinDetectorKey,
		s.publicRand)
}

func (s *CryptoSeeds) Validate() error {
	switch s.cryptoScheme {
	case CryptoSchemePQRingCT:
		if s.seedsType != seedsTypeRand {
			log.Errorf("mismatched seed type %d with crypto scheme %d", s.seedsType, s.cryptoScheme)
			return ErrMismatchedSeedType
		}
		if s.privacyLevel != PrivacyLevelFullPrivacyPre {
			log.Errorf("mismatched privacy level %d with crypto scheme %d", s.privacyLevel, s.cryptoScheme)
			return ErrMismatchedCryptoSchemePrivacyLevel
		}
		if s.coinSpendKeySeed == nil && len(s.coinSpendKeySeed) == 0 {
			return ErrCorruptedSeed
		}
		if s.coinValueKeySeed == nil && len(s.coinValueKeySeed) == 0 {
			return ErrCorruptedSeed
		}
	case CryptoSchemePQRingCTX:
		if s.privacyLevel != PrivacyLevelFullPrivacyRand &&
			s.privacyLevel != PrivacyLevelPseudonym &&
			s.privacyLevel != PrivacyLevelPseudonymCT {
			log.Errorf("mismatched privacy level %d with crypto scheme %d", s.privacyLevel, s.cryptoScheme)
			return ErrMismatchedCryptoSchemePrivacyLevel
		}
		if s.seedsType != seedsTypeRand && s.seedsType != seedsTypeRoot {
			log.Errorf("mismatched seed type %d with crypto scheme %d", s.seedsType, s.cryptoScheme)
			return ErrMismatchedCryptoSchemePrivacyLevel
		}
		if s.coinSpendKeySeed == nil && len(s.coinSpendKeySeed) == 0 {
			return ErrCorruptedSeed
		}
		if s.privacyLevel == PrivacyLevelFullPrivacyRand {
			if s.coinSerialNumberKeySeed == nil && len(s.coinSerialNumberKeySeed) == 0 {
				return ErrCorruptedSeed
			}
			if s.coinValueKeySeed == nil && len(s.coinValueKeySeed) == 0 {
				return ErrCorruptedSeed
			}
		} else if s.privacyLevel == PrivacyLevelFullPrivacyRand {
			//if s.coinSerialNumberKeySeed == nil && len(s.coinSerialNumberKeySeed) == 0 {
			//	return ErrCorruptedSeed
			//}
			if s.coinValueKeySeed == nil && len(s.coinValueKeySeed) == 0 {
				return ErrCorruptedSeed
			}
		}
		if s.coinDetectorKey == nil && len(s.coinDetectorKey) == 0 {
			return ErrCorruptedSeed
		}
		if s.seedsType == seedsTypeRand {
			if s.publicRand == nil || len(s.publicRand) == 0 {
				return ErrCorruptedSeed
			}
		}
	default:
		log.Errorf("invalid crypto scheme: %d", s.cryptoScheme)
		return ErrInvalidCryptoScheme
	}

	return nil
}

func (s *CryptoSeeds) Serialize() ([]byte, error) {
	if err := s.Validate(); err != nil {
		log.Errorf("failed to serialize crypto seeds: %v", err)
		return nil, err
	}

	cryptoSchemeSize := CryptoSchemeSerializeSize()
	serializeCryptoScheme := SerializeCryptoScheme(s.cryptoScheme)

	underlyingSeedLen, err := GetCryptoSchemeParamSeedBytesLen(s.cryptoScheme)
	if err != nil {
		return nil, err
	}

	switch s.cryptoScheme {
	case CryptoSchemePQRingCT:
		if s.privacyLevel != PrivacyLevelFullPrivacyPre {
			return nil, ErrMismatchedCryptoSchemePrivacyLevel
		}
		addressKeyCryptoSeed := make([]byte, 0, cryptoSchemeSize+2*underlyingSeedLen)
		addressKeyCryptoSeed = append(addressKeyCryptoSeed, serializeCryptoScheme...)
		addressKeyCryptoSeed = append(addressKeyCryptoSeed, s.coinSpendKeySeed...)
		addressKeyCryptoSeed = append(addressKeyCryptoSeed, s.coinValueKeySeed...)
		return addressKeyCryptoSeed, nil
	case CryptoSchemePQRingCTX:
		if s.privacyLevel != PrivacyLevelFullPrivacyRand &&
			s.privacyLevel != PrivacyLevelPseudonym &&
			s.privacyLevel != PrivacyLevelPseudonymCT {
			return nil, ErrMismatchedCryptoSchemePrivacyLevel
		}
		expectedSeedLen := 2 * underlyingSeedLen
		if s.privacyLevel == PrivacyLevelFullPrivacyRand {
			expectedSeedLen = 4 * underlyingSeedLen
		}

		// compute total length
		length := cryptoSchemeSize + 1 + expectedSeedLen
		if s.seedsType == seedsTypeRand {
			publicRandLen, _ := GetParamKeyGenPublicRandBytesLen(s.cryptoScheme)
			length += publicRandLen
		}

		addressKeySeed := make([]byte, 0, length)
		addressKeySeed = append(addressKeySeed, serializeCryptoScheme...)
		addressKeySeed = append(addressKeySeed, byte(s.privacyLevel))

		addressKeySeed = append(addressKeySeed, s.coinSpendKeySeed...)
		if s.privacyLevel == PrivacyLevelFullPrivacyRand {
			addressKeySeed = append(addressKeySeed, s.coinSerialNumberKeySeed...)
			addressKeySeed = append(addressKeySeed, s.coinValueKeySeed...)
		}
		if s.privacyLevel == PrivacyLevelPseudonymCT {
			//addressKeySeed = append(addressKeySeed, s.coinSerialNumberKeySeed...)
			addressKeySeed = append(addressKeySeed, s.coinValueKeySeed...)
		}
		addressKeySeed = append(addressKeySeed, s.coinDetectorKey...)

		if s.seedsType == seedsTypeRand {
			addressKeySeed = append(addressKeySeed, s.publicRand...)
		}

		return addressKeySeed, nil

	default:
		return nil, ErrInvalidCryptoScheme
	}
}
func NewRootSeeds(cryptoScheme CryptoScheme, privacyLevel PrivacyLevel,
	coinSpendKeySeed []byte, coinSerialNumberKeySeed []byte,
	coinValueKeySeed []byte, coinDetectorKey []byte) (*CryptoSeeds, error) {
	if cryptoScheme != CryptoSchemePQRingCTX {
		return nil, ErrInvalidCryptoScheme
	}
	if privacyLevel != PrivacyLevelFullPrivacyRand &&
		privacyLevel != PrivacyLevelPseudonym &&
		privacyLevel != PrivacyLevelPseudonymCT {
		return nil, ErrInvalidPrivacyLevel
	}

	seed := &CryptoSeeds{
		seedsType:               seedsTypeRoot,
		cryptoScheme:            cryptoScheme,
		privacyLevel:            privacyLevel,
		coinSpendKeySeed:        coinSpendKeySeed,
		coinSerialNumberKeySeed: coinSerialNumberKeySeed,
		coinValueKeySeed:        coinValueKeySeed,
		coinDetectorKey:         coinDetectorKey,
		publicRand:              nil,
	}
	if privacyLevel == PrivacyLevelPseudonymCT {
		seed.coinSerialNumberKeySeed = nil
		//seed.coinValueKeySeed = nil
	}
	if privacyLevel == PrivacyLevelPseudonym {
		seed.coinSerialNumberKeySeed = nil
		seed.coinValueKeySeed = nil
	}
	return seed, nil
}
func NewRandSeeds(cryptoScheme CryptoScheme, privacyLevel PrivacyLevel,
	coinSpendKeySeed []byte, coinSerialNumberKeySeed []byte, coinValueKeySeed []byte,
	coinDetectorKey []byte, publicRand []byte) (*CryptoSeeds, error) {
	seed := &CryptoSeeds{
		seedsType:               seedsTypeRand,
		cryptoScheme:            cryptoScheme,
		privacyLevel:            privacyLevel,
		coinSpendKeySeed:        coinSpendKeySeed,
		coinSerialNumberKeySeed: nil,
		coinValueKeySeed:        nil,
		coinDetectorKey:         nil,
		publicRand:              nil,
	}
	switch cryptoScheme {
	case CryptoSchemePQRingCT:
		if privacyLevel != PrivacyLevelFullPrivacyPre {
			return nil, ErrInvalidPrivacyLevel
		}
		seed.coinValueKeySeed = coinValueKeySeed
	case CryptoSchemePQRingCTX:
		if privacyLevel != PrivacyLevelFullPrivacyRand &&
			privacyLevel != PrivacyLevelPseudonym &&
			privacyLevel != PrivacyLevelPseudonymCT {
			return nil, ErrInvalidPrivacyLevel
		}

		seed.coinSerialNumberKeySeed = coinSerialNumberKeySeed
		seed.coinValueKeySeed = coinValueKeySeed
		seed.coinDetectorKey = coinDetectorKey
		seed.publicRand = publicRand

		if privacyLevel == PrivacyLevelPseudonymCT {
			seed.coinSerialNumberKeySeed = nil
		}
		if privacyLevel == PrivacyLevelPseudonym {
			seed.coinSerialNumberKeySeed = nil
			seed.coinValueKeySeed = nil
		}
	default:
		return nil, ErrInvalidCryptoScheme

	}

	return seed, nil
}

func deserializeSeed(seed []byte) (*CryptoSeeds, error) {
	cryptoSchemeSize := CryptoSchemeSerializeSize()
	if len(seed) < cryptoSchemeSize {
		return nil, fmt.Errorf("invalid seed seed")
	}
	cryptoScheme, err := DeserializeCryptoScheme(seed[:cryptoSchemeSize])
	if err != nil {
		return nil, fmt.Errorf("can not parse the crypto seed")
	}

	underlyingSeedLen, err := GetCryptoSchemeParamSeedBytesLen(cryptoScheme)
	if err != nil {
		return nil, err
	}

	var privacyLevel PrivacyLevel
	var coinSpendKeyRootSeed, coinSerialNumberKeyRootSeed, coinValueKeyRootSeed, coinDetectorRootKey []byte
	switch cryptoScheme {
	case CryptoSchemePQRingCT:
		if len(seed) != cryptoSchemeSize+2*underlyingSeedLen {
			return nil, fmt.Errorf("invalid length of crypto seed")
		}
		return NewRandSeeds(
			cryptoScheme, PrivacyLevelFullPrivacyPre,
			seed[cryptoSchemeSize:cryptoSchemeSize+underlyingSeedLen],
			nil,
			seed[cryptoSchemeSize+underlyingSeedLen:],
			nil,
			nil)

	case CryptoSchemePQRingCTX:
		if len(seed) < cryptoSchemeSize+1+2*underlyingSeedLen {
			return nil, fmt.Errorf("invalid length of root seed")
		}
		offset := cryptoSchemeSize

		privacyLevel = PrivacyLevel(seed[offset])
		offset += 1

		if privacyLevel != PrivacyLevelFullPrivacyRand &&
			privacyLevel != PrivacyLevelPseudonym &&
			privacyLevel != PrivacyLevelPseudonymCT {
			return nil, fmt.Errorf("corrupted crypto seed")
		}

		publicRandLen, _ := GetParamKeyGenPublicRandBytesLen(cryptoScheme)
		if privacyLevel == PrivacyLevelFullPrivacyRand {
			if len(seed) != cryptoSchemeSize+1+4*underlyingSeedLen &&
				len(seed) != cryptoSchemeSize+1+4*underlyingSeedLen+publicRandLen {
				return nil, fmt.Errorf("invalid length of seed")
			}
		}
		if privacyLevel == PrivacyLevelPseudonymCT {
			if len(seed) != cryptoSchemeSize+1+3*underlyingSeedLen &&
				len(seed) != cryptoSchemeSize+1+3*underlyingSeedLen+publicRandLen {
				return nil, fmt.Errorf("invalid length of seed")
			}
		}
		if privacyLevel == PrivacyLevelPseudonym {
			if len(seed) != cryptoSchemeSize+1+2*underlyingSeedLen && len(seed) != cryptoSchemeSize+1+2*underlyingSeedLen+publicRandLen {
				return nil, fmt.Errorf("invalid length of seed")
			}
		}

		coinSpendKeyRootSeed = seed[offset : offset+underlyingSeedLen]
		offset += underlyingSeedLen
		if privacyLevel == PrivacyLevelFullPrivacyRand {
			coinSerialNumberKeyRootSeed = seed[offset : offset+underlyingSeedLen]
			offset += underlyingSeedLen
			coinValueKeyRootSeed = seed[offset : offset+underlyingSeedLen]
			offset += underlyingSeedLen
		}
		if privacyLevel == PrivacyLevelPseudonymCT {
			//coinSerialNumberKeyRootSeed = seed[offset : offset+underlyingSeedLen]
			//offset += underlyingSeedLen
			coinValueKeyRootSeed = seed[offset : offset+underlyingSeedLen]
			offset += underlyingSeedLen
		}
		coinDetectorRootKey = seed[offset : offset+underlyingSeedLen]
		offset += underlyingSeedLen

		if offset < len(seed) {
			if offset+publicRandLen == len(seed) {
				return NewRandSeeds(cryptoScheme, privacyLevel, coinSpendKeyRootSeed, coinSerialNumberKeyRootSeed,
					coinValueKeyRootSeed, coinDetectorRootKey, seed[offset:])
			}
		}

	default:
		return nil, ErrCorruptedSeed
	}

	return NewRootSeeds(cryptoScheme, privacyLevel, coinSpendKeyRootSeed, coinSerialNumberKeyRootSeed,
		coinValueKeyRootSeed, coinDetectorRootKey)
}
func NewCryptoSeedFromBytes(bytes []byte) (*CryptoSeeds, error) {
	return deserializeSeed(bytes)
}
