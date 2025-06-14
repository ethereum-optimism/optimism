package backend

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/indexing"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/safemath"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// If the initiating message exists, the block it is included in is returned.
func (b *Backend) checkAccessWithDB(acc types.Access) (eth.BlockID, error) {
	// Check if message exists
	bl, err := b.chainDBs.Contains(acc.ChainID, types.ContainsQuery{
		Timestamp: acc.Timestamp,
		BlockNum:  acc.BlockNumber,
		LogIdx:    acc.LogIndex,
		Checksum:  acc.Checksum,
	})
	if err != nil {
		return eth.BlockID{}, err
	}

	return bl.ID(), nil
}

// checkSafety is a helper method to check if a block has the given safety level.
// It is already assumed to exist in the canonical unsafe chain.
func (b *Backend) checkSafety(chainID eth.ChainID, blockID eth.BlockID, safetyLevel types.SafetyLevel) error {
	switch safetyLevel {
	case types.LocalUnsafe:
		return nil // msg exists, nothing more to check
	case types.CrossUnsafe:
		return b.chainDBs.IsCrossUnsafe(chainID, blockID)
	case types.LocalSafe:
		return b.chainDBs.IsLocalSafe(chainID, blockID)
	case types.CrossSafe:
		return b.chainDBs.IsCrossSafe(chainID, blockID)
	case types.Finalized:
		return b.chainDBs.IsFinalized(chainID, blockID)
	default:
		return types.ErrConflict
	}
}

func (b *Backend) CheckAccessList(ctx context.Context, inboxEntries []common.Hash,
	minSafety types.SafetyLevel, execDescr types.ExecutingDescriptor) error {
	switch minSafety {
	case types.LocalUnsafe, types.CrossUnsafe, types.LocalSafe, types.CrossSafe, types.Finalized:
		// valid safety level
	default:
		return ErrUnexpectedMinSafetyLevel
	}

	b.logger.Debug("Checking access-list", "minSafety", minSafety, "length", len(inboxEntries))

	h := b.chainDBs.AcquireHandle()
	defer h.Release()

	entries := inboxEntries
	for len(entries) > 0 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("stopped access-list check early: %w", err)
		}
		remaining, acc, err := types.ParseAccess(entries)
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}
		entries = remaining

		// Register initiating side as a dependency
		h.DependOnDerivedTime(acc.Timestamp)

		// TODO(#16245): backwards compat: if user does not specify executing chain, then assume the initiating chain ID.
		// This supports op-reth, op-rbuilder, proxyd while they are not updated to provide this chain ID.
		execChainID := execDescr.ChainID
		if execDescr.ChainID == (eth.ChainID{}) {
			execChainID = acc.ChainID
		}
		// If not specified, assume the same chain as the initiating side.
		if !b.linker.CanExecute(execChainID, execDescr.Timestamp, acc.ChainID, acc.Timestamp) {
			b.logger.Debug("Access-list link check failed")
			return types.ErrConflict
		}
		if execDescr.Timeout != 0 {
			maxTimestamp := safemath.SaturatingAdd(execDescr.Timestamp, execDescr.Timeout)
			if !b.linker.CanExecute(execChainID, maxTimestamp, acc.ChainID, acc.Timestamp) {
				b.logger.Debug("Access-list link check at timeout time failed")
				return types.ErrConflict
			}
		}

		msgBlockFromDB, err := b.checkAccessWithDB(acc)
		if err != nil {
			b.logger.Debug("Access-list inclusion check failed", "err", err)
			return types.ErrConflict
		}

		if err := b.checkSafety(acc.ChainID, msgBlockFromDB, minSafety); err != nil {
			b.logger.Debug("Access-list safety check failed", "err", err)
			return types.ErrConflict
		}
	}
	return h.Err()
}

func (b *Backend) CrossSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	p, err := b.chainDBs.CrossSafe(chainID)
	if err != nil {
		return types.DerivedIDPair{}, err
	}
	return types.DerivedIDPair{
		Source:  p.Source.ID(),
		Derived: p.Derived.ID(),
	}, nil
}

func (b *Backend) LocalSafe(ctx context.Context, chainID eth.ChainID) (types.DerivedIDPair, error) {
	p, err := b.chainDBs.LocalSafe(chainID)
	if err != nil {
		return types.DerivedIDPair{}, err
	}
	return types.DerivedIDPair{
		Source:  p.Source.ID(),
		Derived: p.Derived.ID(),
	}, nil
}

func (b *Backend) LocalUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	v, err := b.chainDBs.LocalUnsafe(chainID)
	if err != nil {
		return eth.BlockID{}, err
	}
	return v.ID(), nil
}

