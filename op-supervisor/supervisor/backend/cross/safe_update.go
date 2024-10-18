package cross

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type CrossSafeDeps interface {
	CrossSafe(chainID types.ChainID) (types.BlockSeal, error)

	SafeFrontierCheckDeps
	SafeStartDeps

	OpenBlock(chainID types.ChainID, blockNum uint64) (seal types.BlockSeal, logCount uint32, execMsgs []*types.ExecutingMessage, err error)
}

func CrossSafeUpdate(ctx context.Context, logger log.Logger, chainID types.ChainID, d CrossSafeDeps) error {
	// TODO establish L1 reorg-lock of scopeDerivedFrom
	// defer unlock once we are done checking the chain

	// fetch cross-head
	crossSafe, err := d.CrossSafe(chainID)
	if err != nil {
		// TODO handle genesis case
	}

	// TODO: this might not be right, just pulling scopeDerivedFrom out of the signature
	scopeDerivedFrom, err := d.CrossDerivedFrom(chainID, crossSafe.ID())

	// open block N+1
	candidate, _, execMsgs, err := d.OpenBlock(chainID, crossSafe.Number+1)
	if err != nil {
		return fmt.Errorf("failed to open block %d: %w", crossSafe.Number+1, err)
	}
	derivedFrom, err := d.LocalDerivedFrom(chainID, candidate.ID())
	if err != nil {
		// TODO
	}
	if derivedFrom.Number > scopeDerivedFrom.Number {
		return fmt.Errorf("next candidate block %s is derived from %s, outside of scope %s", candidate, derivedFrom, scopeDerivedFrom)
	}

	hazards, err := CrossSafeHazards(d, chainID, scopeDerivedFrom, candidate, execMsgs)
	if err != nil {
		// TODO
	}
	_ = hazards
	//if err := HazardSafeFrontierChecks(d, scopeDerivedFrom, hazards); err != nil {
	//	// TODO
	//}
	//if err := HazardCycleChecks(d, candidate.Timestamp, hazards); err != nil {
	// TODO
	//}
	// TODO promote the candidate block to cross-safe
	return nil
}

func NewCrossSafeWorker(logger log.Logger, chainID types.ChainID, d CrossSafeDeps) *Worker {
	return NewWorker(logger, func(ctx context.Context) error {
		return CrossSafeUpdate(ctx, logger, chainID, d)
	})
}
