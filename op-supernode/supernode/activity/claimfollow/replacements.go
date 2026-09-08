package claimfollow

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type replacementHistory interface {
	DeniedBlocksInRange(uint64, uint64) ([]eth.BlockID, error)
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayloadEnvelope, error)
}

// surviving resolves the claim's original branch against today's canonical
// projection, then clips it at the first relevant denial. No revocation or
// restoration flags survive a branch change. Denials are a sparse snapshot shared
// by all claims, independent of the length of the following deposit-only outage.
func (m *Module) surviving(ctx context.Context, src Rendering, c claim, frontier uint64, denied []eth.BlockID) (uint64, error) {
	history, ok := src.(replacementHistory)
	if !ok {
		return 0, fmt.Errorf("projection source does not expose replacement history")
	}
	end := min(c.tip.Number, frontier)
	old := c.tip
	for {
		if old.Number <= end {
			canonical, err := projectionRef(ctx, src, m.rollupCfg, old.Number)
			if err != nil {
				return 0, err
			}
			if canonical.ID() == old {
				end = old.Number
				break
			}
		}
		if old.Number <= c.carrier {
			return 0, fmt.Errorf("claim carrier changed during resolution")
		}
		env, err := history.PayloadByHash(ctx, old.Hash)
		if err != nil {
			return 0, err
		}
		if env == nil || env.ExecutionPayload == nil || env.ExecutionPayload.ID() != old {
			return 0, fmt.Errorf("original projection header is unavailable or inconsistent")
		}
		old = eth.BlockID{Hash: env.ExecutionPayload.ParentHash, Number: old.Number - 1}
	}
	for _, id := range denied {
		if id.Number <= c.carrier || id.Number > end {
			continue
		}
		canonical, err := projectionRef(ctx, src, m.rollupCfg, id.Number)
		if err != nil {
			return 0, err
		}
		env, err := history.PayloadByHash(ctx, id.Hash)
		if err != nil {
			return 0, fmt.Errorf("reading denied projection header: %w", err)
		}
		if env == nil || env.ExecutionPayload == nil || env.ExecutionPayload.ID() != id {
			return 0, fmt.Errorf("denied projection header is unavailable or inconsistent")
		}
		if env.ExecutionPayload.ParentHash == canonical.ParentHash {
			end = id.Number - 1
		}
	}
	return end, nil
}

// recoveryFrontier excludes a denied canonical block and all descendants, even
// while the engine has not completed its rewind. The same sparse denial snapshot
// also identifies replaced suffixes inside individual claims.
func (m *Module) recoveryFrontier(ctx context.Context, src Rendering, status *eth.SyncStatus, finalized uint64) (*eth.SyncStatus, []eth.BlockID, error) {
	history, ok := src.(replacementHistory)
	if !ok {
		return nil, nil, fmt.Errorf("projection source does not expose replacement history")
	}
	denied, err := history.DeniedBlocksInRange(finalized, status.LocalSafeL2.Number)
	if err != nil {
		return nil, nil, err
	}
	out := *status
	for _, id := range denied {
		if id.Number > out.LocalSafeL2.Number {
			continue
		}
		ref, err := projectionRef(ctx, src, m.rollupCfg, id.Number)
		if err != nil {
			return nil, nil, err
		}
		if ref.ID() != id {
			continue
		}
		if id.Number <= finalized {
			return nil, nil, ErrInvariant
		}
		parent, err := projectionRef(ctx, src, m.rollupCfg, id.Number-1)
		if err != nil {
			return nil, nil, err
		}
		out.LocalSafeL2 = parent
		if out.SafeL2.Number > parent.Number {
			out.SafeL2 = parent
		}
		if out.FinalizedL2.Number > parent.Number {
			out.FinalizedL2 = parent
		}
	}
	return &out, denied, nil
}
