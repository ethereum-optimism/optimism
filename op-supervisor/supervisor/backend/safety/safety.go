package safety

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SafetyIndex interface {
	// Updaters for the latest local safety status of each chain
	UpdateLocalUnsafe(chainID types.ChainID, ref eth.BlockRef) error
	UpdateLocalSafe(chainID types.ChainID, at eth.BlockRef, ref eth.BlockRef) error
	UpdateFinalizeL1(ref eth.BlockRef) error

	// Getters for the latest safety status of each chain
	UnsafeL2(chainID types.ChainID) (heads.HeadPointer, error)
	CrossUnsafeL2(chainID types.ChainID) (heads.HeadPointer, error)
	LocalSafeL2(chainID types.ChainID) (heads.HeadPointer, error)
	CrossSafeL2(chainID types.ChainID) (heads.HeadPointer, error)
	// We only finalize on full L2 block boundaries, hence not a heads.HeadPointer return.
	FinalizedL2(chainId types.ChainID) (eth.BlockID, error)
}

type ChainsDBClient interface {
	IteratorStartingAt(chainID types.ChainID, sealedNum uint64, logIndex uint32) (logs.Iterator, error)
	Check(chainID types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) (h common.Hash, err error)
}

type safetyIndex struct {
	log log.Logger

	chains ChainsDBClient

	unsafe    map[types.ChainID]*View
	safe      map[types.ChainID]*View
	finalized map[types.ChainID]eth.BlockID

	// remember what each non-finalized L2 block is derived from
	derivedFrom map[types.ChainID]map[common.Hash]eth.BlockRef

	// the last received L1 finality signal.
	finalizedL1 eth.BlockRef
}

func NewSafetyIndex(log log.Logger, chains ChainsDBClient) *safetyIndex {
	return &safetyIndex{
		log:         log,
		chains:      chains,
		unsafe:      make(map[types.ChainID]*View),
		safe:        make(map[types.ChainID]*View),
		finalized:   make(map[types.ChainID]eth.BlockID),
		derivedFrom: make(map[types.ChainID]map[common.Hash]eth.BlockRef),
	}
}

// UpdateLocalUnsafe updates the local-unsafe view for the given chain, and advances the cross-unsafe status.
func (r *safetyIndex) UpdateLocalUnsafe(chainID types.ChainID, ref eth.BlockRef) error {
	view, ok := r.safe[chainID]
	if !ok {
		iter, err := r.chains.IteratorStartingAt(chainID, ref.Number, 0)
		if err != nil {
			return fmt.Errorf("failed to open iterator for chain %s block %d", chainID, ref.Number)
		}
		view = &View{
			chainID: chainID,
			iter:    iter,
			localView: heads.HeadPointer{
				LastSealedBlockHash: ref.Hash,
				LastSealedBlockNum:  ref.Number,
				LastSealedTimestamp: ref.Time,
				LogsSince:           0,
			},
			localDerivedFrom: eth.BlockRef{},
			validWithinView:  r.ValidWithinUnsafeView,
		}
		r.unsafe[chainID] = view
	} else if err := view.UpdateLocal(eth.BlockRef{}, ref); err != nil {
		return fmt.Errorf("failed to update local-unsafe: %w", err)
	}
	local, _ := r.unsafe[chainID].Local()
	r.log.Debug("Updated local unsafe head", "chainID", chainID, "local", local)
	r.advanceCrossUnsafe()
	return nil
}

// advanceCrossUnsafe calls Process on all cross-unsafe views.
func (r *safetyIndex) advanceCrossUnsafe() {
	for chainID, view := range r.unsafe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-unsafe view", "chain", chainID, "err", err)
		}
		cross, _ := r.unsafe[chainID].Cross()
		r.log.Debug("Updated cross unsafe head", "chainID", chainID, "cross", cross)
	}
}

// UpdateLocalSafe updates the local-safe view for the given chain, and advances the cross-safe status.
func (r *safetyIndex) UpdateLocalSafe(
	chainID types.ChainID, at eth.BlockRef, ref eth.BlockRef) error {
	view, ok := r.safe[chainID]
	if !ok {
		iter, err := r.chains.IteratorStartingAt(chainID, ref.Number, 0)
		if err != nil {
			return fmt.Errorf("failed to open iterator for chain %s block %d", chainID, ref.Number)
		}
		view = &View{
			chainID: chainID,
			iter:    iter,
			localView: heads.HeadPointer{
				LastSealedBlockHash: ref.Hash,
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
		m = make(map[common.Hash]eth.BlockRef)
		r.derivedFrom[chainID] = m
	}
	m[ref.Hash] = at
	local, _ := r.safe[chainID].Local()
	r.log.Debug("Updated local safe head", "chainID", chainID, "local", local)
	r.advanceCrossSafe()
	return nil
}

// advanceCrossSafe calls Process on all cross-safe views, and advances the finalized safety status.
func (r *safetyIndex) advanceCrossSafe() {
	for chainID, view := range r.safe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-safe view", "chain", chainID, "err", err)
		}
		cross, _ := r.safe[chainID].Cross()
		r.log.Debug("Updated local safe head", "chainID", chainID, "cross", cross)
	}
	r.advanceFinalized()
}

