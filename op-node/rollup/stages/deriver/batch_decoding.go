package deriver

import (
	"github.com/ethereum-optimism/optimism/l2geth/rlp"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"sync"
)

const MaxBatchSize = 10_000_000

type BatchDecoder struct {
	mu    sync.Mutex
	Inner BinaryReaderStage
}

func (cs *BatchDecoder) Read(dest *derive.BatchData) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return rlp.NewStream(cs.Inner, MaxBatchSize).Decode(dest)
}
