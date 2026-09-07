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

// SyncStatus serves optimism_syncStatus. See Module.SyncStatus for the field population, and the
// package comment for why an error before the first claim is the right answer rather than a gap.
func (a *API) SyncStatus(_ context.Context) (*sources.FollowSyncStatus, error) {
	a.m.mu.RLock()
	defer a.m.mu.RUnlock()
	status, err := a.m.syncStatusLocked()
	if err != nil {
		return nil, err
	}
	out := &sources.FollowSyncStatus{SyncStatus: *status}
	if a.m.recoveryTarget != (eth.L2BlockRef{}) && a.m.recoveryTarget.Number >= a.m.localSafe.Number {
		out.Recovery = &sources.FollowRecoveryStatus{Anchor: a.m.localSafe, Target: a.m.recoveryTarget, Safe: a.m.recoverySafe, Finalized: a.m.recoveryFinalized}
		for _, c := range a.m.pending {
			if c.invalidFrom != 0 && c.prefixRef.Number > out.Recovery.Anchor.Number && c.prefixRef.Number <= out.Recovery.Target.Number &&
				(out.Recovery.Prefix == nil || c.prefixRef.Number > out.Recovery.Prefix.Last.Number) {
				out.Recovery.Prefix = &sources.FollowRecoveryPrefix{
					Terminal: eth.BlockID{Hash: c.terminal, Number: c.last}, TerminalParent: c.parent, Last: c.prefixRef,
				}
			}
		}
	}
	return out, nil
}

// RecoveryBlock supplies only canonical deposit-only projection inputs. It never
// returns a private block hash or treats an unavailable claim as an empty block.
func (a *API) RecoveryBlock(ctx context.Context, number uint64, target eth.BlockID) (eth.L2BlockRef, error) {
	src, err := a.m.source()
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	a.m.mu.RLock()
	anchor, frontier, generation := a.m.localSafe, a.m.recoveryTarget, a.m.generation
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
	if a.m.generation != generation || a.m.recoveryTarget.Number < target.Number || a.m.localSafe.Number >= number {
		return eth.L2BlockRef{}, fmt.Errorf("projection recovery snapshot was revoked")
	}
	return ref, nil
}
