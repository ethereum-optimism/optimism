package batcher

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type ChunkReader struct {
	mu sync.Mutex

	Inner FrameReaderStage

	SelectNextBlock func(start eth.BlockID, immediateSpace uint64) eth.BlockID

	// Track the chunk number
	ChunkNum uint64
}

func (cs *ChunkReader) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Start()
}

func (cs *ChunkReader) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.End()
}

func (cs *ChunkReader) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Close()
}

func (cs *ChunkReader) Read(dest *stages.Chunk, maxSize uint64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	dest.ChunkHeader = stages.ChunkHeader{
		Version:  stages.ChunkVersion0,
		ChunkNum: cs.ChunkNum,
		Start:    cs.Inner.Start(),
	}
	dest.Frames = nil
	headerSize := dest.ChunkHeader.HeaderSize()
	if headerSize+4 > maxSize {
		return fmt.Errorf("not enough space for chunk with single frame: %d + 4 > %d", headerSize, maxSize)
	}
	remainingSize := maxSize - headerSize

	for {
		// If we run out of space, return.
		// We always need 4 bytes for the length-prefix of the frame.
		if remainingSize <= 4 {
			cs.ChunkNum += 1
			return nil
		}
		start := cs.Inner.Start()
		end := cs.Inner.End()
		var frame stages.Frame
		if err := cs.Inner.Read(&frame, remainingSize-4); err == io.EOF {
			// Check if we are done with this window yet
			if start == end {
				end = cs.SelectNextBlock(start, remainingSize-4)
				if start == end {
					// If we can't select a newer window, then we ran out of L2 chain to construct chunks for
					if len(dest.Frames) > 0 {
						cs.ChunkNum += 1
						return nil // Don't return EOF if we have previous frames to return
					} else {
						return io.EOF
					}
				}
				if err := cs.Inner.Prepare(start, end, 0); err != nil {
					return fmt.Errorf("failed to prepare frame reader for window %s, %s: %v", start, end, err)
				}
				continue
			} else {
				// If the frame reader returns EOF early, before the window is done,
				// then we are done with this chunk and will need another chunk later to continue the window
				cs.ChunkNum += 1
				return nil
			}
		} else if err != nil {
			return fmt.Errorf("failed to read frame %d (start: %s, end: %s, remaining size: %d) for chunk: %v",
				len(dest.Frames), start, end, remainingSize, err)
		}
		dest.Frames = append(dest.Frames, frame)
		remainingSize -= 4 + frame.HeaderSize() + uint64(len(frame.Content))
	}
}

func (cs *ChunkReader) Prepare(start, end eth.BlockID, chunkNum uint64, offset uint64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if err := cs.Inner.Prepare(start, end, offset); err != nil {
		return fmt.Errorf("failed to prepare ChunkReader for window start: %s end: %s chunk: %d while preparing inner frame stage: %w", start, end, chunkNum, err)
	}
	cs.ChunkNum = chunkNum
	return cs.Inner.Prepare(start, end, offset)
}
