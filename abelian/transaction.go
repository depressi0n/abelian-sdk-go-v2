package abelian

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	api "github.com/abesuite/abec/sdkapi/v2"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian/crypto"
)

type TxInDesc struct {
	BlockHeight      int64
	BlockID          string
	TxVersion        uint32
	TxID             string
	TxOutIndex       uint8
	TxOutData        []byte
	CoinValue        int64
	CoinSerialNumber []byte
}

func SortTxInDescs(txIndescs []*TxInDesc) error {
	// check firstly
	for i := 0; i < len(txIndescs); i++ {
		_, err := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		if err != nil {
			sdkLog.Errorf("SortTxInDescs: %v", err)
			return err
		}
	}

	// sort
	sort.SliceStable(txIndescs, func(i, j int) bool {
		coinAddressIPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		coinAddressJPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[j].TxVersion, txIndescs[j].TxOutData)

		// Part I  [crypto.PrivacyLevelFullPrivacyPre, crypto.PrivacyLevelFullPrivacyRand]
		// Part II [crypto.PrivacyLevelPseudonym, crypto.PrivacyLevelPseudonymCT]
		if coinAddressIPrivacyLevel < crypto.PrivacyLevelPseudonym &&
			coinAddressJPrivacyLevel >= crypto.PrivacyLevelPseudonym {
			return true
		}

		// Part I keep the origin order

		// Part II-I [ (crypto.PrivacyLevelPseudonymCT,1) (crypto.PrivacyLevelPseudonymCT,1) ... ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txIndescs[i].CoinValue == 1 {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txIndescs[j].CoinValue == 1 {
			return false
		}
		// Part II-II [ (crypto.PrivacyLevelPseudonymCT,*) (crypto.PrivacyLevelPseudonym,*) ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return false
		}

		return false
	})
	return nil
}

type TxOutDesc struct {
	AbelAddress *AbelAddress
	CoinValue   int64
}

func SortTxOutDesc(txOutdescs []*TxOutDesc) error {
	sort.SliceStable(txOutdescs, func(i, j int) bool {
		coinAddressIPrivacyLevel := txOutdescs[i].AbelAddress.GetCryptoAddress().GetPrivacyLevel()
		coinAddressJPrivacyLevel := txOutdescs[j].AbelAddress.GetCryptoAddress().GetPrivacyLevel()

		// Part I  [crypto.PrivacyLevelFullPrivacyPre, crypto.PrivacyLevelFullPrivacyRand]
		// Part II [crypto.PrivacyLevelPseudonym, crypto.PrivacyLevelPseudonymCT]
		if coinAddressIPrivacyLevel < crypto.PrivacyLevelPseudonym &&
			coinAddressJPrivacyLevel >= crypto.PrivacyLevelPseudonym {
			return true
		}

		// Part I keep the origin order

		// Part II-I [ (crypto.PrivacyLevelPseudonymCT,1) (crypto.PrivacyLevelPseudonymCT,1) ... ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txOutdescs[i].CoinValue == 1 {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txOutdescs[j].CoinValue == 1 {
			return false
		}
		// Part II-II [ (crypto.PrivacyLevelPseudonymCT,*) (crypto.PrivacyLevelPseudonym,*) ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return false
		}

		return false
	})
	return nil
}

type TxDesc struct {
	TxInDescs        []*TxInDesc
	TxOutDescs       []*TxOutDesc
	TxFee            int64
	TxMemo           []byte
	TxRingBlockDescs map[int64]*TxBlockDesc
}
type CoinRing struct {
	Version          uint32
	RingBlockHeight  int32
	CoinIDRing       *CoinIDRing
	SerializedTxOuts [][]byte
	IsCoinbase       bool
}

func (ring *CoinRing) Serialize(w io.Writer) error {
	apiTxoRing, err := coinRing2ApiTxoRing(ring)
	if err != nil {
		return err
	}
	return apiTxoRing.Serialize(w)
}

