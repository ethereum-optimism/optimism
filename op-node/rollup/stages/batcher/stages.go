package batcher

import (
	"errors"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
)

// ReorgErr is returned when the canonical source data does not match the window of blocks
// that is being read or written.
var ReorgErr = errors.New("reorg detected")

type Windowed interface {
	// Start of the current window
	Start() eth.BlockID
	// End of the current window. If End() == Start() then the stream should return io.EOF upon reading.
	End() eth.BlockID
}

type WindowedReader interface {
	Windowed
	// Reset clears the buffered data in the stage,
	// and resets any underlying stream to the given window from start to end.
	//
	// The start point is exclusive: this is the parent block of the first block we will stream after.
	// The end point is inclusive: this is the last block we will include in the stream.
	//
	// It returns an error if the start or end cannot be found, or if either is not canonical
	Reset(start, end eth.BlockID) error
}

// PayloadReaderStage gets the L2 blocks from some source
type PayloadReaderStage interface {
	WindowedReader
	Read(dest *l2.ExecutionPayload) error
	io.Closer
}

// BatchReaderStage gets batches from execution payloads
type BatchReaderStage interface {
	WindowedReader
	Read(dest *derive.BatchData) error
	io.Closer
}

// BinaryReaderStage reads input data from an inner stage, and transforms it
type BinaryReaderStage interface {
	WindowedReader
	io.ReadCloser
}

// FrameReaderStage reads a frame, which will be at most maxSize bytes when marshalled.
type FrameReaderStage interface {
	Windowed
	// Prepare resets the underlying stream to the given window,
	// and forwards to the given offset, to not read that data from the stream again.
	Prepare(start, end eth.BlockID, offset uint64) error
	// Read constructs a new frame from the underlying stream,
	// which takes at most maxSize bytes when encoded.
	// The block window may be reset to (end, end) if the stream is fully consumed.
	// An io.EOF is returned when the window of data is exhausted,
	// or no additional frame could be read within maxSize.
	// The caller can identify an exhausted window by checking if start==end.
	Read(dest *stages.Frame, maxSize uint64) error
	io.Closer
}

type ChunkReaderStage interface {
	Windowed
	// Prepare resets the underlying stream to the given window and offset,
	// and sets the current chunk number.
	Prepare(start, end eth.BlockID, chunkNum uint64, offset uint64) error
	// Read constructs a new chunk from the underlying stram,
	// which takes at most maxSize bytes when encoded.
	// The current block window may change as new frames are read from the stream.
	// An io.EOF error is returned when the underlying stream is exhausted,
	// i.e. no more blocks can be read.
	Read(dest *stages.Chunk, maxSize uint64) error
	io.Closer
}
