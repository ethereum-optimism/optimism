package safety

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type View struct {
	chainID types.ChainID

	iter logs.Iterator

	localView        heads.HeadPointer
	localDerivedFrom eth.L1BlockRef

	validWithinView func(l1View uint64, execMsg *suptypes.ExecutingMessage) error
}

func (vi *View) Cross() (heads.HeadPointer, error) {
	parentHash, parentNum, timestamp, logsSince, ok := vi.iter.Pointer()
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no db content yet: %w", logs.ErrFuture)
	}
	return heads.HeadPointer{
		LastSealedBlockHash: parentHash,
		LastSealedBlockNum:  parentNum,
		LastSealedTimestamp: timestamp,
		LogsSince:           logsSince,
	}, nil
}

func (vi *View) Local() (heads.HeadPointer, error) {
	if vi.localView == (heads.HeadPointer{}) {
		return heads.HeadPointer{}, logs.ErrFuture
	}
	return vi.localView, nil
}

func (vi *View) UpdateLocal(at eth.L1BlockRef, ref eth.L2BlockRef) error {
	vi.localView = heads.HeadPointer{
		LastSealedBlockHash: suptypes.TruncateHash(ref.Hash),
		LastSealedBlockNum:  ref.Number,
		//LastSealedTimestamp: ref.Time,
		LogsSince: 0,
	}
	vi.localDerivedFrom = at

	// TODO(devnet 2) reorg check against existing DB
	// TODO localView may be larger than what DB contents we have
	return nil
}

func (vi *View) Process() error {
	err := vi.iter.TraverseConditional(func(state logs.IteratorState) error {
		hash, num, ok := state.SealedBlock()
		if !ok {
			return logs.ErrFuture // maybe a more specific error for no-genesis case?
		}
		// TODO: reorg check in the future. To make sure that what we traverse is still canonical.
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
		if execMsg := state.ExecMessage(); execMsg == nil {
			// Check if executing message is within cross L2 view,
			// relative to the L1 view of current message.
			// And check if the message is valid to execute at all
			// (i.e. if it exists on the initiating side).
			// TODO(devnet2): it's inaccurate to check with the view of the local-unsafe
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

// ValidWithinUnsafeView checks if the given executing message is in the database.
// unsafe view is meant to represent all of the database, and so no boundary checks are needed.
func (r *RecentSafetyIndex) ValidWithinUnsafeView(_ uint64, execMsg *suptypes.ExecutingMessage) error {
	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))
	_, err := r.chains.Check(execChainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	return err
}

// ValidWithinSafeView checks if the given executing message is within the database,
// and within the L1 view of the caller.
func (r *RecentSafetyIndex) ValidWithinSafeView(l1View uint64, execMsg *suptypes.ExecutingMessage) error {
	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))

	// Check that the initiating message, which was pulled in by the executing message,
	// does indeed exist. And in which L2 block it exists (if any).
	l2BlockHash, err := r.chains.Check(execChainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	if err != nil {
		return err
	}
	// if the executing message falls within the execFinalized range, then nothing to check
	execFinalized, ok := r.finalized[execChainID]
	if ok && execFinalized.Number > execMsg.BlockNum {
		return nil
	}
	// check if the L1 block of the executing message is known
	execL1Block, ok := r.derivedFrom[execChainID][l2BlockHash]
	if !ok {
		return logs.ErrFuture // TODO need to distinguish between same-data future, and new-data future
	}
	// check if the L1 block is within the view
	if execL1Block.Number > l1View {
		return fmt.Errorf("exec message depends on L2 block %s:%d, derived from L1 block %s, not within view yet: %w",
			l2BlockHash, execMsg.BlockNum, execL1Block, logs.ErrFuture)
	}
	return nil
}
