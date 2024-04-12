package derive

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type SpanChannelOut struct {
	id ChannelID
	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64
	// rlp is the encoded, uncompressed data of the channel. length must be less than MAX_RLP_BYTES_PER_CHANNEL
	// it is a double buffer to allow us to "undo" the last change to the RLP structure when the target size is exceeded
	rlp [2]*bytes.Buffer
	// rlpIndex is the index of the current rlp buffer
	rlpIndex int
	// lastCompressedRLPSize tracks the *uncompressed* size of the last RLP buffer that was compressed
	// it is used to measure the growth of the RLP buffer when adding a new batch to optimize compression
	lastCompressedRLPSize int
	// compressed contains compressed data for making output frames
	compressed *bytes.Buffer
	// compress is the zlib writer for the channel
	compressor *zlib.Writer
	// target is the target size of the compressed data
	target uint64
	// closed indicates if the channel is closed
	closed bool
	// full indicates if the channel is full
	full error
	// spanBatch is the batch being built, which immutably holds genesis timestamp and chain ID, but otherwise can be reset
	spanBatch *SpanBatch
}

func (co *SpanChannelOut) ID() ChannelID {
	return co.id
}

func (co *SpanChannelOut) setRandomID() error {
	_, err := rand.Read(co.id[:])
	return err
}

func NewSpanChannelOut(genesisTimestamp uint64, chainID *big.Int, targetOutputSize uint64) (*SpanChannelOut, error) {
	c := &SpanChannelOut{
		id:         ChannelID{},
		frame:      0,
		spanBatch:  NewSpanBatch(genesisTimestamp, chainID),
		rlp:        [2]*bytes.Buffer{{}, {}},
		compressed: &bytes.Buffer{},
		target:     targetOutputSize,
	}
	var err error
	if err = c.setRandomID(); err != nil {
		return nil, err
	}
	if c.compressor, err = zlib.NewWriterLevel(c.compressed, zlib.BestCompression); err != nil {
		return nil, err
	}
	return c, nil
}

func (co *SpanChannelOut) Reset() error {
	co.closed = false
	co.full = nil
	co.frame = 0
	co.rlp[0].Reset()
	co.rlp[1].Reset()
	co.lastCompressedRLPSize = 0
	co.compressed.Reset()
	co.compressor.Reset(co.compressed)
	co.spanBatch = NewSpanBatch(co.spanBatch.GenesisTimestamp, co.spanBatch.ChainID)
	// setting the new randomID is the only part of the reset that can fail
	return co.setRandomID()
}

// activeRLP returns the active RLP buffer using the current rlpIndex
func (co *SpanChannelOut) activeRLP() *bytes.Buffer {
	return co.rlp[co.rlpIndex]
}

// inactiveRLP returns the inactive RLP buffer using the current rlpIndex
func (co *SpanChannelOut) inactiveRLP() *bytes.Buffer {
	return co.rlp[(co.rlpIndex+1)%2]
}

// swapRLP switches the active and inactive RLP buffers by modifying the rlpIndex
func (co *SpanChannelOut) swapRLP() {
	co.rlpIndex = (co.rlpIndex + 1) % 2
}

// AddBlock adds a block to the channel.
// returns an error if there is a problem adding the block. The only sentinel error
// that it returns is ErrTooManyRLPBytes. If this error is returned, the channel
// should be closed and a new one should be made.
func (co *SpanChannelOut) AddBlock(rollupCfg *rollup.Config, block *types.Block) error {
	if co.closed {
		return ErrChannelOutAlreadyClosed
	}

	batch, l1Info, err := BlockToSingularBatch(rollupCfg, block)
	if err != nil {
		return err
	}
	return co.AddSingularBatch(batch, l1Info.SequenceNumber)
}

