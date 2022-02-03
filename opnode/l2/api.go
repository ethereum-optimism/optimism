// Package l2 connects to the L2 execution engine over the Catalyst API.
package l2

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/rpc"
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
	ParentHash    common.Hash     `json:"parentHash"`
	FeeRecipient  common.Address  `json:"feeRecipient"`
	StateRoot     Bytes32         `json:"stateRoot"`
	ReceiptsRoot  Bytes32         `json:"receiptsRoot"`
	LogsBloom     Bytes256        `json:"logsBloom"`
	Random        Bytes32         `json:"random"`
	BlockNumber   Uint64Quantity  `json:"blockNumber"`
	GasLimit      Uint64Quantity  `json:"gasLimit"`
	GasUsed       Uint64Quantity  `json:"gasUsed"`
	Timestamp     Uint64Quantity  `json:"timestamp"`
	ExtraData     BytesMax32      `json:"extraData"`
	BaseFeePerGas Uint256Quantity `json:"baseFeePerGas"`
	BlockHash     common.Hash     `json:"blockHash"`
	// Array of transaction objects, each object is a byte list (DATA) representing
	// TransactionType || TransactionPayload or LegacyTransaction as defined in EIP-2718
	Transactions []Data `json:"transactions"`
}

func (payload *ExecutionPayload) ID() eth.BlockID {
	return eth.BlockID{Hash: payload.BlockHash, Number: uint64(payload.BlockNumber)}
}

type PayloadAttributes struct {
	// value for the timestamp field of the new payload
	Timestamp Uint64Quantity `json:"timestamp"`
	// value for the random field of the new payload
	Random Bytes32 `json:"random"`
	// suggested value for the coinbase field of the new payload
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"`
	// Transactions to build the block with, omitted if the local tx pool of the engine should be used instead
	Transactions []Data `json:"transactions,omitempty"`
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

type EngineAPI interface {
	GetPayload(ctx context.Context, payloadId PayloadID) (*ExecutionPayload, error)
	ExecutePayload(ctx context.Context, payload *ExecutionPayload) (*ExecutePayloadResult, error)
	ForkchoiceUpdated(ctx context.Context, state *ForkchoiceState, attr *PayloadAttributes) (ForkchoiceUpdatedResult, error)
	Close()
}

type RPCBackend interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	Close()
}

type EthBackend interface {
	eth.BlockByHashSource
	eth.BlockByNumberSource
	eth.NewHeadSource
}

type EngineClient struct {
	RPCBackend
	EthBackend
	Log log.Logger
}

func (el *EngineClient) GetPayload(ctx context.Context, payloadId PayloadID) (*ExecutionPayload, error) {
	e := el.Log.New("payload_id", payloadId)
	e.Debug("getting payload")
	var result ExecutionPayload
	err := el.CallContext(ctx, &result, "engine_getPayloadV1", payloadId)
	if err != nil {
		e = log.New("payload_id", "err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := ErrorCode(rpcErr.ErrorCode())
			if code != UnavailablePayload {
				e.Warn("unexpected error code in get-payload response", "code", code)
			} else {
				e.Warn("unavailable payload in get-payload request")
			}
		} else {
			e.Error("failed to get payload")
		}
		return nil, err
	}
	e.Debug("Received payload")
	return &result, nil
}

func (el *EngineClient) ExecutePayload(ctx context.Context, payload *ExecutionPayload) (*ExecutePayloadResult, error) {
	e := el.Log.New("block_hash", payload.BlockHash)
	e.Debug("sending payload for execution")
	var result ExecutePayloadResult
	err := el.CallContext(ctx, &result, "engine_executePayloadV1", payload)
	if err != nil {
		e.Error("Payload execution failed", "err", err)
		return nil, err
	}
	e.Debug("Received payload execution result", "status", result.Status, "latestValidHash", result.LatestValidHash, "message", result.ValidationError)
	return &result, nil
}

func (el *EngineClient) ForkchoiceUpdated(ctx context.Context, state *ForkchoiceState, attr *PayloadAttributes) (ForkchoiceUpdatedResult, error) {
	e := el.Log.New("state", state, "attr", attr)
	e.Debug("Sharing forkchoice-updated signal")

	var result ForkchoiceUpdatedResult
	err := el.CallContext(ctx, &result, "engine_forkchoiceUpdatedV1", state, attr)
	if err == nil {
		e.Debug("Shared forkchoice-updated signal")
		if attr != nil {
			e.Debug("Received payload id", "payloadId", result.PayloadID)
		}
		return result, nil
	} else {
		e = e.New("err", err)
		if rpcErr, ok := err.(rpc.Error); ok {
			code := ErrorCode(rpcErr.ErrorCode())
			e.Warn("Unexpected error code in forkchoice-updated response", "code", code)
		} else {
			e.Error("Failed to share forkchoice-updated signal")
		}
		return result, err
	}
}

func (el *EngineClient) Close() {
	el.RPCBackend.Close()
}

func BlockToPayload(bl *types.Block, random Bytes32) (*ExecutionPayload, error) {
	extra := bl.Extra()
	if len(extra) > 32 {
		return nil, fmt.Errorf("eth2 merge spec limits extra data to 32 bytes in payload, got %d", len(extra))
	}
	baseFee, overflow := uint256.FromBig(bl.BaseFee())
	if overflow {
		return nil, fmt.Errorf("overflowing base fee")
	}
	txs := bl.Transactions()
	txsEncoded := make([]Data, 0, len(txs))
	for i, tx := range txs {
		txOpaque, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to encode tx %d", i)
		}
		txsEncoded = append(txsEncoded, txOpaque)
	}
	return &ExecutionPayload{
		ParentHash:    bl.ParentHash(),
		FeeRecipient:  bl.Coinbase(),
		StateRoot:     Bytes32(bl.Root()),
		ReceiptsRoot:  Bytes32(bl.ReceiptHash()),
		LogsBloom:     Bytes256(bl.Bloom()),
		Random:        random,
		BlockNumber:   Uint64Quantity(bl.NumberU64()),
		GasLimit:      Uint64Quantity(bl.GasLimit()),
		GasUsed:       Uint64Quantity(bl.GasUsed()),
		Timestamp:     Uint64Quantity(bl.Time()),
		ExtraData:     BytesMax32(extra),
		BaseFeePerGas: Uint256Quantity(*baseFee),
		BlockHash:     bl.Hash(),
		Transactions:  txsEncoded,
	}, nil
}
