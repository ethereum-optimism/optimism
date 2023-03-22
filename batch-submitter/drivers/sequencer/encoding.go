package sequencer

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
	l2rlp "github.com/ethereum-optimism/optimism/l2geth/rlp"
)

const (
	// TxLenSize is the number of bytes used to represent the size of a
	// serialized sequencer transaction.
	TxLenSize = 3
)

var (
	// byteOrder represents the endiannes used for batch serialization
	byteOrder = binary.BigEndian
	// ErrMalformedBatch represents a batch that is not well formed
	// according to the protocol specification
	ErrMalformedBatch = errors.New("malformed batch")
)

// BatchContext denotes a range of transactions that belong the same batch. It
// is used to compress shared fields that would otherwise be repeated for each
// transaction.
type BatchContext struct {
	// NumSequencedTxs specifies the number of sequencer txs included in
	// the batch.
	NumSequencedTxs uint64 `json:"num_sequenced_txs"`

	// NumSubsequentQueueTxs specifies the number of queued txs included in
	// the batch
	NumSubsequentQueueTxs uint64 `json:"num_subsequent_queue_txs"`

	// Timestamp is the L1 timestamp of the batch.
	Timestamp uint64 `json:"timestamp"`

	// BlockNumber is the L1 BlockNumber of the batch.
	BlockNumber uint64 `json:"block_number"`
}

// IsMarkerContext returns true if the BatchContext is a marker context used to
// specify the encoding format. This is only valid if called on the first
// BatchContext in the calldata.
func (c BatchContext) IsMarkerContext() bool {
	return c.Timestamp == 0
}

// MarkerBatchType returns the BatchType specified by a marker BatchContext.
// The return value is only valid if called on the first BatchContext in the
// calldata and IsMarkerContext returns true.
func (c BatchContext) MarkerBatchType() BatchType {
	switch c.BlockNumber {
	case 0:
		return BatchTypeZlib
	default:
		return BatchTypeLegacy
	}
}

// Write encodes the BatchContext into a 16-byte stream using the following
// encoding:
//   - num_sequenced_txs:        3 bytes
//   - num_subsequent_queue_txs: 3 bytes
//   - timestamp:                5 bytes
//   - block_number:             5 bytes
//
// Note that writing to a bytes.Buffer cannot
// error, so errors are ignored here
func (c *BatchContext) Write(w *bytes.Buffer) {
	_ = writeUint64(w, c.NumSequencedTxs, 3)
	_ = writeUint64(w, c.NumSubsequentQueueTxs, 3)
	_ = writeUint64(w, c.Timestamp, 5)
	_ = writeUint64(w, c.BlockNumber, 5)
}

// Read decodes the BatchContext from the passed reader. If fewer than 16-bytes
// remain, an error is returned. Otherwise the first 16-bytes will be read using
// the expected encoding:
//   - num_sequenced_txs:        3 bytes
//   - num_subsequent_queue_txs: 3 bytes
//   - timestamp:                5 bytes
//   - block_number:             5 bytes
func (c *BatchContext) Read(r io.Reader) error {
	if err := readUint64(r, &c.NumSequencedTxs, 3); err != nil {
		return err
	}
	if err := readUint64(r, &c.NumSubsequentQueueTxs, 3); err != nil {
		return err
	}
	if err := readUint64(r, &c.Timestamp, 5); err != nil {
		return err
	}
	return readUint64(r, &c.BlockNumber, 5)
}

// BatchType represents the type of batch being submitted. When the first
// context in the batch has a timestamp of 0, the blocknumber is interpreted as
// an enum that represets the type.
type BatchType int8

const (
	// BatchTypeLegacy represets the legacy batch type.
	BatchTypeLegacy BatchType = -1

	// BatchTypeZlib represents a batch type where the transaction data is
	// compressed using zlib.
	BatchTypeZlib BatchType = 0
)

// BatchTypeFromString returns the BatchType enum based on a human readable
// string.
func BatchTypeFromString(s string) BatchType {
	switch s {
	case "zlib", "ZLIB":
		return BatchTypeZlib
	case "legacy", "LEGACY":
		return BatchTypeLegacy
	default:
		return BatchTypeLegacy
	}
}

