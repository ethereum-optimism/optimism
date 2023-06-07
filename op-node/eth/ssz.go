package eth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
)

// ExecutionPayload is the only SSZ type we have to marshal/unmarshal,
// so instead of importing a SSZ lib we implement the bare minimum.
// This is more efficient than RLP, and matches the L1 consensus-layer encoding of ExecutionPayload.

// All fields (4s are offsets to dynamic data)
const executionPayloadFixedPart = 32 + 20 + 32 + 32 + 256 + 32 + 8 + 8 + 8 + 8 + 4 + 32 + 32 + 4

// MAX_TRANSACTIONS_PER_PAYLOAD in consensus spec
const maxTransactionsPerPayload = 1 << 20

// ErrExtraDataTooLarge occurs when the ExecutionPayload's ExtraData field
// is too large to be properly represented in SSZ.
var ErrExtraDataTooLarge = errors.New("extra data too large")

// The payloads are small enough to read and write at once.
// But this happens often enough that we want to avoid re-allocating buffers for this.
var payloadBufPool = sync.Pool{New: func() any {
	x := make([]byte, 0, 100_000)
	return &x
}}

var ErrBadTransactionOffset = errors.New("transactions offset is smaller than extra data offset, aborting")

func (payload *ExecutionPayload) SizeSSZ() (full uint32) {
	full = executionPayloadFixedPart + uint32(len(payload.ExtraData))
	// One offset to each transaction
	full += uint32(len(payload.Transactions)) * 4
	// Each transaction
	for _, tx := range payload.Transactions {
		full += uint32(len(tx))
	}
	return full
}

// marshalBytes32LE returns the value of z as a 32-byte little-endian array.
func marshalBytes32LE(out []byte, z *Uint256Quantity) {
	_ = out[31] // bounds check hint to compiler
	binary.LittleEndian.PutUint64(out[0:8], z[0])
	binary.LittleEndian.PutUint64(out[8:16], z[1])
	binary.LittleEndian.PutUint64(out[16:24], z[2])
	binary.LittleEndian.PutUint64(out[24:32], z[3])
}

func unmarshalBytes32LE(in []byte, z *Uint256Quantity) {
	_ = in[31] // bounds check hint to compiler
	z[0] = binary.LittleEndian.Uint64(in[0:8])
	z[1] = binary.LittleEndian.Uint64(in[8:16])
	z[2] = binary.LittleEndian.Uint64(in[16:24])
	z[3] = binary.LittleEndian.Uint64(in[24:32])
}

// MarshalSSZ encodes the ExecutionPayload as SSZ type
func (payload *ExecutionPayload) MarshalSSZ(w io.Writer) (n int, err error) {
	// Cast to uint32 to enable 32-bit MIPS support where math.MaxUint32-executionPayloadFixedPart is too big for int
	// In that case, len(payload.ExtraData) can't be longer than an int so this is always false anyway.
	if uint32(len(payload.ExtraData)) > math.MaxUint32-uint32(executionPayloadFixedPart) {
		return 0, ErrExtraDataTooLarge
	}

	scope := payload.SizeSSZ()

	buf := *payloadBufPool.Get().(*[]byte)
	if uint32(cap(buf)) < scope {
		buf = make([]byte, scope)
	} else {
		buf = buf[:scope]
	}
	defer payloadBufPool.Put(&buf)

	offset := uint32(0)
	copy(buf[offset:offset+32], payload.ParentHash[:])
	offset += 32
	copy(buf[offset:offset+20], payload.FeeRecipient[:])
	offset += 20
	copy(buf[offset:offset+32], payload.StateRoot[:])
	offset += 32
	copy(buf[offset:offset+32], payload.ReceiptsRoot[:])
	offset += 32
	copy(buf[offset:offset+256], payload.LogsBloom[:])
	offset += 256
	copy(buf[offset:offset+32], payload.PrevRandao[:])
	offset += 32
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(payload.BlockNumber))
	offset += 8
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(payload.GasLimit))
	offset += 8
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(payload.GasUsed))
	offset += 8
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(payload.Timestamp))
	offset += 8
	// offset to ExtraData
	binary.LittleEndian.PutUint32(buf[offset:offset+4], executionPayloadFixedPart)
	offset += 4
	marshalBytes32LE(buf[offset:offset+32], &payload.BaseFeePerGas)
	offset += 32
	copy(buf[offset:offset+32], payload.BlockHash[:])
	offset += 32
	// offset to Transactions
	binary.LittleEndian.PutUint32(buf[offset:offset+4], executionPayloadFixedPart+uint32(len(payload.ExtraData)))
	offset += 4
	if offset != executionPayloadFixedPart {
		panic("fixed part size is inconsistent")
	}
	// dynamic value 1: ExtraData
	copy(buf[offset:offset+uint32(len(payload.ExtraData))], payload.ExtraData[:])
	offset += uint32(len(payload.ExtraData))
	// dynamic value 2: Transactions
	marshalTransactions(buf[offset:], payload.Transactions)
	return w.Write(buf)
}