func (b *Backend) CrossUnsafe(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	v, err := b.chainDBs.CrossUnsafe(chainID)
	if err != nil {
		return eth.BlockID{}, err
	}
	return v.ID(), nil
}

func (b *Backend) LocalSafeDerivedAt(ctx context.Context, chainID eth.ChainID, source eth.BlockID) (eth.BlockID, error) {
	v, err := b.chainDBs.LocalSafeDerivedAt(chainID, source)
	if err != nil {
		return eth.BlockID{}, err
	}
	return v.ID(), nil
}

func (b *Backend) FindSealedBlock(ctx context.Context, chainID eth.ChainID, number uint64) (eth.BlockID, error) {
	seal, err := b.chainDBs.FindSealedBlock(chainID, number)
	if err != nil {
		return eth.BlockID{}, err
	}
	return seal.ID(), nil
}

// AllSafeDerivedAt returns the last derived block for each chain, from the given L1 block
func (b *Backend) AllSafeDerivedAt(ctx context.Context, source eth.BlockID) (map[eth.ChainID]eth.BlockID, error) {
	chains := b.cfgSet.Chains()
	ret := map[eth.ChainID]eth.BlockID{}

	// Note: no need to reorg/rewind lock: everything is derived from the same L1 block
	for _, chainID := range chains {
		derived, err := b.LocalSafeDerivedAt(ctx, chainID, source)
		if err != nil {
			return nil, fmt.Errorf("failed to get last derived block for chain %v: %w", chainID, err)
		}
		ret[chainID] = derived
	}
	return ret, nil
}

func (b *Backend) Finalized(ctx context.Context, chainID eth.ChainID) (eth.BlockID, error) {
	v, err := b.chainDBs.Finalized(chainID)
	if err != nil {
		return eth.BlockID{}, err
	}
	return v.Derived.ID(), nil
}

func (b *Backend) FinalizedL1(ctx context.Context) (eth.BlockRef, error) {
	v := b.chainDBs.FinalizedL1()
	if v == (eth.BlockRef{}) {
		return eth.BlockRef{}, fmt.Errorf("finality of L1 is not initialized: %w", ethereum.NotFound)
	}
	return v, nil
}

func (b *Backend) ActivationBlock(ctx context.Context, chainID eth.ChainID) (types.DerivedBlockSealPair, error) {
	return b.chainDBs.AnchorPoint(chainID)
}

func (b *Backend) IsLocalUnsafe(ctx context.Context, chainID eth.ChainID, block eth.BlockID) error {
	return b.chainDBs.IsLocalUnsafe(chainID, block)
}

func (b *Backend) IsCrossSafe(ctx context.Context, chainID eth.ChainID, block eth.BlockID) error {
	return b.chainDBs.IsCrossSafe(chainID, block)
}

func (b *Backend) IsLocalSafe(ctx context.Context, chainID eth.ChainID, block eth.BlockID) error {
	return b.chainDBs.IsLocalSafe(chainID, block)
}

func (b *Backend) CrossDerivedToSource(ctx context.Context, chainID eth.ChainID, derived eth.BlockID) (source eth.BlockRef, err error) {
	v, err := b.chainDBs.CrossDerivedToSourceRef(chainID, derived)
	if err != nil {
		return eth.BlockRef{}, err
	}
	return v, nil
}

func (b *Backend) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	return b.l1Accessor.L1BlockRefByNumber(ctx, number)
}

func (m *Backend) findL2(chainID eth.ChainID, num uint64) apis.L2EthExtendedClient {
	// Find a REL or RWEL that is in sync enough to serve our queries.

	// Check all read-write sources
	for _, v := range m.rwels.Values() {
		if v.ChainID() == chainID && v.IsSyncedTo(num) {
			cl, ok := m.rwL2Clients.Get(v.ID())
			if ok {
				return cl
			}
		}
	}

	// Check all read-only sources
	for _, v := range m.rels.Values() {
		if v.ChainID() == chainID && v.IsSyncedTo(num) {
			cl, ok := m.readL2Clients.Get(v.ID())
			if ok {
				return cl
			}
		}
	}

	return nil
}

func (m *Backend) L2BlockRefByTimestamp(ctx context.Context, chainID eth.ChainID, timestamp uint64) (eth.L2BlockRef, error) {
	if !m.cfgSet.HasChain(chainID) {
		return eth.L2BlockRef{}, types.ErrUnknownChain
	}
	rollupCfg := m.rollupConfigs.RollupConfig(chainID)
	num, err := rollupCfg.TargetBlockNumber(timestamp)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	l2Src := m.findL2(chainID, num)
	if l2Src == nil {
		return eth.L2BlockRef{}, fmt.Errorf("unavailable L2 EL for chain %s", chainID)
	}
	return l2Src.L2BlockRefByNumber(ctx, num)
}

