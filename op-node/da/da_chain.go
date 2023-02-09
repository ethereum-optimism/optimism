package da

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1TransactionFetcher interface {
	TxsByNumber(ctx context.Context, number uint64) (types.Transactions, error)
}

// FrameRef contains data to lookup a frame by its reference
type FrameRef struct {
	Number uint64
	Index  uint64
}

// Frame contains the frame data and it's reference
// Frame needs to contain FrameRef because each implementation decides if
// it's writing the entire frame to calldata or just the reference
type Frame struct {
	Data []byte
	Ref  *FrameRef // nil if not yet confirmed on DAChain
}

// Encode serializes the frame for writing to the DASource
// Optimism encode -> encodes the whole frame
// Celestia encode -> encodes only the FrameRef, fails if FrameRef is nil
func (f FrameRef) Encode() []byte { return nil }

// Decode deserializes the frame data back from the DASource
func (f FrameRef) Decode(data []byte) {}

// FrameFetcher returns a Frame by it's reference
type FrameFetcher interface {
	FetchFrame(ctx context.Context, ref FrameRef) (Frame, error)
}

// FrameWriter writes a FrameRef to the DASource
type FrameWriter interface {
	WriteFrame(context.Context, []byte) (FrameRef, error)
}

// DAChain satisifes both read/write on the DASource
type DAChain interface {
	FrameFetcher
	FrameWriter
	L1TransactionFetcher
}

func DataFromDASource(ctx context.Context, block eth.BlockID, daSource DAChain, log log.Logger) []Frame {
	refs, _ := daSource.TxsByNumber(ctx, block.Number)
	var frames []Frame
	for index := range refs {
		frame, _ := daSource.FetchFrame(ctx, FrameRef{
			Number: block.Number,
			Index:  uint64(index),
		})
		frames = append(frames, frame)
	}
	return frames
}
