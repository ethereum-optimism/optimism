package deriver

import (
	"io"

	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
)

type PayloadReaderStage interface {
	Read(payload *l2.ExecutionPayload) error
	io.Closer
}

type BatchReaderStage interface {
	Read(batch *derive.BatchData) error
	io.Closer
}

type BinaryReaderStage interface {
	io.ReadCloser
}

type FrameReaderStage interface {
	// Read reads the next frame of the window,
	// and returns an EOF if the window content is exhausted or not available anymore.
	Read(frame *stages.Frame) error
	io.Closer
}
