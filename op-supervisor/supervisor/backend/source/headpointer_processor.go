package source

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	supTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// headPointerProcessor is a HeadProcessor that updates the Head Storage
// which stores the latest database index for each chain
// a headPointerProcessor is created with a SafetyLevel to update the appropriate value
type headPointerProcessor struct {
	chain  supTypes.ChainID
	store  Storage
	safety supTypes.SafetyLevel
}

func NewHeadPointerProcessor(chain supTypes.ChainID, store Storage, safety supTypes.SafetyLevel) *headPointerProcessor {
	return &headPointerProcessor{
		chain:  chain,
		store:  store,
		safety: safety,
	}
}

// OnNewHead updates the head storage with the latest block index
func (p *headPointerProcessor) OnNewHead(_ context.Context, block eth.L1BlockRef) {
	lastIndexForBlock, err := p.store.LastLogInBlock(p.chain, block.Number)
	if err != nil {
		return
	}
	p.store.Apply(func(h *heads.Heads) error {
		chainHeads := h.Get(p.chain)
		switch p.safety {
		case supTypes.Unsafe:
			chainHeads.Unsafe = lastIndexForBlock
		case supTypes.Safe:
			chainHeads.LocalSafe = lastIndexForBlock
		case supTypes.Finalized:
			chainHeads.LocalFinalized = lastIndexForBlock
		}
		h.Put(p.chain, chainHeads)
		return nil
	})
}
