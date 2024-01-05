package derive

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type SpanChannelOut struct {
	id ChannelID
	// Frame ID of the next frame to emit. Increment after emitting
	frame uint64
	// rlpLength is the uncompressed size of the channel. Must be less than MAX_RLP_BYTES_PER_CHANNEL
	rlpLength int

	// Compressor stage. Write input data to it
	compress Compressor
	// closed indicates if the channel is closed
	closed bool
	// spanBatchBuilder contains information requires to build SpanBatch
	spanBatchBuilder *SpanBatchBuilder
	// reader contains compressed data for making output frames
	reader *bytes.Buffer
}

func (co *SpanChannelOut) ID() ChannelID {
	return co.id
}

func NewSpanChannelOut(compress Compressor, spanBatchBuilder *SpanBatchBuilder) (*SpanChannelOut, error) {
	c := &SpanChannelOut{
		id:               ChannelID{},
		frame:            0,
		rlpLength:        0,
		compress:         compress,
		spanBatchBuilder: spanBatchBuilder,
		reader:           &bytes.Buffer{},
	}
	_, err := rand.Read(c.id[:])
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (co *SpanChannelOut) Reset() error {
	co.frame = 0
	co.rlpLength = 0
	co.compress.Reset()
	co.reader.Reset()
	co.closed = false
	co.spanBatchBuilder.Reset()
	_, err := rand.Read(co.id[:])
	return err
}

// AddBlock adds a block to the channel. It returns the RLP encoded byte size
// and an error if there is a problem adding the block. The only sentinel error
// that it returns is ErrTooManyRLPBytes. If this error is returned, the channel
// should be closed and a new one should be made.
func (co *SpanChannelOut) AddBlock(block *types.Block) (uint64, error) {
	if co.closed {
		return 0, ErrChannelOutAlreadyClosed
	}

	batch, l1Info, err := BlockToSingularBatch(block)
	if err != nil {
		return 0, err
	}
	return co.AddSingularBatch(batch, l1Info.SequenceNumber)
}

// AddSingularBatch adds a batch to the channel. It returns the RLP encoded byte size
// and an error if there is a problem adding the batch. The only sentinel error
// that it returns is ErrTooManyRLPBytes. If this error is returned, the channel
// should be closed and a new one should be made.
//
// AddSingularBatch should be used together with BlockToSingularBatch if you need to access the
// BatchData before adding a block to the channel. It isn't possible to access
// the batch data with AddBlock.
//
// SingularBatch is appended to the channel's SpanBatch.
// A channel can have only one SpanBatch. And compressed results should not be accessible until the channel is closed, since the prefix and payload can be changed.
// So it resets channel contents and rewrites the entire SpanBatch each time, and compressed results are copied to reader after the channel is closed.
// It makes we can only get frames once the channel is full or closed, in the case of SpanBatch.
func (co *SpanChannelOut) AddSingularBatch(batch *SingularBatch, seqNum uint64) (uint64, error) {
	if co.closed {
		return 0, ErrChannelOutAlreadyClosed
	}
	if co.FullErr() != nil {
		// channel is already full
		return 0, co.FullErr()
	}
	var buf bytes.Buffer
	// Append Singular batch to its span batch builder
	co.spanBatchBuilder.AppendSingularBatch(batch, seqNum)
	// Convert Span batch to RawSpanBatch
	rawSpanBatch, err := co.spanBatchBuilder.GetRawSpanBatch()
	if err != nil {
		return 0, fmt.Errorf("failed to convert SpanBatch into RawSpanBatch: %w", err)
	}
	// Encode RawSpanBatch into bytes
	if err = rlp.Encode(&buf, NewBatchData(rawSpanBatch)); err != nil {
		return 0, fmt.Errorf("failed to encode RawSpanBatch into bytes: %w", err)
	}
	// Ensure that the total size of all RLP elements is less than or equal to MAX_RLP_BYTES_PER_CHANNEL
	if buf.Len() > MaxRLPBytesPerChannel {
		return 0, fmt.Errorf("could not take %d bytes as replacement of channel of %d bytes, max is %d. err: %w",
			buf.Len(), co.rlpLength, MaxRLPBytesPerChannel, ErrTooManyRLPBytes)
	}
	co.rlpLength = buf.Len()

	if co.spanBatchBuilder.GetBlockCount() > 1 {
		// Flush compressed data into reader to preserve current result.
		// If the channel is full after this block is appended, we should use preserved data.
		if err := co.compress.Flush(); err != nil {
			return 0, fmt.Errorf("failed to flush compressor: %w", err)
		}
		_, err = io.Copy(co.reader, co.compress)
		if err != nil {
			// Must reset reader to avoid partial output
			co.reader.Reset()
			return 0, fmt.Errorf("failed to copy compressed data to reader: %w", err)
		}
	}

	// Reset compressor to rewrite the entire span batch
	co.compress.Reset()
	// Avoid using io.Copy here, because we need all or nothing
	written, err := co.compress.Write(buf.Bytes())
	if co.compress.FullErr() != nil {
		err = co.compress.FullErr()
		if co.spanBatchBuilder.GetBlockCount() == 1 {
			// Do not return CompressorFullErr for the first block in the batch
			// In this case, reader must be empty. then the contents of compressor will be copied to reader when the channel is closed.
			err = nil
		}
		// If there are more than one blocks in the channel, reader should have data that preserves previous compression result before adding this block.
		// So, as a result, this block is not added to the channel and the channel will be closed.
		return uint64(written), err
	}

	// If compressor is not full yet, reader must be reset to avoid submitting invalid frames
	co.reader.Reset()
	return uint64(written), err
}

// InputBytes returns the total amount of RLP-encoded input bytes.
func (co *SpanChannelOut) InputBytes() int {
	return co.rlpLength
}

// ReadyBytes returns the number of bytes that the channel out can immediately output into a frame.
// Use `Flush` or `Close` to move data from the compression buffer into the ready buffer if more bytes
// are needed. Add blocks may add to the ready buffer, but it is not guaranteed due to the compression stage.
func (co *SpanChannelOut) ReadyBytes() int {
	return co.reader.Len()
}

// Flush flushes the internal compression stage to the ready buffer. It enables pulling a larger & more
// complete frame. It reduces the compression efficiency.
func (co *SpanChannelOut) Flush() error {
	if err := co.compress.Flush(); err != nil {
		return err
	}
	if co.closed && co.ReadyBytes() == 0 && co.compress.Len() > 0 {
		_, err := io.Copy(co.reader, co.compress)
		if err != nil {
			// Must reset reader to avoid partial output
			co.reader.Reset()
			return fmt.Errorf("failed to flush compressed data to reader: %w", err)
		}
	}
	return nil
}

func (co *SpanChannelOut) FullErr() error {
	return co.compress.FullErr()
}

func (co *SpanChannelOut) Close() error {
	if co.closed {
		return ErrChannelOutAlreadyClosed
	}
	co.closed = true
	if err := co.Flush(); err != nil {
		return err
	}
	return co.compress.Close()
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

	if _, err := io.ReadFull(co.reader, f.Data); err != nil {
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
