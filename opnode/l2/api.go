// Package l2 connects to the L2 execution engine over the Engine API.
package l2

import (
	"fmt"
	"reflect"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

type ErrorCode int

const (
	UnavailablePayload ErrorCode = -32001
)

type Bytes32 [32]byte

func (b *Bytes32) UnmarshalJSON(text []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Bytes32) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("Bytes32", text, b[:])
}

func (b Bytes32) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b Bytes32) String() string {
	return hexutil.Encode(b[:])
}

type Bytes256 [256]byte

func (b *Bytes256) UnmarshalJSON(text []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Bytes256) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("Bytes32", text, b[:])
}

func (b Bytes256) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b Bytes256) String() string {
	return hexutil.Encode(b[:])
}

type Uint64Quantity = hexutil.Uint64

type BytesMax32 []byte

func (b *BytesMax32) UnmarshalJSON(text []byte) error {
	if len(text) > 64+2+2 { // account for delimiter "", and 0x prefix
		return fmt.Errorf("input too long, expected at most 32 hex-encoded, 0x-prefixed, bytes: %x", text)
	}
	return (*hexutil.Bytes)(b).UnmarshalJSON(text)
}

func (b *BytesMax32) UnmarshalText(text []byte) error {
	if len(text) > 64+2 { // account for 0x prefix
		return fmt.Errorf("input too long, expected at most 32 hex-encoded, 0x-prefixed, bytes: %x", text)
	}
	return (*hexutil.Bytes)(b).UnmarshalText(text)
}

func (b BytesMax32) MarshalText() ([]byte, error) {
	return (hexutil.Bytes)(b).MarshalText()
}

func (b BytesMax32) String() string {
	return hexutil.Encode(b)
}

type Uint256Quantity = uint256.Int

type Data = hexutil.Bytes

// TODO: implement neat 8 byte typed payload ID and upstream it to geth api definitions
type PayloadID = hexutil.Bytes

type ExecutionPayload struct {
	ParentHashField common.Hash     `json:"parentHash"`
	FeeRecipient    common.Address  `json:"feeRecipient"`
	StateRoot       Bytes32         `json:"stateRoot"`
	ReceiptsRoot    Bytes32         `json:"receiptsRoot"`
	LogsBloom       Bytes256        `json:"logsBloom"`
	Random          Bytes32         `json:"random"`
	BlockNumber     Uint64Quantity  `json:"blockNumber"`
	GasLimit        Uint64Quantity  `json:"gasLimit"`
	GasUsed         Uint64Quantity  `json:"gasUsed"`
	Timestamp       Uint64Quantity  `json:"timestamp"`
	ExtraData       BytesMax32      `json:"extraData"`
	BaseFeePerGas   Uint256Quantity `json:"baseFeePerGas"`
	BlockHash       common.Hash     `json:"blockHash"`
	// Array of transaction objects, each object is a byte list (DATA) representing
	// TransactionType || TransactionPayload or LegacyTransaction as defined in EIP-2718
	TransactionsField []Data `json:"transactions"`
}

func (payload *ExecutionPayload) ID() eth.BlockID {
	return eth.BlockID{Hash: payload.BlockHash, Number: uint64(payload.BlockNumber)}
}

// Implement block interface to enable derive.BlockReferences over a payload
// type Block interface {
// 	Hash() common.Hash
// 	NumberU64() uint64
// 	ParentHash() common.Hash
// 	Transactions() types.Transactions
// }

func (payload *ExecutionPayload) Hash() common.Hash {
	return payload.BlockHash
}

func (payload *ExecutionPayload) NumberU64() uint64 {
	return uint64(payload.BlockNumber)
}

func (payload *ExecutionPayload) Time() uint64 {
	return uint64(payload.Timestamp)
}

func (payload *ExecutionPayload) ParentHash() common.Hash {
	return payload.ParentHashField
}

func (payload *ExecutionPayload) Transactions() types.Transactions {
	res := make([]*types.Transaction, len(payload.TransactionsField))
	for i, t := range payload.TransactionsField {
		res[i] = new(types.Transaction)
		err := res[i].UnmarshalBinary(t)
		if err != nil {
			panic(err)
		}
	}
	return res
}

type PayloadAttributes struct {
	// value for the timestamp field of the new payload
	Timestamp Uint64Quantity `json:"timestamp"`
	// value for the random field of the new payload
	Random Bytes32 `json:"random"`
	// suggested value for the coinbase field of the new payload
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"`
	// Transactions to force into the block (always at the start of the transactions list).
	Transactions []Data `json:"transactions,omitempty"`
	// NoTxPool to disable adding any transactions from the transaction-pool.
	NoTxPool bool `json:"noTxPool,omitempty"`
}

type ExecutePayloadStatus string

const (
	// given payload is valid
	ExecutionValid ExecutePayloadStatus = "VALID"
	// given payload is invalid
	ExecutionInvalid ExecutePayloadStatus = "INVALID"
	// sync process is in progress
	ExecutionSyncing ExecutePayloadStatus = "SYNCING"
)

type ExecutePayloadResult struct {
	// the result of the payload execution
	Status ExecutePayloadStatus `json:"status"`
	// the hash of the most recent valid block in the branch defined by payload and its ancestors
	LatestValidHash common.Hash `json:"latestValidHash"`
	// additional details on the result
	ValidationError string `json:"validationError"`
}

type ForkchoiceState struct {
	// block hash of the head of the canonical chain
	HeadBlockHash common.Hash `json:"headBlockHash"`
	// safe block hash in the canonical chain
	SafeBlockHash common.Hash `json:"safeBlockHash"`
	// block hash of the most recent finalized block
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}

type ForkchoiceUpdatedStatus string

const (
	// given payload is valid
	UpdateSuccess ForkchoiceUpdatedStatus = "SUCCESS"
	// sync process is in progress
	UpdateSyncing ForkchoiceUpdatedStatus = "SYNCING"
)

type ForkchoiceUpdatedResult struct {
	// the result of the payload execution
	Status ForkchoiceUpdatedStatus `json:"status"`
	// the payload id if requested
	PayloadID *PayloadID `json:"payloadId"`
}