func (ring *CoinRing) RingId() (string, error) {
	apiTxoRing, err := coinRing2ApiTxoRing(ring)
	if err != nil {
		return "", err
	}
	return apiTxoRing.RingId()
}
func apiTxoRing2CoinRing(apiTxoRing *api.TxoRing) (*CoinRing, error) {
	coinIDRing, err := outPointRing2CoinIDRing(apiTxoRing.OutPointRing)
	if err != nil {
		return nil, err
	}
	coinRing := &CoinRing{
		Version:          apiTxoRing.Version,
		RingBlockHeight:  apiTxoRing.RingBlockHeight,
		CoinIDRing:       coinIDRing,
		SerializedTxOuts: apiTxoRing.SerializedTxOuts,
		IsCoinbase:       apiTxoRing.IsCoinbase,
	}
	return coinRing, nil
}

func coinRing2ApiTxoRing(coinRing *CoinRing) (*api.TxoRing, error) {
	coinIDRing, err := coinIDRing2OutPointRing(coinRing.CoinIDRing)
	if err != nil {
		return nil, err
	}
	apiTxoRing := &api.TxoRing{
		Version:          coinRing.Version,
		RingBlockHeight:  coinRing.RingBlockHeight,
		OutPointRing:     coinIDRing,
		SerializedTxOuts: coinRing.SerializedTxOuts,
		IsCoinbase:       coinRing.IsCoinbase,
	}
	return apiTxoRing, nil
}

func NewCoinRing(ringVersion uint32, ringBlockHeight int64, blockIDs []string, coins []*CoinID, serializedTxOuts [][]byte, fromCoinbase bool) (*CoinRing, error) {
	if len(coins) == 0 {
		return nil, fmt.Errorf("no coins")
	}
	return &CoinRing{
		Version:         ringVersion,
		RingBlockHeight: int32(ringBlockHeight),
		CoinIDRing: &CoinIDRing{
			Version:  ringVersion,
			BlockIDs: blockIDs,
			CoinIDs:  coins,
		},
		SerializedTxOuts: serializedTxOuts,
		IsCoinbase:       fromCoinbase,
	}, nil
}

type TxInDescWithRing struct {
	BlockHeight      int64
	BlockID          string
	TxVersion        uint32
	TxID             string
	TxOutIndex       uint8
	TxOutData        []byte
	CoinValue        int64
	CoinSerialNumber []byte
	TxoRing          *CoinRing
}

func SortTxInDescWithRing(txIndescs []*TxInDescWithRing) error {
	// check firstly
	for i := 0; i < len(txIndescs); i++ {
		_, err := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		if err != nil {
			sdkLog.Errorf("SortTxInDescs: %v", err)
			return err
		}
	}

	// sort
	sort.SliceStable(txIndescs, func(i, j int) bool {
		coinAddressIPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[i].TxVersion, txIndescs[i].TxOutData)
		coinAddressJPrivacyLevel, _ := crypto.GetTxoPrivacyLevel(txIndescs[j].TxVersion, txIndescs[j].TxOutData)

		// Part I  [crypto.PrivacyLevelFullPrivacyPre, crypto.PrivacyLevelFullPrivacyRand]
		// Part II [crypto.PrivacyLevelPseudonym, crypto.PrivacyLevelPseudonymCT]
		if coinAddressIPrivacyLevel < crypto.PrivacyLevelPseudonym &&
			coinAddressJPrivacyLevel >= crypto.PrivacyLevelPseudonym {
			return true
		}

		// Part I keep the origin order

		// Part II-I [ (crypto.PrivacyLevelPseudonymCT,1) (crypto.PrivacyLevelPseudonymCT,1) ... ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txIndescs[i].CoinValue == 1 {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			txIndescs[j].CoinValue == 1 {
			return false
		}
		// Part II-II [ (crypto.PrivacyLevelPseudonymCT,*) (crypto.PrivacyLevelPseudonym,*) ]
		if coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return true
		}
		if coinAddressJPrivacyLevel == crypto.PrivacyLevelPseudonymCT &&
			coinAddressIPrivacyLevel == crypto.PrivacyLevelPseudonym {
			return false
		}

		return false
	})
	return nil
}

type TxDescWithRing struct {
	TxInDescs  []*TxInDescWithRing
	TxOutDescs []*TxOutDesc
	TxFee      int64
	TxMemo     []byte
}

