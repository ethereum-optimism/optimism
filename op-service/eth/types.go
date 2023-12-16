package eth

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

type ErrorCode int

const (
	UnknownPayload           ErrorCode = -32001 // Payload does not exist / is not available.
	InvalidForkchoiceState   ErrorCode = -38002 // Forkchoice state is invalid / inconsistent.
	InvalidPayloadAttributes ErrorCode = -38003 // Payload attributes are invalid / inconsistent.
)

// InputError distinguishes an user-input error from regular rpc errors,
// to help the (Engine) API user divert from accidental input mistakes.
type InputError struct {
	Inner error
	Code  ErrorCode
}

func (ie InputError) Error() string {
	return fmt.Sprintf("input error %d: %s", ie.Code, ie.Inner.Error())
}

func (ie InputError) Unwrap() error {
	return ie.Inner
}

// Is checks if the error is the given target type.
// Any type of InputError counts, regardless of code.
func (ie InputError) Is(target error) bool {
	_, ok := target.(InputError)
	return ok // we implement Unwrap, so we do not have to check the inner type now
}

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

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b Bytes32) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[29:])
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

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b Bytes256) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[253:])
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

type PayloadID = engine.PayloadID

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutionPayload `json:"executionPayload"`
}

type ExecutionPayload struct {
	ParentHash    common.Hash     `json:"parentHash"`
	FeeRecipient  common.Address  `json:"feeRecipient"`
	StateRoot     Bytes32         `json:"stateRoot"`
	ReceiptsRoot  Bytes32         `json:"receiptsRoot"`
	LogsBloom     Bytes256        `json:"logsBloom"`
	PrevRandao    Bytes32         `json:"prevRandao"`
	BlockNumber   Uint64Quantity  `json:"blockNumber"`
	GasLimit      Uint64Quantity  `json:"gasLimit"`
	GasUsed       Uint64Quantity  `json:"gasUsed"`
	Timestamp     Uint64Quantity  `json:"timestamp"`
	ExtraData     BytesMax32      `json:"extraData"`
	BaseFeePerGas Uint256Quantity `json:"baseFeePerGas"`
	BlockHash     common.Hash     `json:"blockHash"`
	// nil if not present, pre-shanghai
	Withdrawals *types.Withdrawals `json:"withdrawals,omitempty"`
	// Array of transaction objects, each object is a byte list (DATA) representing
	// TransactionType || TransactionPayload or LegacyTransaction as defined in EIP-2718
	Transactions []Data `json:"transactions"`
}

func (payload *ExecutionPayload) ID() BlockID {
	return BlockID{Hash: payload.BlockHash, Number: uint64(payload.BlockNumber)}
}

func (payload *ExecutionPayload) ParentID() BlockID {
	n := uint64(payload.BlockNumber)
	if n > 0 {
		n -= 1
	}
	return BlockID{Hash: payload.ParentHash, Number: n}
}

type rawTransactions []Data

func (s rawTransactions) Len() int { return len(s) }
func (s rawTransactions) EncodeIndex(i int, w *bytes.Buffer) {
	w.Write(s[i])
}

func (payload *ExecutionPayload) CanyonBlock() bool {
	return payload.Withdrawals != nil
}

// CheckBlockHash recomputes the block hash and returns if the embedded block hash matches.
func (payload *ExecutionPayload) CheckBlockHash() (actual common.Hash, ok bool) {
	hasher := trie.NewStackTrie(nil)
	txHash := types.DeriveSha(rawTransactions(payload.Transactions), hasher)

	header := types.Header{
		ParentHash:  payload.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    payload.FeeRecipient,
		Root:        common.Hash(payload.StateRoot),
		TxHash:      txHash,
		ReceiptHash: common.Hash(payload.ReceiptsRoot),
		Bloom:       types.Bloom(payload.LogsBloom),
		Difficulty:  common.Big0, // zeroed, proof-of-work legacy
		Number:      big.NewInt(int64(payload.BlockNumber)),
		GasLimit:    uint64(payload.GasLimit),
		GasUsed:     uint64(payload.GasUsed),
		Time:        uint64(payload.Timestamp),
		Extra:       payload.ExtraData,
		MixDigest:   common.Hash(payload.PrevRandao),
		Nonce:       types.BlockNonce{}, // zeroed, proof-of-work legacy
		BaseFee:     payload.BaseFeePerGas.ToBig(),
	}

	if payload.CanyonBlock() {
		withdrawalHash := types.DeriveSha(*payload.Withdrawals, hasher)
		header.WithdrawalsHash = &withdrawalHash
	}

	blockHash := header.Hash()
	return blockHash, blockHash == payload.BlockHash
}

