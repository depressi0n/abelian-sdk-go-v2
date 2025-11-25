package abelian

import (
	"fmt"

	"github.com/abesuite/abec/sdkapi/v2"

	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

const AutScriptVersion = v2.AutScriptVersion

type AutId = v2.AutId

func NewAutId(autIdentifier string) (AutId, error) {
	return v2.NewAutId(autIdentifier)
}

type CTAUTToken struct {
	Version      uint32
	HostOutpoint OutPoint
	ValueScript  []byte
}

type CTAUTScript = v2.AutScript

func ParseCTAUTScript(txVersion uint32, txID string, memo []byte) (CTAUTScript, error) {
	return v2.ParseAutScript(txVersion, txID, memo)
}

type CTAUTMetadata = v2.Metadata

func RegisteredCTAUTMetadata(script CTAUTScript, scriptVersion uint32, txID string, serializedTxOuts [][]byte) (*CTAUTMetadata, error) {
	return v2.RegisteredAutMetadata(script, scriptVersion, txID, serializedTxOuts)
}
func UpdateMetadataFromCTAUTScript(script CTAUTScript, txVersion uint32, txID string, serializedTxOuts [][]byte, metadata *CTAUTMetadata) (*CTAUTMetadata, error) {
	if script == nil {
		return nil, nil
	}
	var err error
	switch script.(type) {
	case *v2.CTAUTRegisterScript:
		return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is a registration script, which should not be called on an existing instance")
	case *v2.CTAUTReRegisterScript:
		err = v2.UpdateAutMetadata(script, txVersion, txID, serializedTxOuts, metadata)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	case *v2.CTAUTMintScript:
		err = v2.UpdateAutMetadata(script, txVersion, txID, serializedTxOuts, metadata)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	case *v2.CTAUTTransferScript:
		err = v2.UpdateAutMetadata(script, txVersion, txID, serializedTxOuts, metadata)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	case *v2.CTAUTBurnScript:
		err = v2.UpdateAutMetadata(script, txVersion, txID, serializedTxOuts, metadata)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	default:
		return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is not supported")
	}
}

func GetGeneratedCTAUTTokens(ctAutScript v2.AutScript, txVersion uint32, txHash string, serializedTxOuts [][]byte) ([]*CTAUTToken, error) {
	version, outpoints, scripts, err := v2.GetGeneratedOutpoints(ctAutScript, txVersion, txHash, serializedTxOuts)
	if err != nil {
		return nil, err
	}
	res := make([]*CTAUTToken, len(outpoints))
	for i := 0; i < len(outpoints); i++ {
		res[i] = &CTAUTToken{
			Version: version,
			HostOutpoint: OutPoint{
				TxHash: outpoints[i].TxId.String(),
				Index:  outpoints[i].Index,
			},
			ValueScript: scripts[i],
		}
	}
	return res, nil
}
func GetConsumedTokenOutpoint(serializedTx []byte, rings []*CoinRing) ([]*OutPoint, error) {
	apiRings := make(map[string]*v2.TxoRing, len(rings))
	for _, ring := range rings {
		apiTxoRing, err := coinRing2ApiTxoRing(ring)
		if err != nil {
			return nil, err
		}
		ringId, err := apiTxoRing.RingId()
		if err != nil {
			return nil, err
		}

		apiRings[ringId] = apiTxoRing
	}
	consumedOutpoints, err := v2.GetConsumedOutpoints(serializedTx, apiRings)
	if err != nil {
		return nil, err
	}
	res := make([]*OutPoint, len(consumedOutpoints))
	for i := 0; i < len(consumedOutpoints); i++ {
		res[i] = &OutPoint{
			TxHash: consumedOutpoints[i].TxId.String(),
			Index:  consumedOutpoints[i].Index,
		}
	}
	return res, nil
}

const CTAUTIdentifierLength = v2.AutIdentifierLength

type CTAUTScriptType = v2.AutScriptType

const (
	CTAUTTypeRegistration   CTAUTScriptType = v2.AutScriptTypeRegistration
	CTAUTTypeReRegistration CTAUTScriptType = v2.AutScriptTypeReRegistration
	CTAUTTypeMint           CTAUTScriptType = v2.AutScriptTypeMint
	CTAUTTypeTransfer       CTAUTScriptType = v2.AutScriptTypeTransfer
	CTAUTTypeBurn           CTAUTScriptType = v2.AutScriptTypeBurn
)

type CTAUTRegisterScript = v2.CTAUTRegisterScript

func CreateCTAUTRegisterScript(version uint32, name []byte, symbol []byte,
	baseUnitName []byte, subUnitName []byte, unitScale uint64,
	autMemo []byte, plannedTotalAmount uint64,
	issuerCryptoAddresses [][]byte, mintThreshold uint8, reRegisterThreshold uint8,
	expiryBlock int32, OutAutRootCoinNum uint8, memo []byte,
) ([]byte, []byte, error) {
	issuerTokens := make([][]byte, len(issuerCryptoAddresses))
	for i, cryptoAddress := range issuerCryptoAddresses {
		// TODO: extract issuer token from crypto address
		issuerTokens[i] = cryptoAddress
	}

	return v2.NewRegistrationScript(
		version,
		name, symbol,
		baseUnitName, subUnitName, unitScale,
		autMemo, plannedTotalAmount,
		issuerTokens, expiryBlock,
		reRegisterThreshold, mintThreshold,
		OutAutRootCoinNum, memo,
	)
}

func CreateCTAUTReRegisterScript(version uint32,
	identifier [CTAUTIdentifierLength]byte,
	autMemo []byte, plannedTotalAmount uint64,
	issuerCryptoAddresses [][]byte, mintThreshold uint8, reregisterThreshold uint8,
	expiryBlock int32, inAutRootTokenNum uint8, outAutRootTokenNum uint8, memo []byte,
) ([]byte, []byte, error) {
	issuerTokens := make([][]byte, len(issuerCryptoAddresses))
	for i, cryptoAddress := range issuerCryptoAddresses {
		// TODO: extract issuer token from crypto address
		issuerTokens[i] = cryptoAddress
	}

	return v2.NewReRegistrationScript(
		version,
		identifier,
		autMemo, plannedTotalAmount,
		issuerTokens, expiryBlock,
		reregisterThreshold, mintThreshold,
		inAutRootTokenNum, outAutRootTokenNum,
		memo,
	)
}

type Recipient struct {
	CryptoAddress crypto.CryptoAddress
	Value         uint64
	HideValue     bool
}

func CreateCTAUTMintScript(version uint32,
	identifier [CTAUTIdentifierLength]byte,
	vin uint64, inAutRootTokenNum uint8, recipients []*Recipient,
	memo []byte,
) ([]byte, []byte, error) {
	var err error

	autTxOutputDescs := make([]*v2.AutTxOutputDesc, len(recipients))
	for i, recipient := range recipients {
		autTxOutputDescs[i], err = v2.NewAutTxOutputDesc(recipient.Value, recipient.CryptoAddress.Data(), recipient.HideValue)
		if err != nil {
			return nil, nil, err
		}
	}

	return v2.NewMintScript(version, identifier, vin, inAutRootTokenNum, autTxOutputDescs, memo)

}

type InputTokenDesc struct {
	version              uint32
	valueScript          []byte
	cryptoValuePublicKey []byte
	cryptoValueSecretKey []byte
	value                uint64
}

func NewInputTokenDesc(version uint32, valueScript []byte, cryptoValuePublicKey []byte, cryptoValueSecretKey []byte, value uint64) *InputTokenDesc {
	return &InputTokenDesc{
		version:              version,
		valueScript:          valueScript,
		cryptoValuePublicKey: cryptoValuePublicKey,
		cryptoValueSecretKey: cryptoValueSecretKey,
		value:                value,
	}
}

func CreateCTAUTTransferScript(version uint32,
	identifier AutId,
	consumedToken []*InputTokenDesc, recipients []*Recipient,
	memo []byte,
) ([]byte, []byte, error) {
	var err error

	autTxInputDescs := make([]*v2.AutTxInputDesc, len(consumedToken))
	for i := 0; i < len(consumedToken); i++ {
		autTxInputDescs[i], err = v2.NewAutTxInputDesc(
			consumedToken[i].version,
			consumedToken[i].valueScript,
			consumedToken[i].cryptoValuePublicKey,
			consumedToken[i].cryptoValueSecretKey,
			consumedToken[i].value,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	autTxOutputDescs := make([]*v2.AutTxOutputDesc, len(recipients))
	for i, recipient := range recipients {
		autTxOutputDescs[i], err = v2.NewAutTxOutputDesc(recipient.Value, recipient.CryptoAddress.Data(), recipient.HideValue)
		if err != nil {
			return nil, nil, err
		}
	}

	return v2.NewTransferScript(version, identifier, autTxInputDescs, autTxOutputDescs, memo)
}

func CreateCTAUTBurnScript(version uint32,
	identifier [CTAUTIdentifierLength]byte,
	consumedToken []*InputTokenDesc, recipients []*Recipient,
	memo []byte,
) ([]byte, []byte, error) {
	var err error

	autTxInputDescs := make([]*v2.AutTxInputDesc, len(consumedToken))
	for i := 0; i < len(consumedToken); i++ {
		autTxInputDescs[i], err = v2.NewAutTxInputDesc(
			consumedToken[i].version,
			consumedToken[i].valueScript,
			consumedToken[i].cryptoValuePublicKey,
			consumedToken[i].cryptoValueSecretKey,
			consumedToken[i].value,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	autTxOutputDescs := make([]*v2.AutTxOutputDesc, len(recipients))
	for i, recipient := range recipients {
		autTxOutputDescs[i], err = v2.NewAutTxOutputDesc(recipient.Value, recipient.CryptoAddress.Data(), recipient.HideValue)
		if err != nil {
			return nil, nil, err
		}
	}

	return v2.NewBurnScript(version, identifier, autTxInputDescs, autTxOutputDescs, memo)

}
