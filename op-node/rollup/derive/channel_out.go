package derive

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var ErrNotDepositTx = errors.New("first transaction in block is not a deposit tx")
var ErrTooManyRLPBytes = errors.New("batch would cause RLP bytes to go over limit")

type ChannelOut struct {
	id ChannelID
	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64
	// rlpLength is the uncompressed size of the channel. Must be less than MAX_RLP_BYTES_PER_CHANNEL
	rlpLength int

	// Compressor stage. Write input data to it
	compress *zlib.Writer
	// post compression buffer
	buf bytes.Buffer

	closed bool
}

func (co *ChannelOut) ID() ChannelID {
	return co.id
}

func NewChannelOut() (*ChannelOut, error) {
	c := &ChannelOut{
		id:        ChannelID{}, // TODO: use GUID here instead of fully random data
		frame:     0,
		rlpLength: 0,
	}
	_, err := rand.Read(c.id[:])
	if err != nil {
		return nil, err
	}

	compress, err := zlib.NewWriterLevel(&c.buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	c.compress = compress

	return c, nil
}

// TODO: reuse ChannelOut for performance
func (co *ChannelOut) Reset() error {
	co.frame = 0
	co.rlpLength = 0
	co.buf.Reset()
	co.compress.Reset(&co.buf)
	co.closed = false
	_, err := rand.Read(co.id[:])
	if err != nil {
		return err
	}
	return nil
}

// AddBlock adds a block to the channel. It returns an error
// if there is a problem adding the block. The only sentinel
// error that it returns is ErrTooManyRLPBytes. If this error
// is returned, the channel should be closed and a new one
// should be made.
func (co *ChannelOut) AddBlock(block *types.Block) error {
	if co.closed {
		return errors.New("already closed")
	}
	batch, err := blockToBatch(block)
	if err != nil {
		return err
	}
	// We encode to a temporary buffer to determine the encoded length to
	// ensure that the total size of all RLP elements is less than or equal to MAX_RLP_BYTES_PER_CHANNEL
	var buf bytes.Buffer
	if err := rlp.Encode(&buf, batch); err != nil {
		return err
	}
	if co.rlpLength+buf.Len() > MaxRLPBytesPerChannel {
		return fmt.Errorf("could not add %d bytes to channel of %d bytes, max is %d. err: %w",
			buf.Len(), co.rlpLength, MaxRLPBytesPerChannel, ErrTooManyRLPBytes)
	}
	co.rlpLength += buf.Len()

	_, err = io.Copy(co.compress, &buf)
	return err
}

// ReadyBytes returns the number of bytes that the channel out can immediately output into a frame.
// Use `Flush` or `Close` to move data from the compression buffer into the ready buffer if more bytes
// are needed. Add blocks may add to the ready buffer, but it is not guaranteed due to the compression stage.
func (co *ChannelOut) ReadyBytes() int {
	return co.buf.Len()
}

// Flush flushes the internal compression stage to the ready buffer. It enables pulling a larger & more
// complete frame. It reduces the compression efficiency.
func (co *ChannelOut) Flush() error {
	return co.compress.Flush()
}

func (co *ChannelOut) Close() error {
	if co.closed {
		return errors.New("already closed")
	}
	co.closed = true
	return co.compress.Close()
}

// OutputFrame writes a frame to w with a given max size
// Use `ReadyBytes`, `Flush`, and `Close` to modify the ready buffer.
// Returns io.EOF when the channel is closed & there are no more frames
// Returns nil if there is still more buffered data.
// Returns and error if it ran into an error during processing.
func (co *ChannelOut) OutputFrame(w *bytes.Buffer, maxSize uint64) error {
	f := Frame{
		ID:          co.id,
		FrameNumber: uint16(co.frame),
	}

	// Copy data from the local buffer into the frame data buffer
	// Don't go past the maxSize with the fixed frame overhead.
	// Fixed overhead: 32 + 8 + 2 + 4 + 1  = 47 bytes.
	// Add one extra byte for the version byte (for the entire L1 tx though)
	maxDataSize := maxSize - 47 - 1
	if maxDataSize > uint64(co.buf.Len()) {
		maxDataSize = uint64(co.buf.Len())
		// If we are closed & will not spill past the current frame
		// mark it is the final frame of the channel.
		if co.closed {
			f.IsLast = true
		}
	}
	f.Data = make([]byte, maxDataSize)

	if _, err := io.ReadFull(&co.buf, f.Data); err != nil {
		return err
	}

	if err := f.MarshalBinary(w); err != nil {
		return err
	}

	co.frame += 1
	if f.IsLast {
		return io.EOF
	} else {
		return nil
	}
}

// blockToBatch transforms a block into a batch object that can easily be RLP encoded.
func blockToBatch(block *types.Block) (*BatchData, error) {
	var opaqueTxs []hexutil.Bytes
	for i, tx := range block.Transactions() {
		if tx.Type() == types.DepositTxType {
			continue
		}
		otx, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("could not encode tx %v in block %v: %w", i, tx.Hash(), err)
		}
		opaqueTxs = append(opaqueTxs, otx)
	}
	l1InfoTx := block.Transactions()[0]
	if l1InfoTx.Type() != types.DepositTxType {
		return nil, ErrNotDepositTx
	}
	l1Info, err := L1InfoDepositTxData(l1InfoTx.Data())
	if err != nil {
		return nil, fmt.Errorf("could not parse the L1 Info deposit: %w", err)
	}

	return &BatchData{
		BatchV1{
			ParentHash:   block.ParentHash(),
			EpochNum:     rollup.Epoch(l1Info.Number),
			EpochHash:    l1Info.BlockHash,
			Timestamp:    block.Time(),
			Transactions: opaqueTxs,
		},
	}, nil
}
