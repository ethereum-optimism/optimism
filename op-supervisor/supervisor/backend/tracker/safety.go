package tracker

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type HeadPointerV2 struct {
	// LastSealedBlockHash is the last fully-processed block
	LastSealedBlockHash suptypes.TruncatedHash
	LastSealedBlockNum  uint64
	LastSealedTimestamp uint64

	// Number of logs that have been verified since the LastSealedBlock.
	// These logs are contained in the block that builds on top of the LastSealedBlock.
	LogsSince uint32
}

// WithinRange checks if the given log, in the given block,
// is within range (i.e. before or equal to the head-pointer).
// This does not guarantee that the log exists.
func (ptr *HeadPointerV2) WithinRange(blockNum uint64, logIdx uint32) bool {
	if ptr.LastSealedBlockHash == (suptypes.TruncatedHash{}) {
		return false // no block yet
	}
	return blockNum <= ptr.LastSealedBlockNum ||
		(blockNum+1 == ptr.LastSealedBlockNum && logIdx < ptr.LogsSince)
}

func (ptr *HeadPointerV2) IsSealed(blockNum uint64) bool {
	if ptr.LastSealedBlockHash == (suptypes.TruncatedHash{}) {
		return false // no block yet
	}
	return blockNum <= ptr.LastSealedBlockNum
}

type SafetyIndex interface {
	// Methods called by the rollup-node upon new local info
	UpdateLocalUnsafe(chainID types.ChainID, ref eth.L2BlockRef) error
	UpdateLocalSafe(chainID types.ChainID, at eth.L1BlockRef, ref eth.L2BlockRef) eror
	UpdateFinalizeL1(ref eth.L1BlockRef) error

	// Metrhods called by the rollup-node to poll block safety state, promote the blocks
	UnsafeL2(chainID types.ChainID) (HeadPointerV2, error)
	CrossUnsafeL2(chainID types.ChainID) (HeadPointerV2, error)
	LocalSafeL2(chainID types.ChainID) (HeadPointerV2, error)
	CrossSafeL2(chainID types.ChainID) (HeadPointerV2, error)
	// FinalizedL2 returns the latest finalized L2 block of the given chain.
	// We only finalize on full L2 block boundaries, hence not a HeadPointerV2 return.
	FinalizedL2(chainId types.ChainID) (eth.BlockID, error)
}

type EventsIndex interface {
	IteratorStartingAt(sealedNum uint64, logIndex uint32) (logs.Iterator, error)

	// returns ErrConflict if the log does not match the canonical chain.
	// returns ErrFuture if the log is out of reach.
	// returns nil if the log is known and matches the canonical chain.
	Check(blockNum uint64, logIdx uint32, logHash suptypes.TruncatedHash) (h suptypes.TruncatedHash, err error)
}

type RecentSafetyIndex struct {
	log log.Logger

	events map[types.ChainID]EventsIndex

	unsafe map[types.ChainID]*View
	safe   map[types.ChainID]*View

	finalized map[types.ChainID]eth.BlockID

	// remember what each non-finalized L2 block is derived from
	derivedFrom map[types.ChainID]map[suptypes.TruncatedHash]eth.L1BlockRef

	// the last received L1 finality signal.
	finalizedL1 eth.L1BlockRef
}

// TODO constructor RecentSafetyIndex

func (r *RecentSafetyIndex) UpdateLocalUnsafe(chainID types.ChainID, ref eth.L2BlockRef) error {
	view, ok := r.unsafe[chainID]
	if !ok {
		iter, err := r.events[chainID].IteratorStartingAt(ref.Number, 0)
		if err != nil {
			return fmt.Errorf("failed to open iterator for chain %s block %d", chainID, ref.Number)
		}
		view = &View{
			chainID: chainID,
			iter:    iter,
			localView: HeadPointerV2{
				LastSealedBlockHash: suptypes.TruncateHash(ref.Hash),
				LastSealedBlockNum:  ref.Number,
				LastSealedTimestamp: ref.Time,
				LogsSince:           0,
			},
			localDerivedFrom: eth.L1BlockRef{},
			validWithinView:  r.ValidWithinUnsafeView,
		}
		r.unsafe[chainID] = view
	} else if err := view.UpdateLocal(eth.L1BlockRef{}, ref); err != nil {
		return fmt.Errorf("failed to update local-unsafe: %w", err)
	}
	r.pokeCrossUnsafe()
	return nil
}

