package batcher

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"io"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type BatchReader struct {
	mu sync.Mutex

	Inner PayloadReaderStage

	Conf *rollup.Config
}

func (cs *BatchReader) Reset(start, end eth.BlockID) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Reset(start, end)
}

func (cs *BatchReader) Read(dest *derive.BatchData) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var payload l2.ExecutionPayload
	err := cs.Inner.Read(&payload)
	if err != nil {
		if err == io.EOF {
			return io.EOF
		} else {
			return fmt.Errorf("failed to read payload: %w", err)
		}
	}
	l2BlockRef, err := l2.PayloadToBlockRef(&payload, &cs.Conf.Genesis)
	if err != nil {
		return fmt.Errorf("failed to derive block ref from payload: %v", err)
	}
	*dest = derive.BatchData{BatchV1: derive.BatchV1{
		Epoch:        rollup.Epoch(l2BlockRef.L1Origin.Number),
		Timestamp:    l2BlockRef.Time,
		Transactions: payload.Transactions,
	}}
	return nil
}

func (cs *BatchReader) Start() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Start()
}

func (cs *BatchReader) End() eth.BlockID {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.End()
}

func (cs *BatchReader) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.Inner.Close()
}
