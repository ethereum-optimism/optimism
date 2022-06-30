package derive

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// ChannelOut encodes and compresses batches together into a stream, and emits frames
type ChannelOut struct {
	id ChannelID

	// scratch for temporary buffering
	scratch bytes.Buffer

	// Compressor stage. Write input data to it
	compress *zlib.Writer

	// post compression buffer, read to build frames
	buf bytes.Buffer

	// The writer where to output RLP to. This is a multi-writer that buffers the RLP for replay.
	w io.Writer

	// All outputs of the rlp encoding so far, the inputs of the compression
	rlpCopy bytes.Buffer
	// input-offset of each Flush call. To reproduce the same state on rewind.
	flushes []uint64

	// The output offsets of each past frame number. The offset includes the content of the frame itself.
	frames []uint64

	closed bool
}

func (co *ChannelOut) ID() ChannelID {
	return co.id
}

func NewChannelOut(channelTime uint64) (*ChannelOut, error) {
	c := &ChannelOut{
		id: ChannelID{
			Time: channelTime,
		},
	}
	_, err := rand.Read(c.id.Data[:])
	if err != nil {
		return nil, err
	}

	compress, err := zlib.NewWriterLevel(&c.buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	c.compress = compress

	// All contents that are compressed will also be buffered in non-compressed form.
	// To be replayed when rewinding the channel.
	c.w = io.MultiWriter(c.compress, &c.rlpCopy)

	return c, nil
}

// Reset prepares the ChannelOut to be reused for new inputs.
// This does not preserve the old contents added to the channel.
// See Rewind to roll back to a state that can emit an older frame without losing input data.
//
// TODO: reuse ChannelOut for performance
func (co *ChannelOut) Reset(channelTime uint64) error {
	co.frames = nil
	co.buf.Reset()
	co.rlpCopy.Reset()
	co.w = io.MultiWriter(co.compress, &co.rlpCopy)
	co.scratch.Reset()
	co.compress.Reset(&co.buf)
	co.closed = false
	co.id.Time = channelTime
	_, err := rand.Read(co.id.Data[:])
	if err != nil {
		return err
	}
	return nil
}

// Rewind back to a given frame (incl.). This preserves the buffered batch inputs, but prepares for outputting a frame.
// If the frame number is negative, the channel will completely rewind back.
// If this function errors the ChannelOut becomes unusable, but a Rewind to an earlier frame may still recover it for reuse.
func (co *ChannelOut) Rewind(frame int) error {
	if frame >= len(co.frames) {
		return fmt.Errorf("unknown frame")
	}

	offset := uint64(0)
	if frame < 0 {
		frame = -1
	} else {
		offset = co.frames[frame]
	}

	// might as well reset scratch space
	co.scratch.Reset()

	// reset the output buffer
	co.buf.Reset()

	// reset the compression stream
	co.compress.Reset(&co.buf)

	// replay all RLP, but flush the compression where we need to
	prevFlushOffset := uint64(0)
	// don't touch the buffer reader, this reading is temporary, so we wrap the bytes with a new reader
	rlpCopy := bytes.NewReader(co.rlpCopy.Bytes())

	bufOffset := uint64(0)
	lastFlush := 0
	for i, flushOffset := range co.flushes {
		lastFlush = i
		delta := flushOffset - prevFlushOffset
		if _, err := io.CopyN(co.compress, rlpCopy, int64(delta)); err != nil {
			return fmt.Errorf("failed to replay rlp: %v", err)
		}
		if err := co.compress.Flush(); err != nil {
			return fmt.Errorf("failed to replay flush: %v", err)
		}
		prevFlushOffset = flushOffset

		prevBufOffset := bufOffset
		bufOffset = prevBufOffset + uint64(co.buf.Len())
		if bufOffset < offset {
			// we discard completely, and reuse buffer space, if we did not reach the end yet
			co.buf.Reset()
		} else if prevBufOffset < offset {
			// we discard partially if we needed to reach the offset still
			discard := offset - bufOffset
			_, _ = io.CopyN(io.Discard, &co.buf, int64(discard))
		} else {
			// we have reached the output offset,
			// we don't have to maintain any of the flushes we didn't capture in frames we preserve, yay efficiency.
			break
		}
	}
	// Now write all remaining rlp bytes to the compression stream without flushing
	if _, err := io.Copy(co.compress, rlpCopy); err != nil {
		return fmt.Errorf("failed to finish compression: %v", err)
	}
	co.flushes = co.flushes[:lastFlush]
	co.frames = co.frames[:frame+1]
	return nil
}

func (co *ChannelOut) AddBlock(block *types.Block) error {
	if co.closed {
		return errors.New("already closed")
	}
	return blockToBatch(block, co.w)
}

func makeUVarint(x uint64) []byte {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], x)
	return tmp[:n]
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
	// record how much RLP was written to the compression stream before flushing
	co.flushes = append(co.flushes, uint64(co.rlpCopy.Len()))
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
	w.Write(co.id.Data[:])
	w.Write(makeUVarint(co.id.Time))
	frameNr := uint64(len(co.frames))
	w.Write(makeUVarint(frameNr))

	// +1 for single byte of frame content, +1 for lastFrame bool
	if uint64(w.Len())+2 > maxSize {
		return fmt.Errorf("no more space: %d > %d", w.Len(), maxSize)
	}

	remaining := maxSize - uint64(w.Len())
	maxFrameLen := remaining - 1 // -1 for the bool at the end
	// estimate how many bytes we lose with encoding the length of the frame
	// by encoding the max length (larger uvarints take more space)
	maxFrameLen -= uint64(len(makeUVarint(maxFrameLen)))

	// Pull the data into a temporary buffer b/c we use uvarints to record the length
	// Could theoretically use the min of co.buf.Len() & maxFrameLen
	co.scratch.Reset()
	// TODO: we should not change the buffer until we know we don't exit with an early error.
	_, err := io.CopyN(&co.scratch, &co.buf, int64(maxFrameLen))
	if err != nil && err != io.EOF {
		return err
	}
	frameBodyLen := uint64(co.scratch.Len())
	w.Write(makeUVarint(frameBodyLen))
	if _, err := w.ReadFrom(&co.scratch); err != nil {
		return err
	}
	offset := frameBodyLen
	if len(co.frames) > 0 {
		offset += co.frames[len(co.frames)-1]
	}
	co.frames = append(co.frames, offset)

	// Only mark as closed if the channel is closed & there is no more data available
	if co.closed && err == io.EOF {
		w.WriteByte(1)
		return io.EOF
	} else {
		w.WriteByte(0)
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
	l1Info, err := L1InfoDepositTxData(l1InfoTx.Data())
	if err != nil {
		return err // TODO: wrap err
	}

	batch := &BatchData{BatchV1{
		EpochNum:     rollup.Epoch(l1Info.Number),
		EpochHash:    l1Info.BlockHash,
		Timestamp:    block.Time(),
		Transactions: opaqueTxs,
	},
	}
	return rlp.Encode(w, batch)
}