func (r *RecentSafetyIndex) pokeCrossUnsafe() {
	// TODO: instead of synchronous processing, trigger a worker thread, per L2
	for chID, view := range r.unsafe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-unsafe view", "chain", chID, "err", err)
		}
	}
}

func (r *RecentSafetyIndex) UpdateLocalSafe(
	chainID types.ChainID, at eth.L1BlockRef, ref eth.L2BlockRef) error {
	view, ok := r.safe[chainID]
	if !ok {
		iter, err := r.events[chainID].IteratorStartingAt(ref.Number, 0)
		if err != nil {
			return fmt.Errorf("failed to open iterator for chain %s block %d", chainID, ref.Number)
		}
		view = &View{
			chainID: chainID,
			iter:    iter,
			localView: HeadPointerV2{
				LastSealedBlockHash: suptypes.TruncateHash(ref.Hash),
				LastSealedBlockNum:  ref.Number,
				LastSealedTimestamp: ref.Time,
				LogsSince:           0,
			},
			localDerivedFrom: at,
			validWithinView:  r.ValidWithinSafeView,
		}
		r.safe[chainID] = view
	} else if err := view.UpdateLocal(at, ref); err != nil {
		return fmt.Errorf("failed to update local-safe: %w", err)
	}

	// register what this L2 block is derived from
	m, ok := r.derivedFrom[chainID]
	if !ok {
		m = make(map[suptypes.TruncatedHash]eth.L1BlockRef)
		r.derivedFrom[chainID] = m
	}
	m[suptypes.TruncateHash(ref.Hash)] = at
	r.pokeCrossSafe()
	return nil
}

func (r *RecentSafetyIndex) pokeCrossSafe() {
	// TODO: instead of synchronous processing, trigger a worker thread, per L2
	for chID, view := range r.safe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-safe view", "chain", chID, "err", err)
		}
	}
	r.pokeFinalized()

	// TODO prune any L2 derivedFrom entry older than the L2 finalized entry
}

func (r *RecentSafetyIndex) ValidWithinUnsafeView(_ uint64, execMsg *suptypes.ExecutingMessage) error {
	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))
	// TODO combine with the above call
	_, err := r.events[execChainID].Check(execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	return err
}

func (r *RecentSafetyIndex) ValidWithinSafeView(l1View uint64, execMsg *suptypes.ExecutingMessage) error {
	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))

	// Check that the initiating message, which was pulled in by the executing message,
	// does indeed exist. And in which L2 block it exists (if any).
	l2BlockHash, err := r.events[execChainID].Check(execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	if err != nil {
		return err
	}

	finalized, ok := r.finalized[execChainID]
	if ok && finalized.Number > execMsg.BlockNum {
		// The executing message exists, and is derived from a L2 block that has been finalized already,
		// and so nothing has to be checked.
		return nil
	}

	// check if execMsg.blocknum is older than finalized.
	// If yes, then ignore the derived-from check
	l1Block, ok := r.derivedFrom[execChainID][l2BlockHash]
	if !ok {
		// unable to tell
		return logs.ErrFuture // TODO need to distinguish between same-data future, and new-data future
	}
	// check against l1 block view of the caller
	if l1Block.Number > l1View {
		return fmt.Errorf("exec message depends on L2 block %s:%d, derived from L1 block %s, not within view yet: %w",
			hash, execMsg.BlockNum, l1Block, logs.ErrFuture)
	}

	return err
}