func NewTxDesc(txInDescs []*TxInDesc, txOutDescs []*TxOutDesc, txFee int64, txRingBlockDescs map[int64]*TxBlockDesc) *TxDesc {
	return &TxDesc{
		TxInDescs:        txInDescs,
		TxOutDescs:       txOutDescs,
		TxFee:            txFee,
		TxRingBlockDescs: txRingBlockDescs,
	}
}
func NewTxDescWithRing(txInDescs []*TxInDescWithRing, txOutDescs []*TxOutDesc, txFee int64) *TxDescWithRing {
	return &TxDescWithRing{
		TxInDescs:  txInDescs,
		TxOutDescs: txOutDescs,
		TxFee:      txFee,
	}
}
func NewTxDescWithRingWithMemo(txInDescs []*TxInDescWithRing, txOutDescs []*TxOutDesc, txFee int64, txMemo []byte) *TxDescWithRing {
	return &TxDescWithRing{
		TxInDescs:  txInDescs,
		TxOutDescs: txOutDescs,
		TxFee:      txFee,
		TxMemo:     txMemo,
	}
}

type UnsignedRawTx struct {
	Data       []byte
	AutWitness []byte
}

type TxBlockDesc struct {
	BinData []byte
	Height  int64
}

func NewTxBlockDesc(binData []byte, height int64) *TxBlockDesc {
	return &TxBlockDesc{
		BinData: binData,
		Height:  height,
	}
}

type SignedRawTx struct {
	Data []byte
	TxID string
}

// GenerateUnsignedRawTx make an unsigned transaction,
// which can be signed with singer key by calling GenerateSignedRawTx
func GenerateUnsignedRawTx(txDesc *TxDesc) (*UnsignedRawTx, error) {
	// Prepare outPointsToSpend.
	outPointsToSpend := make([]*api.OutPoint, 0, len(txDesc.TxInDescs))
	for i := 0; i < len(txDesc.TxInDescs); i++ {
		outPoint, err := api.NewOutPointFromTxIdStr(txDesc.TxInDescs[i].TxID, txDesc.TxInDescs[i].TxOutIndex)
		if err != nil {
			sdkLog.Errorf("fail to new outpoint in GenerateUnsignedRawTx with error %v", err)
			return nil, err
		}
		outPointsToSpend = append(outPointsToSpend, outPoint)
	}

	// Prepare serializedBlocksForRingGroup.
	serializedBlocksForRingGroup := getSerializedBlocksForRingGroup(txDesc.TxRingBlockDescs)

	// Prepare txRequestOutputDesc.
	txRequestOutputDescs := make([]*api.TxRequestOutputDesc, 0, len(txDesc.TxOutDescs))
	for i := 0; i < len(txDesc.TxOutDescs); i++ {
		cryptoAddressData := txDesc.TxOutDescs[i].AbelAddress.GetCryptoAddress().Data()
		coinValue := uint64(txDesc.TxOutDescs[i].CoinValue)
		txRequestOutputDesc := api.NewTxRequestOutputDesc(cryptoAddressData, coinValue)
		txRequestOutputDescs = append(txRequestOutputDescs, txRequestOutputDesc)
	}

	// Call API to build the serializedTxRequestDesc.
	serializedTxRequestDesc, err := api.BuildTransferTxRequestDescFromBlocks(
		outPointsToSpend,
		serializedBlocksForRingGroup,
		txRequestOutputDescs,
		uint64(txDesc.TxFee),
		txDesc.TxMemo,
	)
	if err != nil {
		sdkLog.Errorf("fail to build tranasction request from blocks: %v", err)
		return nil, err
	}

	return &UnsignedRawTx{
		Data: serializedTxRequestDesc,
	}, nil
}

