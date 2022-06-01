package batcher

import (
	"bytes"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/stages"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type FrameReader struct {
	mu sync.Mutex

	Inner BinaryReaderStage

	offset uint64
}

func (cs *FrameReader) Read(dest *stages.Frame, maxSize uint64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	start := cs.Inner.Start()
	end := cs.Inner.End()
	// Check if we are done yet
	if start == end {
		return io.EOF
	}

	dest.Version = stages.FrameVersion0
	dest.Offset = cs.offset
	dest.End = end
	headerSize := dest.HeaderSize()
	if headerSize >= maxSize { // end early when we don't have enough space for more frames.
		return io.EOF
	}
	maxBodySize := maxSize - headerSize
	var buf bytes.Buffer
	innerN, err := io.CopyN(&buf, cs.Inner, int64(maxBodySize))
	if err == io.EOF {
		err = nil // it's fine to not read until the max body size if the stream is exhausted
		// mark the stream as done by
		if err := cs.Inner.Reset(end, end); err != nil {
			return fmt.Errorf("failed to reset stream to mark windo end at %s: %v", end, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to read new frame (offset: %d): %w", cs.offset, err)
	}
	dest.Content = buf.Bytes()
	// update after writing the previous offset
	cs.offset += uint64(innerN)
	return err
}

func (cs *FrameReader) Prepare(start, end eth.BlockID, offset uint64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// if the window didn't change, and if the offset is the same or later, then continue with the existing stream
	if cs.Inner.Start() == start && cs.Inner.End() == end && offset >= cs.offset {
		innerN, err := io.CopyN(io.Discard, cs.Inner, int64(offset-cs.offset))
		if err != nil {
			return fmt.Errorf("failed to reuse FrameReader while skipping to offset %d in window (start: %s end: %s): %w", offset, start, end, err)
		}
		if cs.offset+uint64(innerN) != offset {
			return fmt.Errorf("failed to reuse FrameReader while skipping to offset %d, ended at %d", offset, cs.offset+uint64(innerN))
		}
		cs.offset = offset
		return nil
	}

	// Reset inner stream
	if err := cs.Inner.Reset(start, end); err != nil {
		return fmt.Errorf("failed to prepare FrameReader while resetting inner stream to start %s end %s: %w", start, end, err)
	}
	// Now forward to the offset
	innerN, err := io.CopyN(io.Discard, cs.Inner, int64(offset))
	if err != nil {
		return fmt.Errorf("failed to prepare FrameReader while skipping to offset %d in window (start: %s end: %s): %w", offset, start, end, err)
	}
	if uint64(innerN) != offset {
		return fmt.Errorf("failed to prepare FrameReader while skipping to offset %d, ended at %d", offset, innerN)
	}
	cs.offset = offset
	return nil
}

func (cs *FrameReader) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Start()
}

func (cs *FrameReader) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.End()
}

func (cs *FrameReader) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.offset = 0
	return cs.Inner.Close()
}
