package abelian

import (
	"encoding/hex"
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

type ExtAutScript = v2.ExtAutScript

func ExtractAutScriptFromHostTx(serializedTx []byte) (*ExtAutScript, error) {
	return v2.ExtractAutScriptFromHostTx(serializedTx)
}

type CTAUTMetadata struct {
	Version uint32

	AutIdentifier AutId

	AutName      string
	AutSymbol    string
	BaseUnitName string
	SubUnitName  string
	UnitScale    uint64

	AutMemo                    string
	PlannedTotalSupply         uint64
	Issuers                    []string
	ReregistrationExpireHeight int32
	ReregistrationThreshold    uint8
	MintThreshold              uint8
	PrivacyType                AutPrivacyType

	MintedAmount         uint64
	BurnedAmount         uint64
	ActiveRootTokenSet   map[OutPoint]struct{}
	UpdateScriptVersions []uint32
}

func RegisteredCTAUTMetadata(extScript *ExtAutScript) (*CTAUTMetadata, error) {
	if extScript == nil {
		return nil, nil
	}
	if extScript.Type() != v2.AutScriptTypeRegistration {
		return nil, fmt.Errorf("RegisteredCTAUTMetadata: the input script is not a registration script")
	}
	metadata, err := extScript.CreateAutMetadata()
	if err != nil {
		return nil, err
	}
	issuers := make([]string, len(metadata.Issuers))
	for i := 0; i < len(metadata.Issuers); i++ {
		issuers[i] = hex.EncodeToString(metadata.Issuers[i].CoinAddress())
	}
	rootTokens := make(map[OutPoint]struct{}, len(metadata.ActiveRootTokenSet))
	for _, hostOutpoint := range metadata.ActiveRootTokenSet {
		outpoint := OutPoint{
			TxHash: hostOutpoint.TxHash.String(),
			Index:  hostOutpoint.Index,
		}
		rootTokens[outpoint] = struct{}{}
	}

	res := &CTAUTMetadata{
		Version:                    metadata.Version,
		AutIdentifier:              metadata.AutIdentifier,
		AutName:                    hex.EncodeToString(metadata.AutName),
		AutSymbol:                  hex.EncodeToString(metadata.AutSymbol),
		BaseUnitName:               hex.EncodeToString(metadata.BaseUnitName),
		SubUnitName:                hex.EncodeToString(metadata.SubUnitName),
		UnitScale:                  metadata.UnitScale,
		AutMemo:                    hex.EncodeToString(metadata.AutMemo),
		PlannedTotalSupply:         metadata.PlannedTotalSupply,
		Issuers:                    issuers,
		PrivacyType:                metadata.PrivacyType,
		ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
		MintThreshold:              metadata.MintThreshold,
		ReregistrationThreshold:    metadata.ReregistrationThreshold,
		MintedAmount:               0,
		BurnedAmount:               0,
		ActiveRootTokenSet:         rootTokens,
		UpdateScriptVersions:       metadata.UpdateScriptVersions,
	}

	return res, nil
}
func UpdateMetadataFromCTAUTScript(extScript *ExtAutScript, metadata *CTAUTMetadata) (*CTAUTMetadata, error) {
	if extScript == nil {
		return nil, nil
	}
	if metadata == nil {
		return nil, nil
	}

	switch instance := extScript.AutScript.(type) {
	case *v2.CTAUTRegisterScript:
		return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is a registration script, which should not be called on an existing instance")
	case *v2.CTAUTReRegisterScript:

		v2Issuers := make([]*v2.AutIssuer, len(metadata.Issuers))
		for i := 0; i < len(metadata.Issuers); i++ {
			issuerCoinAddress, err := hex.DecodeString(metadata.Issuers[i])
			if err != nil {
				return nil, err
			}
			v2Issuers[i] = v2.NewAutIssuerFromCoinAddress(issuerCoinAddress)
		}

		activeRootTokenSet := make(map[string]*v2.HostOutPoint, len(metadata.ActiveRootTokenSet))
		for outpoint := range metadata.ActiveRootTokenSet {
			v2Outpoint, err := v2.NewHostOutPoint(outpoint.TxHash, outpoint.Index)
			if err != nil {
				return nil, err
			}
			activeRootTokenSet[v2Outpoint.String()] = v2Outpoint
		}

		v2Metadata := &v2.Metadata{
			Version:                    metadata.Version,
			AutIdentifier:              metadata.AutIdentifier,
			AutName:                    []byte(metadata.AutName),
			AutSymbol:                  []byte(metadata.AutSymbol),
			BaseUnitName:               []byte(metadata.BaseUnitName),
			SubUnitName:                []byte(metadata.SubUnitName),
			UnitScale:                  metadata.UnitScale,
			AutMemo:                    []byte(metadata.AutMemo),
			PlannedTotalSupply:         metadata.PlannedTotalSupply,
			Issuers:                    v2Issuers,
			PrivacyType:                metadata.PrivacyType,
			ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
			ReregistrationThreshold:    metadata.ReregistrationThreshold,
			MintThreshold:              metadata.MintThreshold,
			MintedAmount:               metadata.MintedAmount,
			BurnedAmount:               metadata.BurnedAmount,
			ActiveRootTokenSet:         activeRootTokenSet,
			UpdateScriptVersions:       metadata.UpdateScriptVersions,
		}
		updateAutMetadata, err := extScript.UpdateAutMetadata(v2Metadata)
		if err != nil {
			return nil, err
		}

		issuers := make([]string, len(updateAutMetadata.Issuers))
		for i := 0; i < len(updateAutMetadata.Issuers); i++ {
			issuers[i] = hex.EncodeToString(updateAutMetadata.Issuers[i].CoinAddress())
		}
		rootTokens := make(map[OutPoint]struct{}, len(updateAutMetadata.ActiveRootTokenSet))
		for _, hostOutpoint := range updateAutMetadata.ActiveRootTokenSet {
			outpoint := OutPoint{
				TxHash: hostOutpoint.TxHash.String(),
				Index:  hostOutpoint.Index,
			}
			rootTokens[outpoint] = struct{}{}
		}
		res := &CTAUTMetadata{
			Version:                    updateAutMetadata.Version,
			AutIdentifier:              updateAutMetadata.AutIdentifier,
			AutName:                    hex.EncodeToString(updateAutMetadata.AutName),
			AutSymbol:                  hex.EncodeToString(updateAutMetadata.AutSymbol),
			BaseUnitName:               hex.EncodeToString(updateAutMetadata.BaseUnitName),
			SubUnitName:                hex.EncodeToString(updateAutMetadata.SubUnitName),
			UnitScale:                  updateAutMetadata.UnitScale,
			AutMemo:                    hex.EncodeToString(updateAutMetadata.AutMemo),
			PlannedTotalSupply:         updateAutMetadata.PlannedTotalSupply,
			Issuers:                    issuers,
			PrivacyType:                updateAutMetadata.PrivacyType,
			ReregistrationExpireHeight: updateAutMetadata.ReregistrationExpireHeight,
			MintThreshold:              updateAutMetadata.MintThreshold,
			ReregistrationThreshold:    updateAutMetadata.ReregistrationThreshold,
			MintedAmount:               0,
			BurnedAmount:               0,
			ActiveRootTokenSet:         rootTokens,
			UpdateScriptVersions:       metadata.UpdateScriptVersions,
		}

		return res, nil
	case *v2.CTAUTMintScript:
		copiedMetadata := &CTAUTMetadata{
			Version:                    metadata.Version,
			AutIdentifier:              metadata.AutIdentifier,
			AutName:                    metadata.AutName,
			AutSymbol:                  metadata.AutSymbol,
			BaseUnitName:               metadata.BaseUnitName,
			SubUnitName:                metadata.SubUnitName,
			UnitScale:                  metadata.UnitScale,
			AutMemo:                    metadata.AutMemo,
			PlannedTotalSupply:         metadata.PlannedTotalSupply,
			Issuers:                    metadata.Issuers,
			PrivacyType:                metadata.PrivacyType,
			ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
			ReregistrationThreshold:    metadata.ReregistrationThreshold,
			MintThreshold:              metadata.MintThreshold,
			MintedAmount:               metadata.MintedAmount,
			BurnedAmount:               metadata.BurnedAmount,
			ActiveRootTokenSet:         metadata.ActiveRootTokenSet,
			UpdateScriptVersions:       metadata.UpdateScriptVersions,
		}

		copiedMetadata.MintedAmount += instance.Vin()
		return copiedMetadata, nil
	case *v2.CTAUTTransferScript:
		copiedMetadata := &CTAUTMetadata{
			Version:                    metadata.Version,
			AutIdentifier:              metadata.AutIdentifier,
			AutName:                    metadata.AutName,
			AutSymbol:                  metadata.AutSymbol,
			BaseUnitName:               metadata.BaseUnitName,
			SubUnitName:                metadata.SubUnitName,
			UnitScale:                  metadata.UnitScale,
			AutMemo:                    metadata.AutMemo,
			PlannedTotalSupply:         metadata.PlannedTotalSupply,
			Issuers:                    metadata.Issuers,
			PrivacyType:                metadata.PrivacyType,
			ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
			ReregistrationThreshold:    metadata.ReregistrationThreshold,
			MintThreshold:              metadata.MintThreshold,
			MintedAmount:               metadata.MintedAmount,
			BurnedAmount:               metadata.BurnedAmount,
			ActiveRootTokenSet:         metadata.ActiveRootTokenSet,
			UpdateScriptVersions:       metadata.UpdateScriptVersions,
		}

		return copiedMetadata, nil
	case *v2.CTAUTBurnScript:
		generatedTokens := extScript.GeneratedTokens()
		if len(generatedTokens) == 0 {
			return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is a burn script, but no tokens are generated")
		}

		// the last token would be viewed as burned
		burnedToken := generatedTokens[len(generatedTokens)-1]

		burnedValue, autTxoType, err := v2.ExtractAutTokenValue(burnedToken.Version, burnedToken.ValueScript, nil, nil)
		if err != nil {
			return nil, err
		}
		if autTxoType == v2.AutTokenTypeHidden {
			return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is a burn script, but the burned token is hidden")
		}
		copiedMetadata := &CTAUTMetadata{
			Version:                    metadata.Version,
			AutIdentifier:              metadata.AutIdentifier,
			AutName:                    metadata.AutName,
			AutSymbol:                  metadata.AutSymbol,
			BaseUnitName:               metadata.BaseUnitName,
			SubUnitName:                metadata.SubUnitName,
			UnitScale:                  metadata.UnitScale,
			AutMemo:                    metadata.AutMemo,
			PlannedTotalSupply:         metadata.PlannedTotalSupply,
			Issuers:                    metadata.Issuers,
			PrivacyType:                metadata.PrivacyType,
			ReregistrationExpireHeight: metadata.ReregistrationExpireHeight,
			ReregistrationThreshold:    metadata.ReregistrationThreshold,
			MintThreshold:              metadata.MintThreshold,
			MintedAmount:               metadata.MintedAmount,
			BurnedAmount:               metadata.BurnedAmount,
			ActiveRootTokenSet:         metadata.ActiveRootTokenSet,
			UpdateScriptVersions:       metadata.UpdateScriptVersions,
		}

		copiedMetadata.BurnedAmount += burnedValue
		return copiedMetadata, nil
	default:
		return nil, fmt.Errorf("UpdateMetadataFromCTAUTScript: the input script is not supported")
	}
}

func GetGeneratedCTAUTTokens(extAutScript *ExtAutScript) ([]*CTAUTToken, error) {
	if extAutScript == nil {
		return nil, nil
	}
	generatedTokens := extAutScript.GeneratedTokens()
	res := make([]*CTAUTToken, len(generatedTokens))
	for i := 0; i < len(generatedTokens); i++ {
		res[i] = &CTAUTToken{
			Version: generatedTokens[i].Version,
			HostOutpoint: OutPoint{
				TxHash: generatedTokens[i].HostOutPoint.TxHash.String(),
				Index:  generatedTokens[i].HostOutPoint.Index,
			},
			ValueScript: generatedTokens[i].ValueScript,
		}
	}
	return res, nil
}
func GetConsumedTokenOutpoint(extAutScript *ExtAutScript) ([]*OutPoint, error) {
	comsumedHostOutpoints := extAutScript.ConsumedHostOutpoints()
	res := make([]*OutPoint, len(comsumedHostOutpoints))
	for i := 0; i < len(comsumedHostOutpoints); i++ {
		res[i] = &OutPoint{
			TxHash: comsumedHostOutpoints[i].TxHash.String(),
			Index:  comsumedHostOutpoints[i].Index,
		}
	}
	return res, nil
}

type AutPrivacyType = v2.AutPrivacyType

const (
	AutPrivacyTypeUnlimited     AutPrivacyType = v2.AutPrivacyTypeUnlimited
	AutPrivacyTypeLimitedPublic AutPrivacyType = v2.AutPrivacyTypeLimitedPublic
	AutPrivacyTypeLimitedHidden AutPrivacyType = v2.AutPrivacyTypeLimitedHidden
)

type CTAUTScriptType = v2.AutScriptType

const (
	CTAUTTypeRegistration   CTAUTScriptType = v2.AutScriptTypeRegistration
	CTAUTTypeReRegistration CTAUTScriptType = v2.AutScriptTypeReRegistration
	CTAUTTypeMint           CTAUTScriptType = v2.AutScriptTypeMint
	CTAUTTypeTransfer       CTAUTScriptType = v2.AutScriptTypeTransfer
	CTAUTTypeBurn           CTAUTScriptType = v2.AutScriptTypeBurn
)

type CTAUTRegisterScript = v2.CTAUTRegisterScript

func CreateCTAUTRegisterScript(
	version uint32,
	name []byte, symbol []byte, baseUnitName []byte, subUnitName []byte, unitScale uint64,
	autMemo []byte, plannedTotalAmount uint64,
	issuerCoinAddresses [][]byte, reregistrationExpireHeight int32, reRegisterThreshold uint8, mintThreshold uint8,
	privacyType AutPrivacyType,
	outStartIndex uint8, outAutRootCoinNum uint8,
	memo []byte,
) ([]byte, []byte, error) {
	issuers := make([][]byte, len(issuerCoinAddresses))
	for i, coinAddress := range issuerCoinAddresses {
		issuers[i] = coinAddress
	}

	return v2.NewRegistrationScript(
		version,
		name, symbol, baseUnitName, subUnitName, unitScale,
		autMemo, plannedTotalAmount,
		issuers, reregistrationExpireHeight, reRegisterThreshold, mintThreshold,
		privacyType,
		outStartIndex, outAutRootCoinNum,
		memo,
	)
}

func CreateCTAUTReRegisterScript(
	version uint32,
	identifier AutId,
	autMemo []byte, plannedTotalAmount uint64,
	issuerCoinAddresses [][]byte, reregistrationExpireHeight int32, reregisterThreshold uint8, mintThreshold uint8,
	privacyType AutPrivacyType,
	inStartIndex uint8, inAutRootTokenNum uint8,
	outStartIndex uint8, outAutRootTokenNum uint8,
	memo []byte,
) ([]byte, []byte, error) {
	issuers := make([][]byte, len(issuerCoinAddresses))
	for i, coinAddress := range issuerCoinAddresses {
		issuers[i] = coinAddress
	}

	return v2.NewReRegistrationScript(
		version,
		identifier,
		autMemo, plannedTotalAmount,
		issuers, reregistrationExpireHeight, reregisterThreshold, mintThreshold,
		privacyType,
		inStartIndex, inAutRootTokenNum,
		outStartIndex, outAutRootTokenNum,
		memo,
	)
}

type Recipient struct {
	CryptoAddress crypto.CryptoAddress
	Value         uint64
	HideValue     bool
}

func CreateCTAUTMintScript(
	version uint32,
	identifier AutId,
	vin uint64,
	inStartIndex uint8, inAutRootTokenNum uint8,
	outStartIndex uint8, recipients []*Recipient,
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

	return v2.NewMintScript(version,
		identifier,
		vin,
		inStartIndex, inAutRootTokenNum,
		outStartIndex, autTxOutputDescs,
		memo,
	)
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

func CreateCTAUTTransferScript(
	version uint32,
	identifier AutId,
	inStartIndex uint8, consumedToken []*InputTokenDesc,
	outStartIndex uint8, recipients []*Recipient,
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

	return v2.NewTransferScript(version,
		identifier,
		inStartIndex, autTxInputDescs,
		outStartIndex, autTxOutputDescs,
		memo,
	)
}

func CreateCTAUTBurnScript(version uint32,
	identifier AutId,
	inStartIndex uint8, consumedToken []*InputTokenDesc,
	outStartIndex uint8, recipients []*Recipient,
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

	return v2.NewBurnScript(version,
		identifier,
		inStartIndex, autTxInputDescs,
		outStartIndex, autTxOutputDescs,
		memo,
	)
}
