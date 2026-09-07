package claimfollow

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type replacementHistory interface {
	DeniedBlocksAtHeight(uint64) ([]common.Hash, error)
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayloadEnvelope, error)
}

// checkReplacement reconstructs partial invalidation after a follower restart
// from the supernode's existing durable deny list. A denial at the same height
// is insufficient: the denied block must extend the surviving canonical parent.
func (m *Module) checkReplacement(ctx context.Context, src Rendering, payload *eth.ExecutionPayload, generation uint64) error {
	n := uint64(payload.BlockNumber)
	m.mu.RLock()
	var covered bool
	for _, c := range m.pending {
		if c.carrier < n && n <= c.last && (c.replacementFrom == 0 || n < c.replacementFrom) {
			covered = true
			break
		}
	}
	m.mu.RUnlock()
	if !covered {
		return nil
	}
	history, ok := src.(replacementHistory)
	if !ok {
		return fmt.Errorf("projection source does not expose replacement history")
	}
	denied, err := history.DeniedBlocksAtHeight(n)
	if err != nil {
		return err
	}
	for _, hash := range denied {
		if hash == payload.BlockHash {
			continue
		}
		old, err := history.PayloadByHash(ctx, hash)
		if err != nil {
			return fmt.Errorf("reading denied projection header %s: %w", hash, err)
		}
		if old == nil || old.ExecutionPayload == nil || old.ExecutionPayload.BlockHash != hash || uint64(old.ExecutionPayload.BlockNumber) != n {
			return fmt.Errorf("denied projection header is unavailable or inconsistent")
		}
		if old.ExecutionPayload.ParentHash != payload.ParentHash {
			continue
		}
		parent, err := src.PayloadByNumber(ctx, n-1)
		if err != nil {
			return err
		}
		if parent == nil || parent.ExecutionPayload == nil || parent.ExecutionPayload.BlockHash != payload.ParentHash {
			return fmt.Errorf("canonical replacement parent changed")
		}
		prefix, err := derive.PayloadToBlockRef(m.rollupCfg, parent.ExecutionPayload)
		if err != nil {
			return err
		}
		m.mu.Lock()
		if generation != m.generation {
			m.mu.Unlock()
			return fmt.Errorf("claim history changed during replacement lookup")
		}
		for _, c := range m.pending {
			if c.carrier < n && n <= c.last && (c.replacementFrom == 0 || n < c.replacementFrom) {
				c.invalidFrom, c.prefixRef, c.completed, c.replacementFrom, c.revokedTip = n, prefix, false, n, eth.BlockID{}
			}
		}
		m.mu.Unlock()
		return nil
	}
	return nil
}