func BlockAsPayload(bl *types.Block, canyonForkTime *uint64) (*ExecutionPayload, error) {
	baseFee, overflow := uint256.FromBig(bl.BaseFee())
	if overflow {
		return nil, fmt.Errorf("invalid base fee in block: %s", bl.BaseFee())
	}
	opaqueTxs := make([]Data, len(bl.Transactions()))
	for i, tx := range bl.Transactions() {
		otx, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("tx %d failed to marshal: %w", i, err)
		}
		opaqueTxs[i] = otx
	}

	payload := &ExecutionPayload{
		ParentHash:    bl.ParentHash(),
		FeeRecipient:  bl.Coinbase(),
		StateRoot:     Bytes32(bl.Root()),
		ReceiptsRoot:  Bytes32(bl.ReceiptHash()),
		LogsBloom:     Bytes256(bl.Bloom()),
		PrevRandao:    Bytes32(bl.MixDigest()),
		BlockNumber:   Uint64Quantity(bl.NumberU64()),
		GasLimit:      Uint64Quantity(bl.GasLimit()),
		GasUsed:       Uint64Quantity(bl.GasUsed()),
		Timestamp:     Uint64Quantity(bl.Time()),
		ExtraData:     bl.Extra(),
		BaseFeePerGas: *baseFee,
		BlockHash:     bl.Hash(),
		Transactions:  opaqueTxs,
	}

	if canyonForkTime != nil && uint64(payload.Timestamp) >= *canyonForkTime {
		payload.Withdrawals = &types.Withdrawals{}
	}

	return payload, nil
}

type PayloadAttributes struct {
	// value for the timestamp field of the new payload
	Timestamp Uint64Quantity `json:"timestamp"`
	// value for the random field of the new payload
	PrevRandao Bytes32 `json:"prevRandao"`
	// suggested value for the coinbase field of the new payload
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"`
	// Withdrawals to include into the block -- should be nil or empty depending on Shanghai enablement
	Withdrawals *types.Withdrawals `json:"withdrawals,omitempty"`
	// Transactions to force into the block (always at the start of the transactions list).
	Transactions []Data `json:"transactions,omitempty"`
	// NoTxPool to disable adding any transactions from the transaction-pool.
	NoTxPool bool `json:"noTxPool,omitempty"`
	// GasLimit override
	GasLimit *Uint64Quantity `json:"gasLimit,omitempty"`
}

type ExecutePayloadStatus string

const (
	// given payload is valid
	ExecutionValid ExecutePayloadStatus = "VALID"
	// given payload is invalid
	ExecutionInvalid ExecutePayloadStatus = "INVALID"
	// sync process is in progress
	ExecutionSyncing ExecutePayloadStatus = "SYNCING"
	// returned if the payload is not fully validated, and does not extend the canonical chain,
	// but will be remembered for later (on reorgs or sync updates and such)
	ExecutionAccepted ExecutePayloadStatus = "ACCEPTED"
	// if the block-hash in the payload is not correct
	ExecutionInvalidBlockHash ExecutePayloadStatus = "INVALID_BLOCK_HASH"
	// proof-of-stake transition only, not used in rollup
	ExecutionInvalidTerminalBlock ExecutePayloadStatus = "INVALID_TERMINAL_BLOCK"
)

type PayloadStatusV1 struct {
	// the result of the payload execution
	Status ExecutePayloadStatus `json:"status"`
	// the hash of the most recent valid block in the branch defined by payload and its ancestors (optional field)
	LatestValidHash *common.Hash `json:"latestValidHash,omitempty"`
	// additional details on the result (optional field)
	ValidationError *string `json:"validationError,omitempty"`
}

type ForkchoiceState struct {
	// block hash of the head of the canonical chain
	HeadBlockHash common.Hash `json:"headBlockHash"`
	// safe block hash in the canonical chain
	SafeBlockHash common.Hash `json:"safeBlockHash"`
	// block hash of the most recent finalized block
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}

type ForkchoiceUpdatedResult struct {
	// the result of the payload execution
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	// the payload id if requested
	PayloadID *PayloadID `json:"payloadId"`
}

// SystemConfig represents the rollup system configuration that carries over in every L2 block,
// and may be changed through L1 system config events.
// The initial SystemConfig at rollup genesis is embedded in the rollup configuration.
type SystemConfig struct {
	// BatcherAddr identifies the batch-sender address used in batch-inbox data-transaction filtering.
	BatcherAddr common.Address `json:"batcherAddr"`
	// Overhead identifies the L1 fee overhead, and is passed through opaquely to op-geth.
	Overhead Bytes32 `json:"overhead"`
	// Scalar identifies the L1 fee scalar, and is passed through opaquely to op-geth.
	Scalar Bytes32 `json:"scalar"`
	// GasLimit identifies the L2 block gas limit
	GasLimit uint64 `json:"gasLimit"`
	// More fields can be added for future SystemConfig versions.
}

type Bytes48 [48]byte

func (b *Bytes48) UnmarshalJSON(text []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Bytes48) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("Bytes32", text, b[:])
}

func (b Bytes48) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b Bytes48) String() string {
	return hexutil.Encode(b[:])
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b Bytes48) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[45:])
}

// Uint64String is a decimal string representation of an uint64, for usage in the Beacon API JSON encoding
type Uint64String uint64

func (v Uint64String) MarshalText() (out []byte, err error) {
	out = strconv.AppendUint(out, uint64(v), 10)
	return
}

func (v *Uint64String) UnmarshalText(b []byte) error {
	n, err := strconv.ParseUint(string(b), 0, 64)
	if err != nil {
		return err
	}
	*v = Uint64String(n)
	return nil
}
