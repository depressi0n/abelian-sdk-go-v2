package abelian

import (
	"bytes"
	"fmt"

	api "github.com/abesuite/abec/sdkapi/v2"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
	abelAddr "github.com/pqabelian/abeutil/address/instanceaddress"
)

const (
	ABEL_ADDRESS_LENGTH_FULL_PRIVACT_PRE = 10729
	ABEL_ADDRESS_LENGTH_RINGCT           = 10859
	ABEL_ADDRESS_LENGTH_PSEUDONYM        = 231
)

// AbelAddress encapsulated crypto address for application layer
type AbelAddress struct {
	data          []byte
	netID         NetworkID
	cryptoAddress *crypto.CryptoAddress
}

func (address *AbelAddress) Data() []byte {
	return address.data
}

func (address *AbelAddress) Validate() error {
	if len(address.data) != ABEL_ADDRESS_LENGTH_FULL_PRIVACT_PRE &&
		len(address.data) != ABEL_ADDRESS_LENGTH_RINGCT &&
		len(address.data) != ABEL_ADDRESS_LENGTH_PSEUDONYM {
		return fmt.Errorf("abel address data length is not one of {%d,%d,%d}",
			ABEL_ADDRESS_LENGTH_FULL_PRIVACT_PRE,
			ABEL_ADDRESS_LENGTH_RINGCT,
			ABEL_ADDRESS_LENGTH_PSEUDONYM,
		)
	}

	chainID := address.GetNetID()
	if chainID < 0 || chainID > 15 {
		return fmt.Errorf("abel address chain id is not in range [0, 15]")
	}

	cryptoAddress := address.GetCryptoAddress()
	bl, _ := api.CheckCryptoAddress(cryptoAddress.Data())
	if !bl {
		return fmt.Errorf("abel address crypto address is not cryptographically valid")
	}

	checksum := address.GetChecksum()
	calculatedChecksum := abelAddr.CheckSum(append([]byte{byte(chainID)}, cryptoAddress.Data()...))
	if !bytes.Equal(checksum, calculatedChecksum) {
		return fmt.Errorf("abel address checksum is not valid")
	}

	return nil
}
func NewAbelAddress(data []byte) (*AbelAddress, error) {
	var abelAddress AbelAddress
	if len(data) <= 1+abelAddr.CheckSumLength() {
		return nil, fmt.Errorf("specifed address has invalid length")
	}
	if int(data[0]) > len(NetName2NetID) {
		return nil, fmt.Errorf("specifed address has unknown net ID")
	}
	var err error
	abelAddress.netID = NetworkID(data[0])
	abelAddress.cryptoAddress, err = crypto.NewCryptoAddress(data[1 : len(data)-abelAddr.CheckSumLength()])
	if err != nil {
		return nil, fmt.Errorf("fail to create crypto address: %s", err)
	}
	abelAddress.data = data
	return &abelAddress, nil
}

func NewAbelAddressFromCryptoAddress(netID NetworkID, cryptoAddress *crypto.CryptoAddress) *AbelAddress {
	instanceAddress := abelAddr.NewInstanceAddress(byte(netID), cryptoAddress.Data())
	serializedInstanceAddress := instanceAddress.Serialize()
	checkSum := abelAddr.CheckSum(serializedInstanceAddress)
	abelAddressData := append(serializedInstanceAddress, checkSum...)

	return &AbelAddress{
		data:          abelAddressData,
		netID:         netID,
		cryptoAddress: cryptoAddress,
	}
}

func (address *AbelAddress) GetNetID() NetworkID {
	return address.netID
}

func (address *AbelAddress) GetCryptoAddress() *crypto.CryptoAddress {
	return address.cryptoAddress
}

func (address *AbelAddress) GetChecksum() []byte {
	instanceAddress := abelAddr.NewInstanceAddress(byte(address.netID), address.cryptoAddress.Data())
	serializedInstanceAddress := instanceAddress.Serialize()
	return abelAddr.CheckSum(serializedInstanceAddress)
}