// String implements the Stringer interface for BatchType.
func (b BatchType) String() string {
	switch b {
	case BatchTypeLegacy:
		return "LEGACY"
	case BatchTypeZlib:
		return "ZLIB"
	default:
		return ""
	}
}

// MarkerContext returns the marker context, if any, for the given batch type.
func (b BatchType) MarkerContext() *BatchContext {
	switch b {

	// No marker context for legacy encoding.
	case BatchTypeLegacy:
		return nil

	// Zlib marker context sets block number equal to zero.
	case BatchTypeZlib:
		return &BatchContext{
			Timestamp:   0,
			BlockNumber: 0,
		}

	default:
		return nil
	}
}

// AppendSequencerBatchParams holds the raw data required to submit a batch of
// L2 txs to L1 CTC contract. Rather than encoding the objects using the
// standard ABI encoding, a custom encoding is and provided in the call data to
// optimize for gas fees, since batch submission of L2 txs is a primary cost
// driver.
type AppendSequencerBatchParams struct {
	// ShouldStartAtElement specifies the intended starting sequence number
	// of the provided transaction. Upon submission, this should match the
	// CTC's expected value otherwise the transaction will revert.
	ShouldStartAtElement uint64

	// TotalElementsToAppend indicates the number of L2 txs represented by
	// this batch. This includes both sequencer and queued txs.
	TotalElementsToAppend uint64

	// Contexts aggregates redundant L1 block numbers and L1 timestamps for
	// the txns encoded in the Tx slice. Further, they specify consecutive
	// tx windows in Txs and implicitly allow one to compute how many
	// (omitted) queued txs are in a given window.
	Contexts []BatchContext

	// Txs contains all sequencer txs that will be recorded in the L1 CTC
	// contract.
	Txs []*CachedTx
}

// Write encodes the AppendSequencerBatchParams using the following format:
//   - should_start_at_element:        5 bytes
//   - total_elements_to_append:       3 bytes
//   - num_contexts:                   3 bytes
//   - num_contexts * batch_context: num_contexts * 16 bytes
//   - [num txs omitted]
//   - tx_len:                       3 bytes
//   - tx_bytes:                     tx_len bytes
//
// Typed batches include a dummy context as the first context
// where the timestamp is 0. The blocknumber is interpreted
// as an enum that defines the type. It is impossible to have
// a timestamp of 0 in practice, so this safely can indicate
// that the batch is typed.
// Type 0 batches have a dummy context where the blocknumber is
// set to 0. The transaction data is compressed with zlib before
// submitting the transaction to the chain. The fields should_start_at_element,
// total_elements_to_append, num_contexts and the contexts themselves
// are not altered.
//
// Note that writing to a bytes.Buffer cannot
// error, so errors are ignored here
func (p *AppendSequencerBatchParams) Write(
	w *bytes.Buffer,
	batchType BatchType,
) error {

	_ = writeUint64(w, p.ShouldStartAtElement, 5)
	_ = writeUint64(w, p.TotalElementsToAppend, 3)

	// There must be contexts if there are transactions
	if len(p.Contexts) == 0 && len(p.Txs) != 0 {
		return ErrMalformedBatch
	}

	// copy the contexts as to not malleate the struct
	// when it is a typed batch
	contexts := make([]BatchContext, 0, len(p.Contexts)+1)
	// Add the marker context, if any, for non-legacy encodings.
	markerContext := batchType.MarkerContext()
	if markerContext != nil {
		contexts = append(contexts, *markerContext)
	}
	contexts = append(contexts, p.Contexts...)

	// Write number of contexts followed by each fixed-size BatchContext.
	_ = writeUint64(w, uint64(len(contexts)), 3)
	for _, context := range contexts {
		context.Write(w)
	}

	switch batchType {
	case BatchTypeLegacy:
		// Write each length-prefixed tx.
		for _, tx := range p.Txs {
			_ = writeUint64(w, uint64(tx.Size()), TxLenSize)
			_, _ = w.Write(tx.RawTx()) // can't fail for bytes.Buffer
		}
	case BatchTypeZlib:
		zw := zlib.NewWriter(w)
		for _, tx := range p.Txs {
			if err := writeUint64(zw, uint64(tx.Size()), TxLenSize); err != nil {
				return err
			}
			if _, err := zw.Write(tx.RawTx()); err != nil {
				return err
			}
		}
		if err := zw.Close(); err != nil {
			return err
		}

	default:
		return fmt.Errorf("Unknown batch type: %s", batchType)
	}

	return nil
}