// GenerateUnsignedRawTxWithRing make an unsigned transaction,
// which can be signed with singer key by calling GenerateSignedRawTx
func GenerateUnsignedRawTxWithRing(txDesc *TxDescWithRing) (*UnsignedRawTx, error) {
	// Prepare outPointsToSpend and serializedTxoRings.
	outPointsToSpend := make([]*api.OutPoint, 0, len(txDesc.TxInDescs))
	serializedTxoRings := make([][]byte, 0, len(txDesc.TxInDescs))
	for i := 0; i < len(txDesc.TxInDescs); i++ {
		outPoint, err := api.NewOutPointFromTxIdStr(txDesc.TxInDescs[i].TxID, txDesc.TxInDescs[i].TxOutIndex)
		if err != nil {
			sdkLog.Errorf("fail to new outpoint in GenerateUnsignedRawTx with error %v", err)
			return nil, err
		}
		outPointsToSpend = append(outPointsToSpend, outPoint)

		// whether txo in corresponding ring
		if txDesc.TxInDescs[i].TxoRing == nil {
			sdkLog.Errorf("fail to found corresponding txo ring detail for outpoint in GenerateUnsignedRawTxWithRing")

			return nil, fmt.Errorf("GenerateUnsignedRawTxWithRing: at least one of the input OutPoing can not find the corresponding TxoRing")
		}

		w := bytes.Buffer{}
		err = txDesc.TxInDescs[i].TxoRing.Serialize(&w)
		if err != nil {
			sdkLog.Errorf("fail to serialized txo ring detail for outpoint in GenerateUnsignedRawTxWithRing")

			return nil, fmt.Errorf("GenerateUnsignedRawTxWithRing: fail to serialized txo ring detail for outpoint in GenerateUnsignedRawTxWithRing")
		}
		serializedTxoRings = append(serializedTxoRings, w.Bytes())

	}

	// Prepare txRequestOutputDesc.
	txRequestOutputDescs := make([]*api.TxRequestOutputDesc, 0, len(txDesc.TxOutDescs))
	for i := 0; i < len(txDesc.TxOutDescs); i++ {
		cryptoAddressData := txDesc.TxOutDescs[i].AbelAddress.GetCryptoAddress().Data()
		coinValue := uint64(txDesc.TxOutDescs[i].CoinValue)
		txRequestOutputDesc := api.NewTxRequestOutputDesc(cryptoAddressData, coinValue)
		txRequestOutputDescs = append(txRequestOutputDescs, txRequestOutputDesc)
	}

	// Call API to build the serializedTxRequestDesc.
	serializedTxRequestDesc, err := api.BuildTransferTxRequestDescFromTxoRings(
		outPointsToSpend,
		serializedTxoRings,
		txRequestOutputDescs,
		uint64(txDesc.TxFee),
		txDesc.TxMemo,
	)
	if err != nil {
		sdkLog.Errorf("fail to build tranasction request from blocks: %v", err)
		return nil, err
	}

	return &UnsignedRawTx{
		Data: serializedTxRequestDesc,
	}, nil
}

// GenerateSignedRawTx signs the unsigned transaction using specified account
func GenerateSignedRawTx(unsignedRawTx *UnsignedRawTx, signerAccounts []Account) (*SignedRawTx, error) {
	if len(signerAccounts) == 0 {
		return nil, fmt.Errorf("no singer specified")
	}
	firstAccountType := signerAccounts[0].AccountType()
	for i := 1; i < len(signerAccounts); i++ {
		if signerAccounts[i].AccountType() != firstAccountType {
			sdkLog.Errorf("all specified account must be the same type")
			return nil, fmt.Errorf("all specified account must be the same type")
		}
	}

	var serializedTxFull []byte
	var txid *api.TxId
	var err error
	switch firstAccountType {
	case AccountTypeSeeds:
		seeds := make([]*api.CryptoRootSeed, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*RootSeedAccount)
			seeds = append(seeds, api.NewRootSeed(
				signerViewAccount.cryptoScheme,
				signerViewAccount.privacyLevel,
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}
		txVersion := api.TxVersion
		switchToMLP, _ := os.LookupEnv("ABELIAN_SDK_SWITCH_TX_MLPAUT_VERSION")
		if switchToMLP != "" {
			txVersion = api.TxVersionV2
		}

		serializedTxFull, txid, err = api.CreateTransferTxByRootSeed(txVersion, unsignedRawTx.Data, seeds)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by root seed: %v", err)
			return nil, err
		}
	case AccountTypeKeys:
		// Prepare cryptoKeys.
		cryptoKeys := make([]*api.CryptoKey, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*CryptoKeysAccount)
			cryptoKeys = append(cryptoKeys, api.NewCryptoKey(
				signerViewAccount.cryptoAddress.Data(),
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}

		txVersion := api.TxVersion
		switchToMLP, _ := os.LookupEnv("ABELIAN_SDK_SWITCH_TX_MLPAUT_VERSION")
		if switchToMLP != "" {
			txVersion = api.TxVersionV2
		}

		// Call API to create the signed raw tx.
		serializedTxFull, txid, err = api.CreateTransferTxByCryptoKeys(txVersion, unsignedRawTx.Data, cryptoKeys)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by crypto keys: %v", err)
			return nil, err
		}
	default:
		return nil, ErrInvalidAccountType
	}

	return &SignedRawTx{
		Data: serializedTxFull,
		TxID: txid.String(),
	}, nil
}

