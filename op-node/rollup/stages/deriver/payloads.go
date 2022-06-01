package deriver

import (
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"sync"
)

type PayloadReader struct {
	mu    sync.Mutex
	Inner BatchReaderStage
}

func (cs *PayloadReader) Read(payload *l2.ExecutionPayload) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var batch derive.BatchData
	if err := cs.Inner.Read(&batch); err != nil {
		return err
	}
	// TODO: filter batch
	// TODO: derive payload attributes
	// TODO: derive full payload, or consolidate with existing payload

	return nil
}
