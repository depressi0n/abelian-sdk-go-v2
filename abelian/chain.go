package abelian

import (
	"fmt"

	api "github.com/abesuite/abec/sdkapi/v2"
)

type NetworkID uint8

const (
	MainNet       NetworkID = 0
	RegressionNet NetworkID = 1
	TestNet       NetworkID = 2
	SimNet        NetworkID = 3
)

func (n NetworkID) String() string {
	switch n {
	case MainNet:
		return "mainnet"
	case RegressionNet:
		return "regressionnet"
	case TestNet:
		return "testnet"
	case SimNet:
		return "simnet"
	default:
		return "unknown"
	}
}

var NetName2NetID = map[string]NetworkID{
	"mainnet":       MainNet,
	"regressionnet": RegressionNet,
	"testnet":       TestNet,
	"simnet":        SimNet,
}

func GetTxoRingSizeByBlockHeight(height int64) uint8 {
	return api.GetTxoRingSizeByBlockHeight(int32(height))
}

func GetBlockNumPerRingGroupByBlockHeight(height int64) uint8 {
	return api.GetBlockNumPerRingGroupByBlockHeight(int32(height))
}

func GetCoinbaseMaturity() int64 {
	return 200
}

func GetRingBlockHeights(height int64) []int64 {
	blockNumPerGroup := GetBlockNumPerRingGroupByBlockHeight(height)
	firstRingBlockHeight := height - height%int64(blockNumPerGroup)

	ringBlockHeights := make([]int64, 0, blockNumPerGroup)
	for i := int64(0); i < int64(blockNumPerGroup); i++ {
		ringBlockHeights = append(ringBlockHeights, firstRingBlockHeight+i)
	}
	return ringBlockHeights
}

func GetRingBlockGroupByHeight(client *Client, height int64) ([][]byte, error) {
	blockNumPerGroup := GetBlockNumPerRingGroupByBlockHeight(height)
	firstRingBlockHeight := height - height%int64(blockNumPerGroup)

	serializedBlockGroups := make([][]byte, 0, blockNumPerGroup)
	for i := int64(0); i < int64(blockNumPerGroup); i++ {
		blockBytes, err := client.GetBlockBytesByHeight(firstRingBlockHeight + i)
		if err != nil {
			return nil, fmt.Errorf("fail to get block group: %v", err)
		}
		serializedBlockGroups = append(serializedBlockGroups, blockBytes)
	}
	return serializedBlockGroups, nil
}

func EstimateTxFee(txinDescs interface{}, txOutDescs []*TxOutDesc) int64 {
	return 1_000_000
}

func NeutrinoToAbel(neutrinoAmount int64) float64 {
	return float64(neutrinoAmount) / 1e7
}

func AbelToNeutrino(abelAmount float64) int64 {
	return int64(abelAmount * 1e7)
}

// CoinIDRing is the porting of wire.OutPointRing to avoid using specific concepts, but generalize them.
type CoinIDRing struct {
	Version  uint32
	BlockIDs []string
	CoinIDs  []*CoinID
}

func (ring *CoinIDRing) RingId() (string, error) {
	outpointRing, err := coinIDRing2OutPointRing(ring)
	if err != nil {
		return "", err
	}
	return outpointRing.RingId()
}
func coinIDRing2OutPointRing(coinIDRing *CoinIDRing) (*api.OutPointRing, error) {
	outpointRing := &api.OutPointRing{
		Version:   coinIDRing.Version,
		BlockIDs:  coinIDRing.BlockIDs,
		OutPoints: make([]*api.OutPoint, len(coinIDRing.CoinIDs)),
	}
	var err error
	for i, coinID := range coinIDRing.CoinIDs {
		outpointRing.OutPoints[i], err = api.NewOutPointFromTxIdStr(coinID.TxID, coinID.Index)
		if err != nil {
			return nil, err
		}
	}
	return outpointRing, nil
}

func outPointRing2CoinIDRing(outPointRing *api.OutPointRing) (*CoinIDRing, error) {
	coinIDs := make([]*CoinID, len(outPointRing.OutPoints))
	for i, outPoint := range outPointRing.OutPoints {
		coinIDs[i] = &CoinID{
			TxID:  outPoint.TxId.String(),
			Index: outPoint.Index,
		}
	}
	return &CoinIDRing{
		Version:  outPointRing.Version,
		BlockIDs: outPointRing.BlockIDs,
		CoinIDs:  coinIDs,
	}, nil
}
func BuildCoinRings(serializedBlocksForRingGroup [][]byte) ([]*CoinRing, error) {
	txoRings, err := api.BuildTxoRingsFromRingBlocks(serializedBlocksForRingGroup)
	if err != nil {
		return nil, err
	}
	coinRings := make([]*CoinRing, len(txoRings))
	for i := 0; i < len(txoRings); i++ {
		coinRings[i], err = apiTxoRing2CoinRing(txoRings[i])
		if err != nil {
			return nil, err
		}
	}

	return coinRings, nil
}
func BuildCoinIDRings(serializedBlocksForRingGroup [][]byte) ([]*CoinIDRing, error) {
	coinRings, err := BuildCoinRings(serializedBlocksForRingGroup)
	if err != nil {
		return nil, err
	}
	res := make([]*CoinIDRing, len(coinRings))
	for i := 0; i < len(coinRings); i++ {
		res[i] = coinRings[i].CoinIDRing
	}
	return res, nil
}
