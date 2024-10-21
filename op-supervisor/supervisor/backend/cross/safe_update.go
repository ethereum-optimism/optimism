package cross

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type CrossSafeDeps interface {
	CrossSafe(chainID types.ChainID) (derivedFrom types.BlockSeal, derived types.BlockSeal, err error)

	SafeFrontierCheckDeps
	SafeStartDeps

	CandidateCrossSafe(chain types.ChainID) (derivedFromScope, crossSafe eth.BlockRef, err error)
	NextDerivedFrom(chain types.ChainID, derivedFrom eth.BlockID) (after eth.BlockRef, err error)
	PreviousDerived(chain types.ChainID, derived eth.BlockID) (prevDerived types.BlockSeal, err error)

	OpenBlock(chainID types.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs []*types.ExecutingMessage, err error)

	UpdateCrossSafe(chain types.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error
}

func CrossSafeUpdate(ctx context.Context, logger log.Logger, chainID types.ChainID, d CrossSafeDeps) error {
	// TODO(#11693): establish L1 reorg-lock of scopeDerivedFrom
	// defer unlock once we are done checking the chain
	candidateScope, err := scopedCrossSafeUpdate(chainID, d)
	if err == nil {
		return nil
	}
	if !errors.Is(err, types.ErrOutOfScope) {
		return err
	}
	// bump the L1 scope up, and repeat the prev L2 block, not the candidate
	newScope, err := d.NextDerivedFrom(chainID, candidateScope.ID())
	if err != nil {
		return fmt.Errorf("failed to identify new L1 scope to expand to after %s: %w", candidateScope, err)
	}
	_, currentCrossSafe, err := d.CrossSafe(chainID)
	if err != nil {
		// TODO: if genesis isn't cross-safe by default, then we can't register something as cross-safe here
		return fmt.Errorf("failed to identify cross-safe scope to repeat: %w", err)
	}
	parent, err := d.PreviousDerived(chainID, currentCrossSafe.ID())
	if err != nil {
		return fmt.Errorf("cannot find parent-block of cross-safe: %w", err)
	}
	crossSafeRef := currentCrossSafe.WithParent(parent.ID())
	if err := d.UpdateCrossSafe(chainID, newScope, crossSafeRef); err != nil {
		return fmt.Errorf("failed to update cross-safe head with L1 scope increment to %s and repeat of L2 block %s: %w", candidateScope, crossSafeRef, err)
	}
	return nil
}

func scopedCrossSafeUpdate(chainID types.ChainID, d CrossSafeDeps) (scope eth.BlockRef, err error) {
	candidateScope, candidate, err := d.CandidateCrossSafe(chainID)
	if err != nil {
		return eth.BlockRef{}, fmt.Errorf("failed to determine candidate block for cross-safe: %w", err)
	}
	opened, _, execMsgs, err := d.OpenBlock(chainID, candidate.Number)
	if err != nil {
		return eth.BlockRef{}, fmt.Errorf("failed to open block %s: %w", candidate, err)
	}
	if opened.ID() != candidate.ID() {
		return eth.BlockRef{}, fmt.Errorf("unsafe L2 DB has %s, but candidate cross-safe was %s: %w", opened, candidate, types.ErrConflict)
	}
	hazards, err := CrossSafeHazards(d, chainID, candidateScope.ID(), types.BlockSealFromRef(opened), execMsgs)
	if err != nil {
		return eth.BlockRef{}, fmt.Errorf("failed to determine dependencies of cross-safe candidate %s: %w", candidate, err)
	}
	if err := HazardSafeFrontierChecks(d, candidateScope.ID(), hazards); err != nil {
		return eth.BlockRef{}, fmt.Errorf("failed to verify block %s in cross-safe frontier: %w", candidate, err)
	}
	//if err := HazardCycleChecks(d, candidate.Timestamp, hazards); err != nil {
	// TODO
	//}

	// promote the candidate block to cross-safe
	if err := d.UpdateCrossSafe(chainID, candidateScope, candidate); err != nil {
		return eth.BlockRef{}, fmt.Errorf("failed to update cross-safe head to %s, derived from scope %s: %w", candidate, candidateScope, err)
	}
	return candidateScope, nil
}

func NewCrossSafeWorker(logger log.Logger, chainID types.ChainID, d CrossSafeDeps) *Worker {
	logger = logger.New("chain", chainID)
	return NewWorker(logger, func(ctx context.Context) error {
		return CrossSafeUpdate(ctx, logger, chainID, d)
	})
}