// Serialize performs the same encoding as Write, but returns the resulting
// bytes slice.
func (p *AppendSequencerBatchParams) Serialize(
	batchType BatchType,
) ([]byte, error) {

	var buf bytes.Buffer
	if err := p.Write(&buf, batchType); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Read decodes the AppendSequencerBatchParams from a bytes stream. If the byte
// stream does not terminate cleanly with an EOF while reading a tx_len, this
// method will return an error. Otherwise, the stream will be parsed according
// to the following format:
//   - should_start_at_element:        5 bytes
//   - total_elements_to_append:       3 bytes
//   - num_contexts:                   3 bytes
//   - num_contexts * batch_context: num_contexts * 16 bytes
//   - [num txs omitted]
//   - tx_len:                       3 bytes
//   - tx_bytes:                     tx_len bytes
func (p *AppendSequencerBatchParams) Read(r io.Reader) error {
	if err := readUint64(r, &p.ShouldStartAtElement, 5); err != nil {
		return err
	}
	if err := readUint64(r, &p.TotalElementsToAppend, 3); err != nil {
		return err
	}

	// Read number of contexts and deserialize each one.
	var numContexts uint64
	if err := readUint64(r, &numContexts, 3); err != nil {
		return err
	}

	// Assume that it is a legacy batch at first, this will be overwrritten if
	// we detect a marker context.
	var batchType = BatchTypeLegacy
	// Ensure that contexts is never nil
	p.Contexts = make([]BatchContext, 0)
	for i := uint64(0); i < numContexts; i++ {
		var batchContext BatchContext
		if err := batchContext.Read(r); err != nil {
			return err
		}

		if i == 0 && batchContext.IsMarkerContext() {
			batchType = batchContext.MarkerBatchType()
			continue
		}

		p.Contexts = append(p.Contexts, batchContext)
	}

	// Define a closure to clean up the reader used by the specified encoding.
	var closeReader func() error
	switch batchType {

	// The legacy serialization does not require clsing, so we instatiate a
	// dummy closure.
	case BatchTypeLegacy:
		closeReader = func() error { return nil }

	// The zlib serialization requires decompression before reading the
	// plaintext bytes, and also requires proper cleanup.
	case BatchTypeZlib:
		zr, err := zlib.NewReader(r)
		if err != nil {
			return err
		}
		closeReader = zr.Close

		r = bufio.NewReader(zr)
	}

	// Deserialize any transactions. Since the number of txs is omitted
	// from the encoding, loop until the stream is consumed.
	for {
		var txLen uint64
		err := readUint64(r, &txLen, TxLenSize)
		// Getting an EOF when reading the txLen expected for a cleanly
		// encoded object. Silence the error and return success if
		// the batch is well formed.
		if err == io.EOF {
			if len(p.Contexts) == 0 && len(p.Txs) != 0 {
				return ErrMalformedBatch
			}
			return closeReader()
		} else if err != nil {
			return err
		}

		tx := new(l2types.Transaction)
		if err := tx.DecodeRLP(l2rlp.NewStream(r, txLen)); err != nil {
			return err
		}

		p.Txs = append(p.Txs, NewCachedTx(tx))
	}
}

// writeUint64 writes a the bottom `n` bytes of `val` to `w`.
func writeUint64(w io.Writer, val uint64, n uint) error {
	if n < 1 || n > 8 {
		panic(fmt.Sprintf("invalid number of bytes %d must be 1-8", n))
	}

	const maxUint64 uint64 = math.MaxUint64
	maxVal := maxUint64 >> (8 * (8 - n))
	if val > maxVal {
		panic(fmt.Sprintf("cannot encode %d in %d byte value", val, n))
	}

	var buf [8]byte
	byteOrder.PutUint64(buf[:], val)
	_, err := w.Write(buf[8-n:])
	return err
}

// readUint64 reads `n` bytes from `r` and returns them in the lower `n` bytes
// of `val`.
func readUint64(r io.Reader, val *uint64, n uint) error {
	var buf [8]byte
	if _, err := r.Read(buf[8-n:]); err != nil {
		return err
	}
	*val = byteOrder.Uint64(buf[:])
	return nil
}