func (m *Backend) OutputV0AtTimestamp(ctx context.Context, chainID eth.ChainID, timestamp uint64) (*eth.OutputV0, error) {
	ref, err := m.L2BlockRefByTimestamp(ctx, chainID, timestamp)
	if err != nil {
		return nil, err
	}
	l2Src := m.findL2(chainID, ref.Number)
	if l2Src == nil {
		return nil, fmt.Errorf("unavailable L2 EL for chain %s", chainID)
	}
	return l2Src.OutputV0AtBlock(ctx, ref.Hash)
}

func (m *Backend) PendingOutputV0AtTimestamp(ctx context.Context, chainID eth.ChainID, timestamp uint64) (*eth.OutputV0, error) {
	ref, err := m.L2BlockRefByTimestamp(ctx, chainID, timestamp)
	if err != nil {
		return nil, err
	}
	l2Src := m.findL2(chainID, ref.Number)
	if l2Src == nil {
		return nil, fmt.Errorf("unavailable L2 EL for chain %s", chainID)
	}
	if ref.Number == 0 {
		// The genesis block cannot have been invalid
		return l2Src.OutputV0AtBlock(ctx, ref.Hash)
	}

	payload, err := l2Src.PayloadByHash(ctx, ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block (%v): %w", ref, err)
	}
	optimisticOutput, err := indexing.DecodeInvalidatedBlockTxFromReplacement(payload.ExecutionPayload.Transactions)
	if errors.Is(err, indexing.ErrNotReplacementBlock) {
		// This block was not replaced so use the canonical output root as pending
		return l2Src.OutputV0AtBlock(ctx, ref.Hash)
	} else if err != nil {
		return nil, fmt.Errorf("failed parse replacement block (%v): %w", ref, err)
	}
	return optimisticOutput, nil
}

func (b *Backend) SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error) {
	chains := b.cfgSet.Chains()
	slices.SortFunc(chains, func(a, b eth.ChainID) int {
		return a.Cmp(b)
	})
	chainInfos := make([]eth.ChainRootInfo, len(chains))
	superRootChains := make([]eth.ChainIDAndOutput, len(chains))

	h := b.chainDBs.AcquireHandle()
	defer h.Release()
	h.DependOnDerivedTime(uint64(timestamp))

	var crossSafeSource eth.BlockID
	for i, chainID := range chains {
		output, err := b.OutputV0AtTimestamp(ctx, chainID, uint64(timestamp))
		if err != nil {
			return eth.SuperRootResponse{}, err
		}
		pending, err := b.PendingOutputV0AtTimestamp(ctx, chainID, uint64(timestamp))
		if err != nil {
			return eth.SuperRootResponse{}, err
		}
		canonicalRoot := eth.OutputRoot(output)
		chainInfos[i] = eth.ChainRootInfo{
			ChainID:   chainID,
			Canonical: canonicalRoot,
			Pending:   pending.Marshal(),
		}
		superRootChains[i] = eth.ChainIDAndOutput{ChainID: chainID, Output: canonicalRoot}

		ref, err := b.L2BlockRefByTimestamp(ctx, chainID, uint64(timestamp))
		if err != nil {
			return eth.SuperRootResponse{}, err
		}
		source, err := b.chainDBs.CrossDerivedToSource(chainID, ref.ID())
		if err != nil {
			// Transform error to ethereum.NotFound at RPC boundary so that the challenger can detect this case
			if errors.Is(err, types.ErrFuture) {
				err = errors.Join(err, ethereum.NotFound)
			}
			return eth.SuperRootResponse{}, fmt.Errorf("cross-derived-to-source failed for chain %s: %w", chainID, err)
		}
		h.DependOnSourceBlock(source.Number)
		if crossSafeSource.Number == 0 || crossSafeSource.Number < source.Number {
			crossSafeSource = source.ID()
		}
	}

	if !h.IsValid() {
		return eth.SuperRootResponse{}, h.Err()
	}
	super := eth.SuperV1{
		Timestamp: uint64(timestamp),
		Chains:    superRootChains,
	}
	superRoot := eth.SuperRoot(&super)
	return eth.SuperRootResponse{
		CrossSafeDerivedFrom: crossSafeSource,
		Timestamp:            uint64(timestamp),
		SuperRoot:            superRoot,
		Version:              super.Version(),
		Chains:               chainInfos,
	}, nil
}

func (b *Backend) SyncStatus(ctx context.Context) (eth.SupervisorSyncStatus, error) {
	return b.statusTracker.SyncStatus()
}
