package claimfollow

import (
	"context"
	"fmt"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// API serves private checkpoints and canonical recovery inputs on the sibling
// <chainID>/claimed route. The chain's ordinary route remains its public projection.
type API struct {
	m *Module
}

// NewAPI wraps a module as the "optimism" namespace service.
func NewAPI(m *Module) *API { return &API{m: m} }

// SyncStatus serves the last fully checked snapshot through optimism_syncStatus.
func (a *API) SyncStatus(_ context.Context) (*sources.FollowSyncStatus, error) {
	a.m.mu.RLock()
	defer a.m.mu.RUnlock()
	status, err := a.m.syncStatusLocked()
	if err != nil {
		return nil, err
	}
	out := a.m.view
	out.SyncStatus = *status
	return &out, nil
}

// RecoveryBlock supplies only canonical deposit-only projection inputs. It never
// returns a private block hash or treats an unavailable claim as an empty block.
func (a *API) RecoveryBlock(ctx context.Context, number uint64, target eth.BlockID) (eth.L2BlockRef, error) {
	src, err := a.m.source()
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	a.m.mu.RLock()
	plan, generation := a.m.view.Recovery, a.m.generation
	anchor := a.m.view.LocalSafeL2
	var frontier eth.L2BlockRef
	if plan != nil {
		frontier = plan.Target
	}
	a.m.mu.RUnlock()
	if number <= anchor.Number || number > target.Number || target.Number > frontier.Number {
		return eth.L2BlockRef{}, fmt.Errorf("block %d is outside the available recovery suffix", number)
	}
	checkTarget := func() error {
		env, err := src.PayloadByNumber(ctx, target.Number)
		if err != nil {
			return err
		}
		if env == nil || env.ExecutionPayload == nil || env.ExecutionPayload.ID() != target {
			return fmt.Errorf("projection recovery target changed")
		}
		return nil
	}
	if err := checkTarget(); err != nil {
		return eth.L2BlockRef{}, err
	}
	env, err := src.PayloadByNumber(ctx, number)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if env == nil || env.ExecutionPayload == nil {
		return eth.L2BlockRef{}, fmt.Errorf("projection block %d is unavailable", number)
	}
	for _, tx := range env.ExecutionPayload.Transactions {
		if len(tx) == 0 || (tx[0] != optypes.DepositTxType && tx[0] != optypes.PostExecTxType) {
			return eth.L2BlockRef{}, fmt.Errorf("projection block %d contains sequencer transactions", number)
		}
	}
	ref, err := derive.PayloadToBlockRef(a.m.rollupCfg, env.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if err := checkTarget(); err != nil {
		return eth.L2BlockRef{}, err
	}
	a.m.mu.RLock()
	defer a.m.mu.RUnlock()
	if a.m.generation != generation || a.m.view.Recovery == nil || a.m.view.Recovery.Target.Number < target.Number || a.m.view.LocalSafeL2.Number >= number {
		return eth.L2BlockRef{}, fmt.Errorf("projection recovery snapshot was revoked")
	}
	return ref, nil
}