func marshalTransactions(out []byte, txs []Data) {
	offset := uint32(0)
	txOffset := uint32(len(txs)) * 4
	for _, tx := range txs {
		binary.LittleEndian.PutUint32(out[offset:offset+4], txOffset)
		offset += 4
		nextTxOffset := txOffset + uint32(len(tx))
		copy(out[txOffset:nextTxOffset], tx)
		txOffset = nextTxOffset
	}
}

// UnmarshalSSZ decodes the ExecutionPayload as SSZ type
func (payload *ExecutionPayload) UnmarshalSSZ(scope uint32, r io.Reader) error {
	if scope < executionPayloadFixedPart {
		return fmt.Errorf("scope too small to decode execution payload: %d", scope)
	}

	buf := *payloadBufPool.Get().(*[]byte)
	if uint32(cap(buf)) < scope {
		buf = make([]byte, scope)
	} else {
		buf = buf[:scope]
	}
	defer payloadBufPool.Put(&buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("failed to read fixed-size part of ExecutionPayload: %w", err)
	}
	offset := uint32(0)
	copy(payload.ParentHash[:], buf[offset:offset+32])
	offset += 32
	copy(payload.FeeRecipient[:], buf[offset:offset+20])
	offset += 20
	copy(payload.StateRoot[:], buf[offset:offset+32])
	offset += 32
	copy(payload.ReceiptsRoot[:], buf[offset:offset+32])
	offset += 32
	copy(payload.LogsBloom[:], buf[offset:offset+256])
	offset += 256
	copy(payload.PrevRandao[:], buf[offset:offset+32])
	offset += 32
	payload.BlockNumber = Uint64Quantity(binary.LittleEndian.Uint64(buf[offset : offset+8]))
	offset += 8
	payload.GasLimit = Uint64Quantity(binary.LittleEndian.Uint64(buf[offset : offset+8]))
	offset += 8
	payload.GasUsed = Uint64Quantity(binary.LittleEndian.Uint64(buf[offset : offset+8]))
	offset += 8
	payload.Timestamp = Uint64Quantity(binary.LittleEndian.Uint64(buf[offset : offset+8]))
	offset += 8
	extraDataOffset := binary.LittleEndian.Uint32(buf[offset : offset+4])
	if extraDataOffset != executionPayloadFixedPart {
		return fmt.Errorf("unexpected extra data offset: %d <> %d", extraDataOffset, executionPayloadFixedPart)
	}
	offset += 4
	unmarshalBytes32LE(buf[offset:offset+32], &payload.BaseFeePerGas)
	offset += 32
	copy(payload.BlockHash[:], buf[offset:offset+32])
	offset += 32
	transactionsOffset := binary.LittleEndian.Uint32(buf[offset : offset+4])
	if transactionsOffset < extraDataOffset {
		return ErrBadTransactionOffset
	}
	offset += 4
	if offset != executionPayloadFixedPart {
		panic("fixed part size is inconsistent")
	}
	if transactionsOffset > extraDataOffset+32 || transactionsOffset > scope {
		return fmt.Errorf("extra-data is too large: %d", transactionsOffset-extraDataOffset)
	}
	extraDataSize := transactionsOffset - extraDataOffset
	payload.ExtraData = make(BytesMax32, extraDataSize)
	copy(payload.ExtraData, buf[extraDataOffset:transactionsOffset])
	txs, err := unmarshalTransactions(buf[transactionsOffset:])
	if err != nil {
		return fmt.Errorf("failed to unmarshal transactions list: %w", err)
	}
	payload.Transactions = txs
	return nil
}

func unmarshalTransactions(in []byte) (txs []Data, err error) {
	scope := uint32(len(in))
	if scope == 0 { // empty txs list
		return make([]Data, 0), nil
	}
	if scope < 4 {
		return nil, fmt.Errorf("not enough scope to read first tx offset: %d", scope)
	}
	offset := uint32(0)
	firstTxOffset := binary.LittleEndian.Uint32(in[offset : offset+4])
	offset += 4
	if firstTxOffset%4 != 0 {
		return nil, fmt.Errorf("invalid first tx offset: %d, not a multiple of offset size", firstTxOffset)
	}
	if firstTxOffset > scope {
		return nil, fmt.Errorf("invalid first tx offset: %d, out of scope %d", firstTxOffset, scope)
	}
	txCount := firstTxOffset / 4
	if txCount > maxTransactionsPerPayload {
		return nil, fmt.Errorf("too many transactions: %d > %d", txCount, maxTransactionsPerPayload)
	}
	txs = make([]Data, txCount)
	currentTxOffset := firstTxOffset
	for i := uint32(0); i < txCount; i++ {
		nextTxOffset := scope
		if i+1 < txCount {
			nextTxOffset = binary.LittleEndian.Uint32(in[offset : offset+4])
			offset += 4
		}
		if nextTxOffset < currentTxOffset || nextTxOffset > scope {
			return nil, fmt.Errorf("tx %d has bad next offset: %d, current is %d, scope is %d", i, nextTxOffset, currentTxOffset, scope)
		}
		currentTxSize := nextTxOffset - currentTxOffset
		txs[i] = make(Data, currentTxSize)
		copy(txs[i], in[currentTxOffset:nextTxOffset])
		currentTxOffset = nextTxOffset
	}
	return txs, nil
}
