package cross

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type CrossUnsafeDeps interface {
	CrossUnsafe(chainID types.ChainID) (types.BlockSeal, error)

	UnsafeStartDeps
	UnsafeFrontierCheckDeps

	OpenBlock(chainID types.ChainID, blockNum uint64) (seal types.BlockSeal, logCount uint32, execMsgs []*types.ExecutingMessage, err error)
}

func CrossUnsafeUpdate(ctx context.Context, logger log.Logger, chainID types.ChainID, d CrossUnsafeDeps) error {
	var candidate types.BlockSeal
	var execMsgs []*types.ExecutingMessage

	// fetch cross-head
	crossSafe, err := d.CrossUnsafe(chainID)
	if err != nil {
		if errors.Is(err, types.ErrFuture) {
			// If genesis / no cross-safe block yet, then start with block 0
			// TODO
		} else {
			return err
		}
	} else {
		// Open block N+1: this is a local-unsafe block,
		// just after cross-safe, that can be promoted if it passes the dependency checks.
		candidate, _, execMsgs, err = d.OpenBlock(chainID, crossSafe.Number+1)
		if err != nil {
			return fmt.Errorf("failed to open block %d: %w", crossSafe.Number+1, err)
		}
	}

	hazards, err := CrossUnsafeHazards(d, chainID, candidate, execMsgs)
	if err != nil {
		// TODO(#11693): reorgs can be detected by checking if the error is ErrConflict,
		// missing data is identified by ErrFuture,
		// and other errors (e.g. DB issues) are identifier by remaining error kinds.
		return fmt.Errorf("failed to check for cross-chain hazards: %w", err)
	}
	// TODO apply hazard checks
	_ = hazards
	//if err := HazardUnsafeFrontierChecks(d, hazards); err != nil {
	//	// TODO
	//}
	//if err := HazardCycleChecks(d, candidate.Timestamp, hazards); err != nil {
	//// TODO
	//}
	// TODO promote the candidate block to cross-unsafe
	return nil
}

func NewCrossUnsafeWorker(logger log.Logger, chainID types.ChainID, d CrossUnsafeDeps) *Worker {
	return NewWorker(logger, func(ctx context.Context) error {
		return CrossUnsafeUpdate(ctx, logger, chainID, d)
	})
}
