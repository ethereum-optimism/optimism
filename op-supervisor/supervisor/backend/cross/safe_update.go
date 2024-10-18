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
	CrossSafe(chainID types.ChainID) (types.BlockSeal, error)

	SafeFrontierCheckDeps
	SafeStartDeps

	CandidateCrossSafe(chain types.ChainID) (derivedFromScope, crossSafe types.BlockSeal, err error)

	OpenBlock(chainID types.ChainID, blockNum uint64) (seal types.BlockSeal, logCount uint32, execMsgs []*types.ExecutingMessage, err error)

	UpdateCrossSafe(chain types.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error
}

func CrossSafeUpdate(ctx context.Context, logger log.Logger, chainID types.ChainID, d CrossSafeDeps) error {
	// TODO(#11693): establish L1 reorg-lock of scopeDerivedFrom
	// defer unlock once we are done checking the chain

	candidateScope, candidate, err := d.CandidateCrossSafe(chainID)
	if err != nil {
		return fmt.Errorf("failed to determine candidate block for cross-safe: %w", err)
	}

	if err := scopedCrossSafeUpdate(candidateScope, candidate, chainID, d); err != nil {
		if errors.Is(err, types.ErrOutOfScope) {
			// TODO bump the L1 scope up, and repeat the prev L2 block, not the candidate
		} else {
			return err
		}
	}

	// promote the candidate block to cross-safe
	if err := d.UpdateCrossSafe(chainID, candidateScope, candidate); err != nil {
		return fmt.Errorf("failed to update cross-safe head to %s, derived from scope %s: %w", candidate, candidateScope, err)
	}
	return nil
}

func scopedCrossSafeUpdate(candidateScope, candidate types.BlockSeal, chainID types.ChainID, d CrossSafeDeps) error {
	candidate, _, execMsgs, err := d.OpenBlock(chainID, candidate.Number)
	if err != nil {
		return fmt.Errorf("failed to open block %s: %w", candidate, err)
	}
	hazards, err := CrossSafeHazards(d, chainID, candidateScope.ID(), candidate, execMsgs)
	if err != nil {
		return fmt.Errorf("failed to determine dependencies of cross-safe candidate %s: %w", candidate, err)
	}
	if err := HazardSafeFrontierChecks(d, candidateScope.ID(), hazards); err != nil {
		return fmt.Errorf("failed to verify block %s in cross-safe frontier: %w", candidate, err)
	}
	//if err := HazardCycleChecks(d, candidate.Timestamp, hazards); err != nil {
	// TODO
	//}
	return nil
}

func NewCrossSafeWorker(logger log.Logger, chainID types.ChainID, d CrossSafeDeps) *Worker {
	return NewWorker(logger, func(ctx context.Context) error {
		return CrossSafeUpdate(ctx, logger, chainID, d)
	})
}
