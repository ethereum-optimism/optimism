package safety

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SafetyIndex interface {
	// Updaters for the latest local safety status of each chain
	UpdateLocalUnsafe(chainID types.ChainID, ref eth.L2BlockRef) error
	UpdateLocalSafe(chainID types.ChainID, at eth.L1BlockRef, ref eth.L2BlockRef) error
	UpdateFinalizeL1(ref eth.L1BlockRef) error

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
	Check(chainID types.ChainID, blockNum uint64, logIdx uint32, logHash suptypes.TruncatedHash) (h suptypes.TruncatedHash, err error)
}

type RecentSafetyIndex struct {
	log log.Logger

	chains ChainsDBClient

	unsafe    map[types.ChainID]*View
	safe      map[types.ChainID]*View
	finalized map[types.ChainID]eth.BlockID

	// remember what each non-finalized L2 block is derived from
	derivedFrom map[types.ChainID]map[suptypes.TruncatedHash]eth.L1BlockRef

	// the last received L1 finality signal.
	finalizedL1 eth.L1BlockRef
}

func NewRecentSafetyIndex(log log.Logger, chains ChainsDBClient) *RecentSafetyIndex {
	return &RecentSafetyIndex{
		log:         log,
		chains:      chains,
		unsafe:      make(map[types.ChainID]*View),
		safe:        make(map[types.ChainID]*View),
		finalized:   make(map[types.ChainID]eth.BlockID),
		derivedFrom: make(map[types.ChainID]map[suptypes.TruncatedHash]eth.L1BlockRef),
	}
}

// UpdateLocalUnsafe updates the local-unsafe view for the given chain, and advances the cross-unsafe status.
func (r *RecentSafetyIndex) UpdateLocalUnsafe(chainID types.ChainID, ref eth.L2BlockRef) error {
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
	r.advanceCrossUnsafe()
	local, _ := r.unsafe[chainID].Local()
	cross, _ := r.unsafe[chainID].Cross()
	r.log.Debug("Updated unsafe head", "chainID", chainID, "local", local, "cross", cross)
	return nil
}

// advanceCrossUnsafe calls Process on all cross-unsafe views.
func (r *RecentSafetyIndex) advanceCrossUnsafe() {
	for chID, view := range r.unsafe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-unsafe view", "chain", chID, "err", err)
		}
	}
}

// UpdateLocalSafe updates the local-safe view for the given chain, and advances the cross-safe status.
func (r *RecentSafetyIndex) UpdateLocalSafe(
	chainID types.ChainID, at eth.L1BlockRef, ref eth.L2BlockRef) error {
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
	r.advanceCrossSafe()
	return nil
}

// advanceCrossSafe calls Process on all cross-safe views, and advances the finalized safety status.
func (r *RecentSafetyIndex) advanceCrossSafe() {
	for chID, view := range r.safe {
		if err := view.Process(); err != nil {
			r.log.Error("Failed to update cross-safe view", "chain", chID, "err", err)
		}
	}
	r.advanceFinalized()
	// TODO prune any L2 derivedFrom entry older than the L2 finalized entry
}

// UpdateFinalizeL1 updates the finalized L1 block, and advances the finalized safety status.
func (r *RecentSafetyIndex) UpdateFinalizeL1(ref eth.L1BlockRef) error {
	if ref.Number <= r.finalizedL1.Number {
		return fmt.Errorf("ignoring old L1 finality signal of %s, already have %s", ref, r.finalizedL1)
	}
	r.finalizedL1 = ref
	r.advanceFinalized()
	return nil
}

// advanceFinalized should be called whenever the finalized L1 block, or the cross-safe history, changes.
// This then promotes the irreversible cross-safe L2 blocks to a finalized safety status.
func (r *RecentSafetyIndex) advanceFinalized() {
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
			// TODO Temporary: truncated hashes have been replaced with full hashes
			fullHash := common.Hash{}
			fullHash.SetBytes(crossSafe.LastSealedBlockHash[:])
			r.finalized[chID] = eth.BlockID{Hash: fullHash, Number: crossSafe.LastSealedBlockNum}
		}
	}
}

// UnsafeL2 returns the latest unsafe L2 block of the given chain.
func (r *RecentSafetyIndex) UnsafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no unsafe data for chain %s", chainID)
	}
	return view.Local()
}

// CrossUnsafeL2 returns the latest cross-unsafe L2 block of the given chain.
func (r *RecentSafetyIndex) CrossUnsafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.unsafe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no cross-unsafe data for chain %s", chainID)
	}
	return view.Cross()
}

// LocalSafeL2 returns the latest local-safe L2 block of the given chain.
func (r *RecentSafetyIndex) LocalSafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no local-safe data for chain %s", chainID)
	}
	return view.Local()
}

// CrossSafeL2 returns the latest cross-safe L2 block of the given chain.
func (r *RecentSafetyIndex) CrossSafeL2(chainID types.ChainID) (heads.HeadPointer, error) {
	view, ok := r.safe[chainID]
	if !ok {
		return heads.HeadPointer{}, fmt.Errorf("no cross-safe data for chain %s", chainID)
	}
	return view.Cross()
}

// FinalizedL2 returns the latest finalized L2 block of the given chain.
func (r *RecentSafetyIndex) FinalizedL2(chainId types.ChainID) (eth.BlockID, error) {
	finalized, ok := r.finalized[chainId]
	if !ok {
		return eth.BlockID{}, fmt.Errorf("not seen finalized data of chain %s at finalized L1 block %s", chainId, r.finalizedL1)
	}
	return finalized, nil
}

var _ SafetyIndex = (*RecentSafetyIndex)(nil)