// AddSingularBatch adds a SingularBatch to the channel, compressing the data if necessary.
// if the new batch would make the channel exceed the target size, the last batch is reverted,
// and the compression happens on the previous RLP buffer instead
// if the input is too small to need compression, data is accumulated but not compressed
func (co *SpanChannelOut) AddSingularBatch(batch *SingularBatch, seqNum uint64) error {
	// sentinel error for closed or full channel
	if co.closed {
		return ErrChannelOutAlreadyClosed
	}
	if err := co.FullErr(); err != nil {
		return err
	}

	// update the SpanBatch with the SingularBatch
	if err := co.spanBatch.AppendSingularBatch(batch, seqNum); err != nil {
		return fmt.Errorf("failed to append SingularBatch to SpanBatch: %w", err)
	}
	// convert Span batch to RawSpanBatch
	rawSpanBatch, err := co.spanBatch.ToRawSpanBatch()
	if err != nil {
		return fmt.Errorf("failed to convert SpanBatch into RawSpanBatch: %w", err)
	}

	// switch to the other buffer and reset it for new use
	// (the RLP buffer which is being made inactive holds the RLP encoded span batch just before the new batch was added)
	co.swapRLP()
	co.activeRLP().Reset()
	if err = rlp.Encode(co.activeRLP(), NewBatchData(rawSpanBatch)); err != nil {
		return fmt.Errorf("failed to encode RawSpanBatch into bytes: %w", err)
	}

	// check the RLP length against the max
	if co.activeRLP().Len() > MaxRLPBytesPerChannel {
		return fmt.Errorf("could not take %d bytes as replacement of channel of %d bytes, max is %d. err: %w",
			co.activeRLP().Len(), co.inactiveRLP().Len(), MaxRLPBytesPerChannel, ErrTooManyRLPBytes)
	}

	// if the compressed data *plus* the new rlp data is under the target size, return early
	// this optimizes out cases where the compressor will obviously come in under the target size
	rlpGrowth := co.activeRLP().Len() - co.lastCompressedRLPSize
	if uint64(co.compressed.Len()+rlpGrowth) < co.target {
		return nil
	}

	// we must compress the data to check if we've met or exceeded the target size
	if err = co.compress(); err != nil {
		return err
	}
	co.lastCompressedRLPSize = co.activeRLP().Len()

	// if the channel is now full, either return the compressed data, or the compressed previous data
	if err := co.FullErr(); err != nil {
		// if there is only one batch in the channel, it *must* be returned
		if len(co.spanBatch.Batches) == 1 {
			return nil
		}

		// if there is more than one batch in the channel, we revert the last batch
		// by switching the RLP buffer and doing a fresh compression
		co.swapRLP()
		if err := co.compress(); err != nil {
			return err
		}
		// return the full error
		return err
	}

	return nil
}

// compress compresses the active RLP buffer and checks if the compressed data is over the target size.
// it resets all the compression buffers because Span Batches aren't meant to be compressed incrementally.
func (co *SpanChannelOut) compress() error {
	co.compressed.Reset()
	co.compressor.Reset(co.compressed)
	if _, err := co.compressor.Write(co.activeRLP().Bytes()); err != nil {
		return err
	}
	if err := co.compressor.Close(); err != nil {
		return err
	}
	co.checkFull()
	return nil
}

// InputBytes returns the total amount of RLP-encoded input bytes.
func (co *SpanChannelOut) InputBytes() int {
	return co.activeRLP().Len()
}

// ReadyBytes returns the total amount of compressed bytes that are ready to be output.
// Span Channel Out does not provide early output, so this will always be 0 until the channel is closed or full
func (co *SpanChannelOut) ReadyBytes() int {
	if co.closed || co.FullErr() != nil {
		return co.compressed.Len()
	}
	return 0
}

// Flush implements the Channel Out
// Span Channel Out manages the flushing of the compressor internally, so this is a no-op
func (co *SpanChannelOut) Flush() error {
	return nil
}

// checkFull sets the full error if the compressed data is over the target size.
// the error is only set once, and the channel is considered full from that point on
func (co *SpanChannelOut) checkFull() {
	// if the channel is already full, don't update further
	if co.full != nil {
		return
	}
	if uint64(co.compressed.Len()) >= co.target {
		co.full = ErrCompressorFull
	}
}

func (co *SpanChannelOut) FullErr() error {
	return co.full
}

func (co *SpanChannelOut) Close() error {
	if co.closed {
		return ErrChannelOutAlreadyClosed
	}
	co.closed = true
	// if the channel was already full,
	// the compressor is already flushed and closed
	if co.FullErr() != nil {
		return nil
	}
	// if this channel is not full, we need to compress the last batch
	// this also flushes/closes the compressor
	return co.compress()
}

// OutputFrame writes a frame to w with a given max size and returns the frame
// number.
// Use `ReadyBytes`, `Flush`, and `Close` to modify the ready buffer.
// Returns an error if the `maxSize` < FrameV0OverHeadSize.
// Returns io.EOF when the channel is closed & there are no more frames.
// Returns nil if there is still more buffered data.
// Returns an error if it ran into an error during processing.
func (co *SpanChannelOut) OutputFrame(w *bytes.Buffer, maxSize uint64) (uint16, error) {
	// Check that the maxSize is large enough for the frame overhead size.
	if maxSize < FrameV0OverHeadSize {
		return 0, ErrMaxFrameSizeTooSmall
	}

	f := createEmptyFrame(co.id, co.frame, co.ReadyBytes(), co.closed, maxSize)

	if _, err := io.ReadFull(co.compressed, f.Data); err != nil {
		return 0, err
	}

	if err := f.MarshalBinary(w); err != nil {
		return 0, err
	}

	co.frame += 1
	fn := f.FrameNumber
	if f.IsLast {
		return fn, io.EOF
	} else {
		return fn, nil
	}
}
