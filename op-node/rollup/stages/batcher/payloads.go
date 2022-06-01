package batcher

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"io"
	"sync"
)

type PayloadReader struct {
	mu sync.Mutex

	start eth.BlockID
	end   eth.BlockID

	Source        func(ctx context.Context, num uint64) (*l2.ExecutionPayload, error)
	cancelRequest func()

	lastComplete eth.BlockID
}

func (cs *PayloadReader) Reset(start, end eth.BlockID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.start = start
	cs.end = end
	cs.lastComplete = start
	return nil
}

func (cs *PayloadReader) Read(dest *l2.ExecutionPayload) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	// Check if we reached the end already.
	if cs.lastComplete == cs.end {
		return io.EOF
	}
	ctx, cancel := context.WithCancel(context.Background())
	cs.cancelRequest = cancel
	payload, err := cs.Source(ctx, cs.lastComplete.Number+1)
	if err != nil {
		if err == ctx.Err() {
			// stage was closed gracefully, send an EOF
			return io.EOF
		}
		return fmt.Errorf("failed to retrieve payload at number %d", cs.lastComplete.Number+1)
	}
	if payload.ParentHash != cs.lastComplete.Hash {
		return ReorgErr
	}
	if uint64(payload.BlockNumber) == cs.end.Number && payload.BlockHash != cs.end.Hash {
		return ReorgErr
	}
	if uint64(payload.BlockNumber) > cs.end.Number {
		return fmt.Errorf("retrieved block out of window bounds: %s (end: %s)", payload.ID(), cs.end)
	}
	cs.lastComplete = payload.ID()
	*dest = *payload
	return nil
}

func (cs *PayloadReader) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.start
}

func (cs *PayloadReader) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.start
}

func (cs *PayloadReader) Close() error {
	// no locking, Close can be called in parallel to ongoing reads
	if cs.cancelRequest != nil {
		cs.cancelRequest()
	}
	return nil
}
