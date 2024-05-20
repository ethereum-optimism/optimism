package eth

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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

var ErrBedrockScalarPaddingNotEmpty = errors.New("version 0 scalar value has non-empty padding")

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

type Bytes96 [96]byte

func (b *Bytes96) UnmarshalJSON(text []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(b), text, b[:])
}

func (b *Bytes96) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("Bytes96", text, b[:])
}

func (b Bytes96) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b Bytes96) String() string {
	return hexutil.Encode(b[:])
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (b Bytes96) TerminalString() string {
	return fmt.Sprintf("%x..%x", b[:3], b[93:])
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

type Uint256Quantity = hexutil.U256

type Data = hexutil.Bytes

type (
	PayloadID   = engine.PayloadID
	PayloadInfo struct {
		ID        PayloadID
		Timestamp uint64
	}
)

type ExecutionPayloadEnvelope struct {
	ParentBeaconBlockRoot *common.Hash      `json:"parentBeaconBlockRoot,omitempty"`
	ExecutionPayload      *ExecutionPayload `json:"executionPayload"`
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
	// Array of transaction objects, each object is a byte list (DATA) representing
	// TransactionType || TransactionPayload or LegacyTransaction as defined in EIP-2718
	Transactions []Data `json:"transactions"`
	// Nil if not present (Bedrock)
	Withdrawals *types.Withdrawals `json:"withdrawals,omitempty"`
	// Nil if not present (Bedrock, Canyon, Delta)
	BlobGasUsed *Uint64Quantity `json:"blobGasUsed,omitempty"`
	// Nil if not present (Bedrock, Canyon, Delta)
	ExcessBlobGas *Uint64Quantity `json:"excessBlobGas,omitempty"`
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
func (envelope *ExecutionPayloadEnvelope) CheckBlockHash() (actual common.Hash, ok bool) {
	payload := envelope.ExecutionPayload

	hasher := trie.NewStackTrie(nil)
	txHash := types.DeriveSha(rawTransactions(payload.Transactions), hasher)

	header := types.Header{
		ParentHash:       payload.ParentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         payload.FeeRecipient,
		Root:             common.Hash(payload.StateRoot),
		TxHash:           txHash,
		ReceiptHash:      common.Hash(payload.ReceiptsRoot),
		Bloom:            types.Bloom(payload.LogsBloom),
		Difficulty:       common.Big0, // zeroed, proof-of-work legacy
		Number:           big.NewInt(int64(payload.BlockNumber)),
		GasLimit:         uint64(payload.GasLimit),
		GasUsed:          uint64(payload.GasUsed),
		Time:             uint64(payload.Timestamp),
		Extra:            payload.ExtraData,
		MixDigest:        common.Hash(payload.PrevRandao),
		Nonce:            types.BlockNonce{}, // zeroed, proof-of-work legacy
		BaseFee:          (*uint256.Int)(&payload.BaseFeePerGas).ToBig(),
		ParentBeaconRoot: envelope.ParentBeaconBlockRoot,
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
		BaseFeePerGas: Uint256Quantity(*baseFee),
		BlockHash:     bl.Hash(),
		Transactions:  opaqueTxs,
		ExcessBlobGas: (*Uint64Quantity)(bl.ExcessBlobGas()),
		BlobGasUsed:   (*Uint64Quantity)(bl.BlobGasUsed()),
	}

	if canyonForkTime != nil && uint64(payload.Timestamp) >= *canyonForkTime {
		payload.Withdrawals = &types.Withdrawals{}
	}

	return payload, nil
}

func BlockAsPayloadEnv(bl *types.Block, canyonForkTime *uint64) (*ExecutionPayloadEnvelope, error) {
	payload, err := BlockAsPayload(bl, canyonForkTime)
	if err != nil {
		return nil, err
	}
	return &ExecutionPayloadEnvelope{
		ExecutionPayload:      payload,
		ParentBeaconBlockRoot: bl.BeaconRoot(),
	}, nil
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
	// parentBeaconBlockRoot optional extension in Dencun
	ParentBeaconBlockRoot *common.Hash `json:"parentBeaconBlockRoot,omitempty"`

	// Optimism additions

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
	// Overhead identifies the L1 fee overhead.
	// Pre-Ecotone this is passed as-is to the engine.
	// Post-Ecotone this is always zero, and not passed into the engine.
	Overhead Bytes32 `json:"overhead"`
	// Scalar identifies the L1 fee scalar
	// Pre-Ecotone this is passed as-is to the engine.
	// Post-Ecotone this encodes multiple pieces of scalar data.
	Scalar Bytes32 `json:"scalar"`
	// GasLimit identifies the L2 block gas limit
	GasLimit uint64 `json:"gasLimit"`
	// More fields can be added for future SystemConfig versions.
}

// The Ecotone upgrade introduces a versioned L1 scalar format
// that is backward-compatible with pre-Ecotone L1 scalar values.
const (
	// L1ScalarBedrock is implied pre-Ecotone, encoding just a regular-gas scalar.
	L1ScalarBedrock = byte(0)
	// L1ScalarEcotone is new in Ecotone, allowing configuration of both a regular and a blobs scalar.
	L1ScalarEcotone = byte(1)
)

type EcotoneScalars struct {
	BlobBaseFeeScalar uint32
	BaseFeeScalar     uint32
}

func (sysCfg *SystemConfig) EcotoneScalars() (EcotoneScalars, error) {
	if err := CheckEcotoneL1SystemConfigScalar(sysCfg.Scalar); err != nil {
		if errors.Is(err, ErrBedrockScalarPaddingNotEmpty) {
			// L2 spec mandates we set baseFeeScalar to MaxUint32 if there are non-zero bytes in
			// the padding area.
			return EcotoneScalars{BlobBaseFeeScalar: 0, BaseFeeScalar: math.MaxUint32}, nil
		}
		return EcotoneScalars{}, err
	}
	return DecodeScalar(sysCfg.Scalar)
}

// DecodeScalar decodes the blobBaseFeeScalar and baseFeeScalar from a 32-byte scalar value.
// It uses the first byte to determine the scalar format.
func DecodeScalar(scalar [32]byte) (EcotoneScalars, error) {
	switch scalar[0] {
	case L1ScalarBedrock:
		return EcotoneScalars{
			BlobBaseFeeScalar: 0,
			BaseFeeScalar:     binary.BigEndian.Uint32(scalar[28:32]),
		}, nil
	case L1ScalarEcotone:
		return EcotoneScalars{
			BlobBaseFeeScalar: binary.BigEndian.Uint32(scalar[24:28]),
			BaseFeeScalar:     binary.BigEndian.Uint32(scalar[28:32]),
		}, nil
	default:
		return EcotoneScalars{}, fmt.Errorf("unexpected system config scalar: %x", scalar)
	}
}

// EncodeScalar encodes the EcotoneScalars into a 32-byte scalar value
// for the Ecotone serialization format.
func EncodeScalar(scalars EcotoneScalars) (scalar [32]byte) {
	scalar[0] = L1ScalarEcotone
	binary.BigEndian.PutUint32(scalar[24:28], scalars.BlobBaseFeeScalar)
	binary.BigEndian.PutUint32(scalar[28:32], scalars.BaseFeeScalar)
	return
}

func CheckEcotoneL1SystemConfigScalar(scalar [32]byte) error {
	versionByte := scalar[0]
	switch versionByte {
	case L1ScalarBedrock:
		if ([27]byte)(scalar[1:28]) != ([27]byte{}) { // check padding
			return ErrBedrockScalarPaddingNotEmpty
		}
		return nil
	case L1ScalarEcotone:
		if ([23]byte)(scalar[1:24]) != ([23]byte{}) { // check padding
			return fmt.Errorf("invalid version 1 scalar padding: %x", scalar[1:24])
		}
		return nil
	default:
		// ignore the event if it's an unknown scalar format
		return fmt.Errorf("unrecognized scalar version: %d", versionByte)
	}
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

type EngineAPIMethod string

const (
	FCUV1 EngineAPIMethod = "engine_forkchoiceUpdatedV1"
	FCUV2 EngineAPIMethod = "engine_forkchoiceUpdatedV2"
	FCUV3 EngineAPIMethod = "engine_forkchoiceUpdatedV3"

	NewPayloadV2 EngineAPIMethod = "engine_newPayloadV2"
	NewPayloadV3 EngineAPIMethod = "engine_newPayloadV3"

	GetPayloadV2 EngineAPIMethod = "engine_getPayloadV2"
	GetPayloadV3 EngineAPIMethod = "engine_getPayloadV3"
)
