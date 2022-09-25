package derive

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var ErrNotDepositTx = errors.New("first transaction in block is not a deposit tx")

type ChannelOut struct {
	id ChannelID
	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64
	// How much we've pulled from the reader so far
	offset uint64
	// scratch for temporary buffering
	scratch bytes.Buffer

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
		id:     ChannelID{}, // TODO: use GUID here instead of fully random data
		frame:  0,
		offset: 0,
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
	co.offset = 0
	co.buf.Reset()
	co.scratch.Reset()
	co.compress.Reset(&co.buf)
	co.closed = false
	_, err := rand.Read(co.id[:])
	if err != nil {
		return err
	}
	return nil
}

func (co *ChannelOut) AddBlock(block *types.Block) error {
	if co.closed {
		return errors.New("already closed")
	}
	return blockToBatch(block, co.compress)
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

// blockToBatch writes the raw block bytes (after batch encoding) to the writer
func blockToBatch(block *types.Block, w io.Writer) error {
	var opaqueTxs []hexutil.Bytes
	for _, tx := range block.Transactions() {
		if tx.Type() == types.DepositTxType {
			continue
		}
		otx, err := tx.MarshalBinary()
		if err != nil {
			return err // TODO: wrap err
		}
		opaqueTxs = append(opaqueTxs, otx)
	}
	l1InfoTx := block.Transactions()[0]
	if l1InfoTx.Type() != types.DepositTxType {
		return ErrNotDepositTx
	}
	l1Info, err := L1InfoDepositTxData(l1InfoTx.Data())
	if err != nil {
		return err // TODO: wrap err
	}

	batch := &BatchData{BatchV1{
		ParentHash:   block.ParentHash(),
		EpochNum:     rollup.Epoch(l1Info.Number),
		EpochHash:    l1Info.BlockHash,
		Timestamp:    block.Time(),
		Transactions: opaqueTxs,
	},
	}
	return rlp.Encode(w, batch)
}
