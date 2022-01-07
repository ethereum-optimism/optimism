package sequencer

import (
	"bytes"
	"encoding/binary"
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

var byteOrder = binary.BigEndian

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

// Write encodes the BatchContext into a 16-byte stream using the following
// encoding:
//  - num_sequenced_txs:        3 bytes
//  - num_subsequent_queue_txs: 3 bytes
//  - timestamp:                5 bytes
//  - block_number:             5 bytes
func (c *BatchContext) Write(w *bytes.Buffer) {
	writeUint64(w, c.NumSequencedTxs, 3)
	writeUint64(w, c.NumSubsequentQueueTxs, 3)
	writeUint64(w, c.Timestamp, 5)
	writeUint64(w, c.BlockNumber, 5)
}

// Read decodes the BatchContext from the passed reader. If fewer than 16-bytes
// remain, an error is returned. Otherwise the first 16-bytes will be read using
// the expected encoding:
//  - num_sequenced_txs:        3 bytes
//  - num_subsequent_queue_txs: 3 bytes
//  - timestamp:                5 bytes
//  - block_number:             5 bytes
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
	// (ommitted) queued txs are in a given window.
	Contexts []BatchContext

	// Txs contains all sequencer txs that will be recorded in the L1 CTC
	// contract.
	Txs []*CachedTx
}

// Write encodes the AppendSequencerBatchParams using the following format:
//  - should_start_at_element:        5 bytes
//  - total_elements_to_append:       3 bytes
//  - num_contexts:                   3 bytes
//    - num_contexts * batch_context: num_contexts * 16 bytes
//  - [num txs ommitted]
//    - tx_len:                       3 bytes
//    - tx_bytes:                     tx_len bytes
func (p *AppendSequencerBatchParams) Write(w *bytes.Buffer) error {
	writeUint64(w, p.ShouldStartAtElement, 5)
	writeUint64(w, p.TotalElementsToAppend, 3)

	// Write number of contexts followed by each fixed-size BatchContext.
	writeUint64(w, uint64(len(p.Contexts)), 3)
	for _, context := range p.Contexts {
		context.Write(w)
	}

	// Write each length-prefixed tx.
	for _, tx := range p.Txs {
		writeUint64(w, uint64(tx.Size()), TxLenSize)
		_, _ = w.Write(tx.RawTx()) // can't fail for bytes.Buffer
	}

	return nil
}

// Serialize performs the same encoding as Write, but returns the resulting
// bytes slice.
func (p *AppendSequencerBatchParams) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	if err := p.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Read decodes the AppendSequencerBatchParams from a bytes stream. If the byte
// stream does not terminate cleanly with an EOF while reading a tx_len, this
// method will return an error. Otherwise, the stream will be parsed according
// to the following format:
//  - should_start_at_element:        5 bytes
//  - total_elements_to_append:       3 bytes
//  - num_contexts:                   3 bytes
//    - num_contexts * batch_context: num_contexts * 16 bytes
//  - [num txs ommitted]
//    - tx_len:                       3 bytes
//    - tx_bytes:                     tx_len bytes
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

	for i := uint64(0); i < numContexts; i++ {
		var batchContext BatchContext
		if err := batchContext.Read(r); err != nil {
			return err
		}

		p.Contexts = append(p.Contexts, batchContext)
	}

	// Deserialize any transactions. Since the number of txs is ommitted
	// from the encoding, loop until the stream is consumed.
	for {
		var txLen uint64
		err := readUint64(r, &txLen, TxLenSize)
		// Getting an EOF when reading the txLen expected for a cleanly
		// encoded object. Silece the error and return success.
		if err == io.EOF {
			return nil
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
func writeUint64(w *bytes.Buffer, val uint64, n uint) {
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
	_, _ = w.Write(buf[8-n:]) // can't fail for bytes.Buffer
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