type View struct {
	chainID types.ChainID

	iter logs.Iterator

	localView        HeadPointerV2
	localDerivedFrom eth.L1BlockRef

	validWithinView func(l1View uint64, execMsg *suptypes.ExecutingMessage) error
}

func (vi *View) Cross() (HeadPointerV2, error) {
	parentHash, parentNum, timestamp, logsSince, ok := vi.iter.Pointer()
	if !ok {
		return HeadPointerV2{}, fmt.Errorf("no db content yet: %w", logs.ErrFuture)
	}
	return HeadPointerV2{
		LastSealedBlockHash: parentHash,
		LastSealedBlockNum:  parentNum,
		LastSealedTimestamp: timestamp,
		LogsSince:           logsSince,
	}, nil
}

func (vi *View) Local() (HeadPointerV2, error) {
	if vi.localView == (HeadPointerV2{}) {
		return HeadPointerV2{}, logs.ErrFuture
	}
	return vi.localView, nil
}

func (vi *View) UpdateLocal(at eth.L1BlockRef, ref eth.L2BlockRef) error {
	vi.localView = HeadPointerV2{
		LastSealedBlockHash: suptypes.TruncateHash(ref.Hash),
		LastSealedBlockNum:  ref.Number,
		LastSealedTimestamp: ref.Time,
		LogsSince:           0,
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

func (r *RecentSafetyIndex) UpdateFinalizeL1(ref eth.L1BlockRef) error {
	if ref.Number <= r.finalizedL1.Number {
		return fmt.Errorf("ignoring old L1 finality signal of %s, already have %s", ref, r.finalizedL1)
	}
	r.finalizedL1 = ref
	r.pokeFinalized()
	return nil
}

func (r *RecentSafetyIndex) UnsafeL2(chainID types.ChainID) (HeadPointerV2, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return HeadPointerV2{}, fmt.Errorf("no unsafe data for chain %s", chainID)
	}
	return view.Local()
}

func (r *RecentSafetyIndex) CrossUnsafeL2(chainID types.ChainID) (HeadPointerV2, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return HeadPointerV2{}, fmt.Errorf("no cross-unsafe data for chain %s", chainID)
	}
	return view.Cross()
}

func (r *RecentSafetyIndex) LocalSafeL2(chainID types.ChainID) (HeadPointerV2, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return HeadPointerV2{}, fmt.Errorf("no local-safe data for chain %s", chainID)
	}
	return view.Local()
}

func (r *RecentSafetyIndex) CrossSafeL2(chainID types.ChainID) (HeadPointerV2, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return HeadPointerV2{}, fmt.Errorf("no cross-safe data for chain %s", chainID)
	}
	return view.Cross()
}

func (r *RecentSafetyIndex) FinalizedL2(chainId types.ChainID) (eth.BlockID, error) {
	finalized, ok := r.finalized[chainId]
	if !ok {
		return eth.BlockID{}, fmt.Errorf("not seen finalized data of chain %s at finalized L1 block %s", chainId, r.finalizedL1)
	}
	return finalized, nil
}

// pokeFinalized should be called whenever the finalized L1 block, or the cross-safe history, changes.
// This then promotes the irreversible cross-safe L2 blocks to a finalized safety status.
func (r *RecentSafetyIndex) pokeFinalized() {
	// Whatever was considered cross-safe at the finalized block-height can
	// now be considered finalized, since the inputs have become irreversible.
	for chID, view := range r.safe {
		crossSafe, err := view.Cross()
		if err != nil {
			r.log.Info("Failed to get cross-safe data, cannot finalize", "chain", chID, "err", err)
			continue
		}
		// TODO we need to consider older cross-safe data,
		//  if we want to finalize something at all on longer lagging finality signal.
		// Could consider just iterating over all derivedFrom contents?
		l1Dep := r.derivedFrom[chID][crossSafe.LastSealedBlockHash]
		if l1Dep.Number < r.finalizedL1.Number {
			r.finalized[chID] = crossSafe.ID()
		}
	}
}

var _ SafetyIndex = (*RecentSafetyIndex)(nil)
