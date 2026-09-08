// Package claimfollow maps canonical projection history to private checkpoints
// and a deposit-only replay interval. Claims are operator attestations; the
// projection supplies their canonical position and block schedule.
package claimfollow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
)

var (
	ErrNoGenesisRef = errors.New("claim follow module has no genesis ref and has not read a claim yet")
	ErrInvariant    = errors.New("claim follow snapshot contradicts finalized history")
)

const DefaultMaxBlocksPerPoll = 512

type Rendering interface {
	SyncStatus(context.Context) (*eth.SyncStatus, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
	FetchReceipts(context.Context, eth.BlockID) (eth.BlockInfo, optypes.Receipts, error)
}

// Config selects the projection registry, private genesis and scan budget.
type Config struct {
	Registry         common.Address
	GenesisHash      common.Hash // Private genesis; all other inputs come from the projection.
	StartBlock       uint64      // Zero scans from genesis.
	MaxBlocksPerPoll uint64      // Zero uses DefaultMaxBlocksPerPoll.
}

// A claim arrives at carrier before its range ends at last. tip identifies its
// observed projection branch; validity is recomputed from canonical history.
type claim struct {
	carrier, last    uint64
	tip              eth.BlockID
	terminal, parent common.Hash
}

// Step runs on the poll loop; the mutex protects snapshots served by RPC handlers.
type Module struct {
	cfg                 Config
	rollupCfg           *rollup.Config
	log                 gethlog.Logger
	metrics             Metrics
	mu                  sync.RWMutex
	rendering           Rendering
	next                uint64
	cursor              eth.BlockID
	history             map[uint64]eth.L2BlockRef
	pending             []claim
	generation          uint64
	view                sources.FollowSyncStatus
	finalizedProjection eth.BlockID
	invariantErr        error
}

// New initializes the private checkpoint without requiring a first claim.
func New(cfg Config, rollupCfg *rollup.Config, log gethlog.Logger, metrics Metrics) *Module {
	if cfg.MaxBlocksPerPoll == 0 {
		cfg.MaxBlocksPerPoll = DefaultMaxBlocksPerPoll
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	m := &Module{cfg: cfg, rollupCfg: rollupCfg, log: log, metrics: metrics, next: cfg.StartBlock, history: make(map[uint64]eth.L2BlockRef)}
	if cfg.GenesisHash != (common.Hash{}) {
		ref := genesisRef(cfg.GenesisHash, rollupCfg)
		m.view.SyncStatus = eth.SyncStatus{LocalSafeL2: ref, SafeL2: ref, FinalizedL2: ref}
	}
	return m
}
func genesisRef(hash common.Hash, cfg *rollup.Config) eth.L2BlockRef {
	return eth.L2BlockRef{Hash: hash, Number: cfg.Genesis.L2.Number, Time: cfg.Genesis.L2Time, L1Origin: cfg.Genesis.L1}
}
func (m *Module) Attach(r Rendering) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rendering = r
}
func (m *Module) source() (Rendering, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.rendering == nil {
		return nil, errors.New("claim follow source is not attached")
	}
	return m.rendering, nil
}
func (m *Module) SyncStatus() (*eth.SyncStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.syncStatusLocked()
}
func (m *Module) syncStatusLocked() (*eth.SyncStatus, error) {
	if m.invariantErr != nil {
		return nil, m.invariantErr
	}
	if m.view.LocalSafeL2.Hash == (common.Hash{}) {
		return nil, ErrNoGenesisRef
	}
	out := m.view.SyncStatus
	return &out, nil
}

// Rewind only the changed canonical suffix. A mere safety-label retreat does not
// call this: its already scanned blocks remain available if safety advances again.
func (m *Module) rewind(h uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rewindLocked(h)
}

func (m *Module) rewindLocked(h uint64) {
	m.generation++
	m.view.Recovery = nil
	if m.next == 0 {
		return
	}
	h = min(h, m.next-1)
	finalized := m.view.FinalizedL2
	if h < finalized.Number {
		m.invariantErr = ErrInvariant
		return
	}
	m.next = max(m.cfg.StartBlock, h+1)
	ref, known := m.history[h]
	if !known {
		m.invariantErr = fmt.Errorf("%w: reset passed retained projection history", ErrInvariant)
		return
	}
	m.cursor = ref.ID()
	kept := m.pending[:0]
	for _, c := range m.pending {
		if c.carrier <= h {
			kept = append(kept, c)
		}
	}
	m.pending = kept
	for n := range m.history {
		if n > h {
			delete(m.history, n)
		}
	}
	m.view.LocalSafeL2, m.view.SafeL2 = finalized, finalized
}

func projectionRef(ctx context.Context, src Rendering, cfg *rollup.Config, n uint64) (eth.L2BlockRef, error) {
	env, err := src.PayloadByNumber(ctx, n)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if env == nil || env.ExecutionPayload == nil || uint64(env.ExecutionPayload.BlockNumber) != n {
		return eth.L2BlockRef{}, fmt.Errorf("projection block %d is unavailable or inconsistent", n)
	}
	return derive.PayloadToBlockRef(cfg, env.ExecutionPayload)
}

func (m *Module) Step(ctx context.Context) error {
	src, err := m.source()
	if err != nil {
		return err
	}
	// Capture the generation BEFORE sampling upstream status. Reset can race any RPC.
	m.mu.RLock()
	generation, cursor := m.generation, m.cursor
	m.mu.RUnlock()
	status, err := src.SyncStatus(ctx)
	if err != nil {
		return err
	}
	if status == nil {
		return errors.New("empty projection sync status")
	}
	if status.FinalizedL2.Number > status.SafeL2.Number || status.SafeL2.Number > status.LocalSafeL2.Number {
		return ErrInvariant
	}
	m.mu.RLock()
	unchanged := generation == m.generation
	m.mu.RUnlock()
	if !unchanged {
		return fmt.Errorf("projection frontier changed during scan")
	}
	if cursor != (eth.BlockID{}) {
		// Compare the common height first: a lower local-safe label on the same
		// branch is not an execution invalidation.
		for n := min(cursor.Number, status.LocalSafeL2.Number); ; n-- {
			m.mu.RLock()
			old, known := m.history[n]
			m.mu.RUnlock()
			if !known {
				return fmt.Errorf("%w: reorg passed retained projection history", ErrInvariant)
			}
			ref, err := projectionRef(ctx, src, m.rollupCfg, n)
			if err != nil {
				return err
			}
			if ref == old {
				if n < min(cursor.Number, status.LocalSafeL2.Number) {
					m.metrics.RecordRenderingReorg()
					m.mu.Lock()
					if generation != m.generation {
						m.mu.Unlock()
						return fmt.Errorf("projection frontier changed during scan")
					}
					m.rewindLocked(n)
					generation = m.generation
					m.mu.Unlock()
				}
				break
			}
			if n == 0 {
				return ErrInvariant
			}
		}
	}
	if err := m.scan(ctx, src, status.LocalSafeL2.Number, generation); err != nil {
		return err
	}
	m.mu.RLock()
	scanned := m.next > status.LocalSafeL2.Number
	claims := append([]claim(nil), m.pending...)
	previous := m.view
	finalProjection := m.finalizedProjection
	m.mu.RUnlock()
	if !scanned {
		return nil
	}

	frontier, denied, err := m.recoveryFrontier(ctx, src, status, previous.FinalizedL2.Number)
	if err != nil {
		return err
	}

	// Build a fresh view. Only publish it after checking the complete upstream
	// snapshot still names the blocks used by this scan.
	out := sources.FollowSyncStatus{SyncStatus: eth.SyncStatus{
		LocalSafeL2: previous.FinalizedL2, SafeL2: previous.FinalizedL2,
		FinalizedL2: previous.FinalizedL2, CurrentL1: status.CurrentL1,
	}}
	if out.CurrentL1 == (eth.L1BlockRef{}) {
		out.CurrentL1 = previous.CurrentL1
	}
	plan := &sources.FollowRecoveryStatus{Target: frontier.LocalSafeL2, Safe: frontier.SafeL2, Finalized: frontier.FinalizedL2}
	for _, c := range claims {
		if c.carrier > frontier.LocalSafeL2.Number {
			continue
		}
		end, err := m.surviving(ctx, src, c, frontier.LocalSafeL2.Number, denied)
		if err != nil {
			return err
		}
		ref, err := projectionRef(ctx, src, m.rollupCfg, end)
		if err != nil {
			return err
		}
		if end == c.last {
			ref.Hash, ref.ParentHash = c.terminal, c.parent
			if ref.Number > out.LocalSafeL2.Number {
				out.LocalSafeL2 = ref
			}
			if ref.Number <= frontier.SafeL2.Number && ref.Number > out.SafeL2.Number {
				out.SafeL2 = ref
			}
			if ref.Number <= frontier.FinalizedL2.Number && ref.Number > out.FinalizedL2.Number {
				out.FinalizedL2 = ref
			}
		} else if end > out.LocalSafeL2.Number && (plan.Prefix == nil || end > plan.Prefix.Last.Number) {
			plan.Prefix = &sources.FollowRecoveryPrefix{Parent: eth.BlockID{Hash: c.parent, Number: c.last - 1}, Last: ref}
		}
	}
	plan.Anchor = out.LocalSafeL2
	if plan.Prefix != nil && plan.Prefix.Last.Number <= plan.Anchor.Number {
		plan.Prefix = nil
	}
	out.Recovery = plan
	for _, expected := range []eth.L2BlockRef{status.LocalSafeL2, status.SafeL2, status.FinalizedL2} {
		ref, err := projectionRef(ctx, src, m.rollupCfg, expected.Number)
		if err != nil {
			return err
		}
		if ref != expected {
			return fmt.Errorf("projection safety frontier changed during scan")
		}
	}
	retained, err := projectionRef(ctx, src, m.rollupCfg, previous.FinalizedL2.Number)
	if err != nil {
		return err
	}
	if finalProjection != (eth.BlockID{}) && retained.ID() != finalProjection {
		return ErrInvariant
	}
	final, err := projectionRef(ctx, src, m.rollupCfg, out.FinalizedL2.Number)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if generation != m.generation || m.history[status.LocalSafeL2.Number].ID() != status.LocalSafeL2.ID() {
		return fmt.Errorf("projection frontier changed during scan")
	}
	if out.FinalizedL2.Number > out.SafeL2.Number || out.SafeL2.Number > out.LocalSafeL2.Number {
		return ErrInvariant
	}
	m.view, m.finalizedProjection = out, final.ID()
	for n := range m.history {
		if n < min(frontier.FinalizedL2.Number, m.cursor.Number) {
			delete(m.history, n)
		}
	}
	kept := m.pending[:0]
	for _, c := range m.pending {
		if c.last > out.FinalizedL2.Number {
			kept = append(kept, c)
		}
	}
	m.pending = kept
	m.metrics.RecordSafe(out.LocalSafeL2.Number)
	if out.FinalizedL2.Number > previous.FinalizedL2.Number {
		m.metrics.RecordFinalized(out.FinalizedL2.Number)
	}
	return nil
}

func (m *Module) scan(ctx context.Context, src Rendering, tip, generation uint64) error {
	for budget := m.cfg.MaxBlocksPerPoll; budget > 0; budget-- {
		m.mu.RLock()
		n, cursor := m.next, m.cursor
		m.mu.RUnlock()
		if n > tip {
			return nil
		}
		env, err := src.PayloadByNumber(ctx, n)
		if err != nil {
			return err
		}
		if env == nil || env.ExecutionPayload == nil || uint64(env.ExecutionPayload.BlockNumber) != n {
			return fmt.Errorf("missing projection block %d", n)
		}
		payload := env.ExecutionPayload
		if cursor != (eth.BlockID{}) && payload.ParentHash != cursor.Hash {
			return fmt.Errorf("projection ancestry changed during scan")
		}
		claims, err := m.readClaims(ctx, src, payload)
		if err != nil {
			return err
		}

		ref, err := derive.PayloadToBlockRef(m.rollupCfg, payload)
		if err != nil {
			return err
		}

		m.mu.Lock()
		if generation != m.generation {
			m.mu.Unlock()
			return fmt.Errorf("projection reset during scan")
		}
		for i := range m.pending {
			c := &m.pending[i]
			if n <= c.last && c.tip.Number+1 == n && c.tip.Hash == payload.ParentHash {
				c.tip = payload.ID()
			}
		}
		m.pending = append(m.pending, claims...)
		for range claims {
			m.metrics.RecordClaim()
		}
		m.history[n] = ref
		m.next, m.cursor = n+1, payload.ID()
		m.mu.Unlock()
	}
	return nil
}

func (m *Module) readClaims(ctx context.Context, src Rendering, payload *eth.ExecutionPayload) ([]claim, error) {
	candidates := make(map[common.Hash]*codec.RangeClaim)
	var order []common.Hash
	for _, raw := range payload.Transactions {
		var tx types.Transaction
		if tx.UnmarshalBinary(raw) != nil || tx.To() == nil || *tx.To() != m.cfg.Registry {
			continue
		}
		if c, ok := m.decodeClaim(&tx); ok {
			candidates[tx.Hash()] = c
			order = append(order, tx.Hash())
		}
	}
	if len(order) == 0 {
		return nil, nil
	}
	succeeded, err := m.succeeded(ctx, src, payload.ID())
	if err != nil {
		return nil, err
	}
	var out []claim
	for _, hash := range order {
		c := candidates[hash]
		if !succeeded[hash] {
			m.metrics.RecordRejectedClaim("reverted")
			continue
		}
		out = append(out, claim{
			carrier: uint64(payload.BlockNumber), last: c.LastBlock, tip: payload.ID(),
			terminal: c.PrivateTerminalBlockHash, parent: c.PrivateTerminalParentHash,
		})
	}
	return out, nil
}

func (m *Module) decodeClaim(tx *types.Transaction) (*codec.RangeClaim, bool) {
	data := tx.Data()
	if len(data) < 4 || !bytes.Equal(data[:4], render.PostClaimSelector[:]) {
		m.log.Error("A registry-addressed transaction is not a postClaim call; skipping",
			"tx", tx.Hash(), "calldata", len(data))
		m.metrics.RecordRejectedClaim("selector")
		return nil, false
	}
	c, err := codec.Decode(data[4:])
	if err != nil {
		m.log.Error("A registry-addressed transaction does not carry a canonically-encoded claim; skipping",
			"tx", tx.Hash(), "err", err)
		m.metrics.RecordRejectedClaim("decode")
		return nil, false
	}
	return c, true
}

// succeeded maps a block's transaction hashes to whether they succeeded.
func (m *Module) succeeded(ctx context.Context, src Rendering, block eth.BlockID) (map[common.Hash]bool, error) {
	_, receipts, err := src.FetchReceipts(ctx, block)
	if err != nil {
		return nil, err
	}
	out := make(map[common.Hash]bool, len(receipts))
	for _, r := range receipts {
		if r == nil {
			continue
		}
		out[r.TxHash] = r.Status == types.ReceiptStatusSuccessful
	}
	return out, nil
}

func (m *Module) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := m.Step(ctx); err != nil {
			m.log.Warn("Claim follow poll deferred", "err", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
