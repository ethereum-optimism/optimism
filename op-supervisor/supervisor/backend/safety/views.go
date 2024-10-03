package safety

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type View struct {
	chainID types.ChainID

	iter logs.Iterator

	localView        heads.HeadPointer
	localDerivedFrom eth.BlockRef

	validWithinView func(l1View uint64, execMsg *types.ExecutingMessage) error
}

func (vi *View) Cross() (heads.HeadPointer, error) {
	return vi.iter.HeadPointer()
}

func (vi *View) Local() (heads.HeadPointer, error) {
	if vi.localView == (heads.HeadPointer{}) {
		return heads.HeadPointer{}, logs.ErrFuture
	}
	return vi.localView, nil
}

func (vi *View) UpdateLocal(at eth.BlockRef, ref eth.BlockRef) error {
	vi.localView = heads.HeadPointer{
		LastSealedBlockHash: ref.Hash,
		LastSealedBlockNum:  ref.Number,
		//LastSealedTimestamp: ref.Time,
		LogsSince: 0,
	}
	vi.localDerivedFrom = at

	// TODO(#11693): reorg check against existing DB
	// TODO(#12186): localView may be larger than what DB contents we have
	return nil
}

func (vi *View) Process() error {
	err := vi.iter.TraverseConditional(func(state logs.IteratorState) error {
		hash, num, ok := state.SealedBlock()
		if !ok {
			return logs.ErrFuture // maybe a more specific error for no-genesis case?
		}
		// TODO(#11693): reorg check in the future. To make sure that what we traverse is still canonical.
		_ = hash
		// check if L2 block is within view
		if !vi.localView.WithinRange(num, 0) {
			return logs.ErrFuture
		}
		_, initLogIndex, ok := state.InitMessage()
		if !ok {
			return nil // no readable message, just an empty block
		}
		// check if the message is within view
		if !vi.localView.WithinRange(num, initLogIndex) {
			return logs.ErrFuture
		}
		// check if it is an executing message. If so, check the dependency
		if execMsg := state.ExecMessage(); execMsg != nil {
			// Check if executing message is within cross L2 view,
			// relative to the L1 view of current message.
			// And check if the message is valid to execute at all
			// (i.e. if it exists on the initiating side).
			// TODO(#12187): it's inaccurate to check with the view of the local-unsafe
			// it should be limited to the L1 view at the time of the inclusion of execution of the message.
			err := vi.validWithinView(vi.localDerivedFrom.Number, execMsg)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		panic("expected reader to complete with an exit-error")
	}
	if errors.Is(err, logs.ErrFuture) {
		// register the new cross-safe block as cross-safe up to the current L1 view
		return nil
	}
	return err
}