func GenerateSignedRawTxForCTAUT(unsignedRawTx *UnsignedRawTx, signerAccounts []Account) (*SignedRawTx, error) {
	if len(signerAccounts) == 0 {
		return nil, fmt.Errorf("no singer specified")
	}
	firstAccountType := signerAccounts[0].AccountType()
	for i := 1; i < len(signerAccounts); i++ {
		if signerAccounts[i].AccountType() != firstAccountType {
			sdkLog.Errorf("all specified account must be the same type")
			return nil, fmt.Errorf("all specified account must be the same type")
		}
	}

	var serializedTxFull []byte
	var txid *api.TxId
	var err error
	switch firstAccountType {
	case AccountTypeSeeds:
		seeds := make([]*api.CryptoRootSeed, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*RootSeedAccount)
			seeds = append(seeds, api.NewRootSeed(
				signerViewAccount.cryptoScheme,
				signerViewAccount.privacyLevel,
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}
		serializedTxFull, txid, err = api.CreateTransferTxByRootSeedForCTAUT(unsignedRawTx.Data, unsignedRawTx.AutWitness, seeds)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by root seed: %v", err)
			return nil, err
		}
	case AccountTypeKeys:
		// Prepare cryptoKeys.
		cryptoKeys := make([]*api.CryptoKey, 0, len(signerAccounts))
		for i := 0; i < len(signerAccounts); i++ {
			coinSerialNumberKeyMaterial, coinValueKeyMaterial, coinDetectorKeyMaterial := signerAccounts[i].ViewKeyMaterial()
			coinSpendSecretKeyMaterial := signerAccounts[i].SpendKeyMaterial()
			signerViewAccount := signerAccounts[i].(*CryptoKeysAccount)
			cryptoKeys = append(cryptoKeys, api.NewCryptoKey(
				signerViewAccount.cryptoAddress.Data(),
				coinSpendSecretKeyMaterial,
				coinSerialNumberKeyMaterial,
				coinValueKeyMaterial,
				coinDetectorKeyMaterial,
			))
		}

		txVersion := api.TxVersion
		//switchToMLP, _ := os.LookupEnv("ABELIAN_SDK_SWITCH_TX_MLPAUT_VERSION")
		//if switchToMLP != "" {
		//	txVersion = api.TxVersionV2
		//}
		// Call API to create the signed raw tx.
		serializedTxFull, txid, err = api.CreateTransferTxByCryptoKeys(txVersion, unsignedRawTx.Data, cryptoKeys)
		if err != nil {
			sdkLog.Errorf("fail to create transfer tx by crypto keys: %v", err)
			return nil, err
		}
	default:
		return nil, ErrInvalidAccountType
	}

	return &SignedRawTx{
		Data: serializedTxFull,
		TxID: txid.String(),
	}, nil
}
func getSerializedBlocksForRingGroup(ringBlockDescs map[int64]*TxBlockDesc) [][]byte {
	heights := make([]int64, 0, len(ringBlockDescs))
	for height := range ringBlockDescs {
		heights = append(heights, height)
	}

	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	serializedBlocksForRingGroup := make([][]byte, 0, len(ringBlockDescs))
	for i := 0; i < len(heights); i++ {
		serializedBlocksForRingGroup = append(serializedBlocksForRingGroup, ringBlockDescs[heights[i]].BinData)
	}

	return serializedBlocksForRingGroup
}
