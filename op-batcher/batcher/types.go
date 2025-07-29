package batcher

import (
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockWithDABytes struct {
	*types.Block
	estimatedDABytes *uint64
}

func (b *BlockWithDABytes) EstimatedDABytes() uint64 {
	if b.estimatedDABytes == nil {
		daSize, _ := metrics.EstimateBatchSize(b.Block)
		b.estimatedDABytes = &daSize
	}
	return *b.estimatedDABytes
}