// UpdateFinalizeL1 updates the finalized L1 block, and advances the finalized safety status.
func (r *safetyIndex) UpdateFinalizeL1(ref eth.BlockRef) error {
	if ref.Number <= r.finalizedL1.Number {
		return fmt.Errorf("ignoring old L1 finality signal of %s, already have %s", ref, r.finalizedL1)
	}
	r.finalizedL1 = ref
	r.log.Debug("Updated L1 finalized head", "L1finalized", ref)
	r.advanceFinalized()
	return nil
}

// advanceFinalized should be called whenever the finalized L1 block, or the cross-safe history, changes.
// This then promotes the irreversible cross-safe L2 blocks to a finalized safety status.
func (r *safetyIndex) advanceFinalized() {
	// Whatever was considered cross-safe at the finalized block-height can
	// now be considered finalized, since the inputs have become irreversible.
	for chainID, view := range r.safe {
		crossSafe, err := view.Cross()
		if err != nil {
			r.log.Info("Failed to get cross-safe data, cannot finalize", "chain", chainID, "err", err)
			continue
		}
		// TODO(#12184): we need to consider older cross-safe data,
		//  if we want to finalize something at all on longer lagging finality signal.
		// Could consider just iterating over all derivedFrom contents?
		l1Dep := r.derivedFrom[chainID][crossSafe.LastSealedBlockHash]
		if l1Dep.Number < r.finalizedL1.Number {
			r.finalized[chainID] = eth.BlockID{Hash: crossSafe.LastSealedBlockHash, Number: crossSafe.LastSealedBlockNum}
			finalized := r.finalized[chainID]
			r.log.Debug("Updated finalized head", "chainID", chainID, "finalized", finalized)
		}
	}
}

// UnsafeL2 returns the latest unsafe L2 block of the given chain.
func (r *safetyIndex) UnsafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no unsafe data for chain %s", chainID)
	}
	return view.Local()
}

// CrossUnsafeL2 returns the latest cross-unsafe L2 block of the given chain.
func (r *safetyIndex) CrossUnsafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no cross-unsafe data for chain %s", chainID)
	}
	return view.Cross()
}

// LocalSafeL2 returns the latest local-safe L2 block of the given chain.
func (r *safetyIndex) LocalSafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no local-safe data for chain %s", chainID)
	}
	return view.Local()
}

// CrossSafeL2 returns the latest cross-safe L2 block of the given chain.
func (r *safetyIndex) CrossSafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no cross-safe data for chain %s", chainID)
	}
	return view.Cross()
}

// FinalizedL2 returns the latest finalized L2 block of the given chain.
func (r *safetyIndex) FinalizedL2(chainId types.ChainID) (eth.BlockID, error) {
	finalized, ok := r.finalized[chainId]
	if !ok {
		return eth.BlockID{}, fmt.Errorf("not seen finalized data of chain %s at finalized L1 block %s", chainId, r.finalizedL1)
	}
	return finalized, nil
}

// ValidWithinUnsafeView checks if the given executing message is in the database.
// unsafe view is meant to represent all of the database, and so no boundary checks are needed.
func (r *safetyIndex) ValidWithinUnsafeView(_ uint64, execMsg *types.ExecutingMessage) error {
	execChainID := types.ChainIDFromUInt64(uint64(execMsg.Chain))
	_, err := r.chains.Check(execChainID, execMsg.BlockNum, execMsg.LogIdx, execMsg.Hash)
	return err
}

// ValidWithinSafeView checks if the given executing message is within the database,
// and within the L1 view of the caller.
func (r *safetyIndex) ValidWithinSafeView(l1View uint64, execMsg *types.ExecutingMessage) error {
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
		return logs.ErrFuture // TODO(#12185) need to distinguish between same-data future, and new-data future
	}
	// check if the L1 block is within the view
	if execL1Block.Number > l1View {
		return fmt.Errorf("exec message depends on L2 block %s:%d, derived from L1 block %s, not within view yet: %w",
			l2BlockHash, execMsg.BlockNum, execL1Block, logs.ErrFuture)
	}
	return nil
}

var _ SafetyIndex = (*safetyIndex)(nil)
