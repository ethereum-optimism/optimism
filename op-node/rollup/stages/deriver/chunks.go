package deriver

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type FrameReader struct {
	mu sync.Mutex

	Source        func(ctx context.Context, chunkNum uint64) (*stages.Chunk, error)
	cancelRequest func()

	// Track the chunk number
	ChunkNum uint64

	frames []stages.Frame

	start eth.BlockID
	end   eth.BlockID
}

func (cs *FrameReader) Read(frame *stages.Frame) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.frames) > 0 {
		*frame = cs.frames[0]
		cs.frames = cs.frames[1:]
		if cs.end != frame.End {
			// old end becomes new start when we move the window
			cs.start = cs.end
			cs.end = frame.End
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cs.cancelRequest = cancel
	chunk, err := cs.Source(ctx, cs.ChunkNum)
	if err != nil {
		return fmt.Errorf("failed to retrieve chunk: %v", err)
	}

	cs.frames = chunk.Frames
	cs.ChunkNum = chunk.ChunkNum + 1

	prevStart := cs.start
	cs.start = chunk.Start

	// if we changed the start, then we need to return a reset
	if prevStart != cs.start {
		return io.EOF
	}

	if len(cs.frames) > 0 {
		*frame = cs.frames[0]
		cs.frames = cs.frames[1:]
		if cs.end != frame.End {
			// old end becomes new start when we move the window
			cs.start = cs.end
			cs.end = frame.End
		}
		return nil
	} else {
		return fmt.Errorf("empty chunk %d", chunk.ChunkNum)
	}
}

func (cs *FrameReader) Close() error {
	// no locking, Close can be called in parallel to ongoing reads
	if cs.cancelRequest != nil {
		cs.cancelRequest()
	}
	return nil
}
