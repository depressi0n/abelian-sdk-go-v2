package abelian

import (
	"encoding/json"
	"fmt"
)

type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      string        `json:"id"`
}
type RPCError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e RPCError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

type JSONRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
	ID     string          `json:"id"`
}

type ChainInfo struct {
	NumBlocks       int64   `json:"blocks"`
	IsTestnet       bool    `json:"testnet"`
	Version         int64   `json:"version"`
	ProtocolVersion int64   `json:"protocolversion"`
	RelayFee        float64 `json:"relayfee"`
	NetID           uint8   `json:"netid"`
}

type Block struct {
	Height        int64    `json:"height"`
	Confirmations int64    `json:"confirmations"`
	Version       int64    `json:"version"`
	VersionHex    string   `json:"versionHex"`
	Time          int64    `json:"time"`
	Nonce         uint64   `json:"nonce"`
	Size          int64    `json:"size"`
	FullSize      int64    `json:"fullsize"`
	Difficulty    float64  `json:"difficulty"`
	BlockHash     string   `json:"hash"`
	PrevBlockHash string   `json:"previousblockhash"`
	NextBlockHash string   `json:"nextblockhash"`
	ContentHash   string   `json:"contenthash"`
	MerkleRoot    string   `json:"merkleroot"`
	Bits          string   `json:"bits"`
	SealHash      string   `json:"sealhash"`
	Mixdigest     string   `json:"mixdigest"`
	TxHashes      []string `json:"tx"`
	RawTxs        []*Tx    `json:"rawTx"`
}

type TxVin struct {
	TXORing      TXORing `json:"prevutxoring"`
	SerialNumber string  `json:"serialnumber"`
}

type OutPoint struct {
	TxHash string `json:"txid"`
	Index  uint8  `json:"index"`
}

type TXORing struct {
	Version     int64      `json:"version"`
	BlockHashes []string   `json:"blockhashs"`
	OutPoints   []OutPoint `json:"outpoints"`
}

type TxVout struct {
	N      int64  `json:"n"`
	Script string `json:"script"`
}

type Tx struct {
	Hex           string    `json:"hex"`
	TxID          string    `json:"txid"`
	TxHash        string    `json:"hash"`
	Time          int64     `json:"time"`
	BlockHash     string    `json:"blockhash"`
	BlockTime     int64     `json:"blocktime"`
	Confirmations int64     `bson:"confirmations"`
	Version       uint32    `json:"version"`
	Size          int64     `json:"size"`
	FullSize      int64     `json:"fullsize"`
	Memo          string    `json:"memo"`
	Fee           float64   `json:"fee"`
	Witness       string    `json:"witness"`
	Vin           []*TxVin  `json:"vin"`
	Vout          []*TxVout `json:"vout"`
}

type AutMetadata struct {
	Version                    uint32   `json:"version"`
	AutIdentifier              string   `json:"identifier"`
	AutName                    string   `json:"name"`
	AutSymbol                  string   `json:"symbol"`
	BaseUnitName               string   `json:"baseUnitName"`
	SubUnitName                string   `json:"subUnitName"`
	UnitScale                  uint64   `json:"unitScale"`
	AutMemo                    string   `json:"autMemo"`
	PlannedTotalSupply         uint64   `json:"plannedTotalSupply"`
	Issuers                    []string `json:"issuers"`
	ReregistrationExpireHeight int32    `json:"reRegistrationHeight"`
	ReRegistrationThreshold    uint8    `json:"reRegistrationThreshold"`
	MintThreshold              uint8    `json:"mintThreshold"`
	PrivacyType                uint8    `json:"privacyType"`

	MintedAmount         uint64     `json:"mintedAmount"`
	BurnedAmount         uint64     `json:"burnedAmount"`
	UpdateScriptVersions []uint32   `json:"updateScriptVersions"`
	ActiveRootTokenSet   []OutPoint `json:"activeRootTokenSet"`
}
