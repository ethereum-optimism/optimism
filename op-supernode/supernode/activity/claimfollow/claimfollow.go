// Package claimfollow translates accepted projection claims into private-chain
// safety checkpoints. It reads only public projection data; private hashes come
// from accepted terminal commitments, while block scheduling comes from the
// corresponding projection blocks.
//
// The claimed RPC route serves checkpoints and a fully scanned recovery frontier.
// A private LightCL executes deposit-only replacement inputs against its own state
// using the ordinary attributes handler. Projection hashes never become private
// forkchoice hashes. Reorgs revoke affected checkpoints; only finality is monotonic.
// Surviving partial ranges are authenticated through their original private
// terminal commitment and the projection's persisted replacement history.
//
// Before the first claim, the configured private genesis and projection L1 view
// let the sequencer and batcher bootstrap without waiting on their own first batch.
package claimfollow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	// ErrNoGenesisRef is returned by SyncStatus when the module was built without a genesis hash and
	// no claim has been read yet — the one state in which it has nothing true to say. An operator
	// cannot reach it: startup requires the private-chain genesis whenever rollup config enables the module,
	// precisely so that this is a startup failure rather than a bootstrap that silently hangs.
	//
	// It is kept, rather than replaced by a panic, because the follow consumer treats an erroring
	// source as a skipped tick and a library caller who assembled a Config by hand deserves the same
	// harmless outcome the rest of this package gives every other failure.
	ErrNoGenesisRef = errors.New("claim follow module has no genesis ref and has not read a claim yet")

	// ErrInvariant is returned instead of a status that would violate the consumer's ordering
	// rules or contradicts retained finalized history.
	ErrInvariant = errors.New("claim follow module computed a status violating finalized <= safe <= local_safe")
)

// DefaultMaxBlocksPerPoll bounds one poll's scan so that catching up is incremental rather than one
// unbounded stall.
const DefaultMaxBlocksPerPoll = 512

// Rendering is the module's view of the chain container driving the rendering chain.
//
// Every method is an accessor the container already exposes, and each is read IN PROCESS: the
// module never dials an RPC endpoint of its own, not even a loopback one. PayloadByNumber is the
// load-bearing choice — one call yields the block's identity, its timestamp, its transactions AND
// (through its L1-attributes deposit) its L1 origin and sequence number, which is every input the
// scan and the ref completion need except receipts.
//
// The module is served from a route mounted on that same chain's RPC handler, so it sits behind the
// chain's readiness gate. That is the correct semantics rather than an accepted cost: a module
// whose only inputs are one chain's derived data has nothing new to say while that chain is
// restarting or paused, and the router's hold-then-503 reaches the consumer as a skipped tick.
//
// chain_container.ChainContainer satisfies this.
type Rendering interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)
	FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error)
}

// Config is the module's static configuration.
type Config struct {
	// Registry is the ClaimRegistry's address on the rendering chain. A transaction to any other
	// address is not a claim and is not looked at.
	Registry common.Address
	// GenesisHash is the PRIVATE chain's genesis block hash: the one field of the private genesis
	// ref that no public data carries, and therefore the only thing this module has to be told about
	// the chain it follows.
	//
	// It is what the not-yet state is served from, and it is required whenever the module is
	// enabled. See the package comment: without it the module says nothing until the first claim,
	// and the operator's own batcher — which shares the op-node this feeds — will not load a block
	// until it is told a non-zero current_l1, so no claim ever comes.
	GenesisHash common.Hash
	// StartBlock is the first rendering block to scan. Zero scans from genesis.
	StartBlock uint64
	// MaxBlocksPerPoll bounds one poll's scan. Zero means DefaultMaxBlocksPerPoll.
	MaxBlocksPerPoll uint64
}

// claim is one registry-accepted claim, plus the completion state that turns it into a ref.
//
// carrier is the RENDERING block the claim transaction was included in. It is NOT the block the ref
// comes from: a claim LEADS the range it describes, so it rides in the range's first block while
// the ref it resolves to lives at the range's LAST block, a whole cadence later. That gap is why
// completion is a separate step from reading, and why finality gates on `last` rather than on
// `carrier` (see promote). What carrier is for is the rewind: a reorg that reaches the block a
// claim arrived in is what un-reads that claim.
type claim struct {
	carrier         uint64
	invalidFrom     uint64
	revokedTip      eth.BlockID // previously observed suffix, retained across temporary revocation
	replacementFrom uint64      // persisted denial confirms a replacement
	prefixRef       eth.L2BlockRef
	first           uint64
	last            uint64
	terminal        common.Hash
	parent          common.Hash
	ref             eth.L2BlockRef
	completed       bool
}

// Module is the follow module's state machine. Step drives it; SyncStatus reads it.
//
// It is safe for concurrent use: Step runs on the poll loop and SyncStatus runs on RPC handlers.
type Module struct {
	cfg       Config
	rollupCfg *rollup.Config
	log       gethlog.Logger
	metrics   Metrics
	rendering Rendering

	mu sync.RWMutex

	// Local and cross-safe claims are tracked separately. haveSafe indicates
	// that the initial private genesis checkpoint is available.
	safe      eth.L2BlockRef
	localSafe eth.L2BlockRef
	haveSafe  bool
	// finalized is the highest completed claim whose TERMINAL rendering block is finalized — see
	// promote for why the terminal block and not the carrier.
	finalized eth.L2BlockRef
	// currentL1 is the rendering's own current L1, forwarded with canonical reorgs.
	currentL1 eth.L1BlockRef
	// recoveryTarget is a fully scanned local-safe projection frontier. Blocks
	// after the last completed claim still require private deposit execution.
	recoveryTarget    eth.L2BlockRef
	recoverySafe      eth.L2BlockRef
	recoveryFinalized eth.L2BlockRef

	// next is the next rendering block number to scan. anchored and lastHash anchor the reorg check
	// on block next-1, including the surviving ancestor after a rewind.
	next         uint64
	lastHash     common.Hash
	anchored     bool
	history      map[uint64]eth.L2BlockRef
	generation   uint64
	invariantErr error

	// pending holds claims that are not yet BOTH completed and finalized, in ascending carrier
	// order. A claim leaves it once its ref has been served as finalized.
	pending []*claim
}

// New builds a module. It performs no I/O, and it does NOT take its data source.
//
// The two are built in this order out of necessity, not taste: the chain container that feeds the
// module has to be constructed with this module's API among its routes, and the module has to be
// constructed to have an API. Attach closes the loop, and an unattached module answers every step
// with an error — which is a skipped tick, the same as any other not-yet state here.
func New(cfg Config, rollupCfg *rollup.Config, lgr gethlog.Logger, m Metrics) *Module {
	if cfg.MaxBlocksPerPoll == 0 {
		cfg.MaxBlocksPerPoll = DefaultMaxBlocksPerPoll
	}
	if m == nil {
		m = NoopMetrics{}
	}
	mod := &Module{
		cfg:       cfg,
		rollupCfg: rollupCfg,
		log:       lgr,
		metrics:   m,
		next:      cfg.StartBlock,
		history:   make(map[uint64]eth.L2BlockRef),
	}
	// The not-yet state, seeded at construction so the module answers from its very first tick.
	// Serving from t=0 is the point: everything downstream of the op-node this feeds — the
	// operator's batcher above all — inherits this module's silence, and the batcher will not load a
	// block until it is told a current_l1. See the package comment on the bootstrap deadlock.
	if cfg.GenesisHash != (common.Hash{}) {
		ref := genesisRef(cfg.GenesisHash, rollupCfg)
		mod.safe, mod.localSafe, mod.finalized, mod.haveSafe = ref, ref, ref, true
		lgr.Info("Claim follow module will serve the private chain's genesis ref until the first claim", "genesis", ref)
	}
	return mod
}

// genesisRef is the PRIVATE chain's genesis L2BlockRef, assembled from one configured hash and five
// facts the module already holds.
//
// Only the hash is unknowable from public data. The rest are definitions or consequences of the
// pair's block-for-block construction, and each is the same value derive.PayloadToBlockRef would
// produce for a genesis payload, which is what makes the served ref compare equal to the one the
// private EL derives for itself:
//
//   - number and timestamp come from the rendering's rollup config. The two chains are
//     block-for-block — one public block per private block, at the same height and the same time —
//     and both halves' deployments pin the same l1StartBlockHash, which is what makes their genesis
//     timestamps equal rather than merely similar (DESIGN.md, "Hardfork adoption on the rendering").
//   - l1origin is the rollup config's genesis L1, for the same reason, and it is REAL, which
//     matters: the driver hash-checks the L1 origin of all three served refs against L1
//     (op-node/rollup/driver/driver.go), so a synthesised or zero origin would take the whole status
//     down. This one passes because it is the chain's actual genesis anchor.
//   - parentHash is zero and sequenceNumber is zero by the definition of a genesis block; the
//     latter is exactly what PayloadToBlockRef hardcodes on its genesis branch.
func genesisRef(hash common.Hash, rollupCfg *rollup.Config) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           hash,
		ParentHash:     common.Hash{},
		Number:         rollupCfg.Genesis.L2.Number,
		Time:           rollupCfg.Genesis.L2Time,
		L1Origin:       rollupCfg.Genesis.L1,
		SequenceNumber: 0,
	}
}

// Attach binds the rendering chain the module reads. Call it once, before the poll loop starts.
func (m *Module) Attach(r Rendering) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rendering = r
}

// source returns the attached rendering chain, or an error if nothing has been attached.
func (m *Module) source() (Rendering, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.rendering == nil {
		return nil, errors.New("claim follow module has no rendering chain attached")
	}
	return m.rendering, nil
}

// SyncStatus returns private local-safe, cross-safe and finalized checkpoints,
// plus the projection's current L1 view. Before the first claim, all private
// checkpoints are genesis, allowing the sequencer and batcher to bootstrap.
func (m *Module) SyncStatus() (*eth.SyncStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.syncStatusLocked()
}

func (m *Module) syncStatusLocked() (*eth.SyncStatus, error) {
	if m.invariantErr != nil {
		return nil, m.invariantErr
	}
	if !m.haveSafe {
		return nil, ErrNoGenesisRef
	}
	out := &eth.SyncStatus{
		LocalSafeL2: m.localSafe,
		SafeL2:      m.safe,
		FinalizedL2: m.finalized,
		CurrentL1:   m.currentL1,
	}
	// The consumer rejects a status that breaks either ordering rule, and so do we, one step
	// earlier: serving a status we already know is invalid would only make the consumer decide it
	// is invalid, with less context to say why.
	if out.SafeL2.Number > out.LocalSafeL2.Number || out.FinalizedL2.Number > out.SafeL2.Number {
		return nil, fmt.Errorf("%w: finalized %d, safe %d, local safe %d",
			ErrInvariant, out.FinalizedL2.Number, out.SafeL2.Number, out.LocalSafeL2.Number)
	}
	return out, nil
}

// Step performs one poll: re-anchor the cursor, scan new rendering blocks for claims, complete the
// refs of claims whose terminal block the rendering has now derived, and promote whatever the
// rendering has finalized.
//
// It returns an error for the caller to log; the poll loop never exits on one. A returned error
// leaves unprocessed blocks available for a later retry.
func (m *Module) Step(ctx context.Context) error {
	src, err := m.source()
	if err != nil {
		return err
	}
	status, err := src.SyncStatus(ctx)
	if err != nil {
		return fmt.Errorf("reading the rendering chain's sync status: %w", err)
	}
	if status == nil {
		return errors.New("the rendering chain returned an empty sync status")
	}
	m.observeCurrentL1(status.CurrentL1)
	renderSafe, renderFinalized := status.SafeL2.Number, status.FinalizedL2.Number
	local := status.LocalSafeL2
	if local == (eth.L2BlockRef{}) {
		local = status.SafeL2
	}
	renderLocal := local.Number
	if err := m.reanchor(ctx, src, renderLocal, renderFinalized); err != nil {
		return err
	}
	m.mu.RLock()
	generation := m.generation
	m.mu.RUnlock()
	if err := m.scan(ctx, src, renderLocal); err != nil {
		return err
	}
	// A status read can race an engine reset before our generation was sampled.
	// Bind completion and safety promotion to the identities actually scanned.
	m.mu.RLock()
	scannedThrough := m.next
	scannedLocal := m.history[renderLocal]
	m.mu.RUnlock()
	if scannedThrough <= renderLocal {
		return nil
	}
	if scannedLocal != local {
		return fmt.Errorf("projection frontier changed while scanning claims")
	}
	for _, expected := range []eth.L2BlockRef{status.SafeL2, status.FinalizedL2} {
		env, err := src.PayloadByNumber(ctx, expected.Number)
		if err != nil {
			return err
		}
		if env == nil || env.ExecutionPayload == nil {
			return fmt.Errorf("projection safety block is unavailable")
		}
		actual, err := derive.PayloadToBlockRef(m.rollupCfg, env.ExecutionPayload)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("projection safety snapshot changed while scanning claims")
		}
	}
	if err := m.complete(ctx, src, renderLocal); err != nil {
		return err
	}
	m.mu.Lock()
	if generation != m.generation {
		m.mu.Unlock()
		return fmt.Errorf("claim history changed during scan")
	}
	m.promoteLocked(renderSafe, renderFinalized)
	if m.next > renderLocal {
		m.recoveryTarget = local
		m.recoverySafe = status.SafeL2
		m.recoveryFinalized = status.FinalizedL2
	}
	for n := range m.history {
		if n < min(renderFinalized, m.next-1) {
			delete(m.history, n)
		}
	}
	m.mu.Unlock()
	return nil
}

// observeCurrentL1 records the rendering's current L1 for the served status.
//
// The consumer tolerates a zero here and never uses it for origin selection, so this is a courtesy
// rather than a requirement — but it is the honest answer to "which L1 view is this chain's safety
// anchored to", because a private range becomes safe exactly when its claim lands in an L1 batch
// the rendering derived, and it is what keeps a downstream reader (op-batcher rejects a zero
// CurrentL1 outright) from being told silence. A zero from the chain is held rather than forwarded,
// and canonical regressions are forwarded: the value is a view, not a commitment.
func (m *Module) observeCurrentL1(current eth.L1BlockRef) {
	if current == (eth.L1BlockRef{}) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentL1 = current
}

// reanchor finds the last common projection ancestor within retained history.
func (m *Module) reanchor(ctx context.Context, src Rendering, renderSafe, _ uint64) error {
	m.mu.RLock()
	anchored, next := m.anchored, m.next
	m.mu.RUnlock()
	if !anchored {
		return nil
	}
	for n := min(next-1, renderSafe); ; n-- {
		m.mu.RLock()
		old, known := m.history[n]
		m.mu.RUnlock()
		if !known {
			return fmt.Errorf("%w: projection reorg passed retained finalized history", ErrInvariant)
		}
		env, err := src.PayloadByNumber(ctx, n)
		if err != nil {
			return err
		}
		if env == nil || env.ExecutionPayload == nil {
			return fmt.Errorf("missing projection block %d", n)
		}
		if env.ExecutionPayload.BlockHash == old.Hash {
			if n != next-1 {
				m.metrics.RecordRenderingReorg()
				m.rewind(n)
			}
			return nil
		}
		if n == 0 {
			return fmt.Errorf("%w: projection genesis changed", ErrInvariant)
		}
	}
}

// rewind revokes the affected suffix while retaining finalized checkpoints and
// any surviving claim carrier needed to authenticate a partial private prefix.
func (m *Module) rewind(h uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	// A reset notification may arrive before the poller has read the affected
	// range. It must never advance the scan past an unread claim carrier.
	if m.next == 0 {
		m.recoveryTarget = eth.L2BlockRef{}
		return
	}
	oldTip := m.next - 1
	h = min(h, oldTip)
	if h < m.finalized.Number {
		m.invariantErr = fmt.Errorf("%w: rewind below private finality", ErrInvariant)
		return
	}
	m.localSafe, m.safe = m.finalized, m.finalized
	next := h + 1
	if next < m.cfg.StartBlock {
		next = m.cfg.StartBlock
	}
	m.next = next
	ref, known := m.history[h]
	m.anchored = known
	m.lastHash = ref.Hash
	m.recoveryTarget = eth.L2BlockRef{}
	kept := m.pending[:0]
	for _, c := range m.pending {
		if c.carrier <= h {
			if c.last > h {
				if c.revokedTip == (eth.BlockID{}) {
					if c.replacementFrom != 0 {
						c.revokedTip = c.prefixRef.ID()
					} else {
						c.revokedTip = m.history[min(c.last, oldTip)].ID()
					}
				}
				c.completed = false
				if c.invalidFrom == 0 || h+1 < c.invalidFrom {
					c.invalidFrom, c.prefixRef = h+1, m.history[h]
				}
			} else if c.completed {
				if c.ref.Number > m.localSafe.Number {
					m.localSafe = c.ref
				}
			}
			kept = append(kept, c)
		}
	}
	m.pending = kept
	for n := range m.history {
		if n > h {
			delete(m.history, n)
		}
	}
}

// scan walks new rendering blocks up to the chain's safe head, recording every claim it finds.
//
// The cursor advances only after a block is FULLY processed, so a transient read failure leaves
// that block to be re-scanned next poll rather than skipped.
func (m *Module) scan(ctx context.Context, src Rendering, renderSafe uint64) error {
	m.mu.RLock()
	next, budget, generation := m.next, m.cfg.MaxBlocksPerPoll, m.generation
	m.mu.RUnlock()

	for n := next; n <= renderSafe && budget > 0; n, budget = n+1, budget-1 {
		env, err := src.PayloadByNumber(ctx, n)
		if err != nil {
			return fmt.Errorf("reading rendering block %d: %w", n, err)
		}
		if env == nil || env.ExecutionPayload == nil {
			return fmt.Errorf("rendering block %d came back without a payload", n)
		}
		payload := env.ExecutionPayload
		m.mu.RLock()
		anchored, lastHash := m.anchored, m.lastHash
		m.mu.RUnlock()
		if anchored && payload.ParentHash != lastHash {
			// The chain moved between the anchor check and this read. Stop; the next poll's
			// reanchor sees it and rewinds. Advancing here would build the cursor on a fork.
			m.log.Warn("Rendering block does not extend the claim follow module's cursor; deferring to the next poll",
				"block", n, "parent", payload.ParentHash, "cursor", lastHash)
			m.metrics.RecordRenderingReorg()
			return nil
		}
		if err := m.applyBlock(ctx, src, n, payload, generation); err != nil {
			return err
		}
		if err := m.checkReplacement(ctx, src, payload, generation); err != nil {
			return err
		}
		m.mu.Lock()
		if generation != m.generation {
			m.mu.Unlock()
			return fmt.Errorf("claim history changed during scan")
		}
		ref, err := derive.PayloadToBlockRef(m.rollupCfg, payload)
		if err != nil {
			m.mu.Unlock()
			return err
		}
		m.history[n] = ref
		for _, c := range m.pending {
			if c.invalidFrom != 0 && c.revokedTip == ref.ID() {
				// Restore temporary revocation, preserving a confirmed denial.
				c.invalidFrom, c.prefixRef, c.revokedTip = c.replacementFrom, eth.L2BlockRef{}, eth.BlockID{}
				if c.replacementFrom != 0 {
					c.prefixRef = m.history[c.replacementFrom-1]
				}
			}
		}
		m.next, m.anchored, m.lastHash = n+1, true, payload.BlockHash
		m.mu.Unlock()
	}
	return nil
}

// applyBlock records one rendering block's claims.
func (m *Module) applyBlock(ctx context.Context, src Rendering, num uint64, payload *eth.ExecutionPayload, generation uint64) error {
	type candidate struct {
		hash  common.Hash
		claim *codec.RangeClaim
	}
	var candidates []candidate
	for i, opaque := range payload.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(opaque); err != nil {
			// A block the chain itself derived cannot carry an undecodable transaction; if one
			// somehow does, it is not a claim and the rest of the block still is.
			m.log.Error("Skipping an undecodable transaction in a rendering block",
				"block", num, "index", i, "err", err)
			continue
		}
		to := tx.To()
		if to == nil || *to != m.cfg.Registry {
			continue
		}
		c, ok := m.decodeClaim(&tx)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{hash: tx.Hash(), claim: c})
	}
	if len(candidates) == 0 {
		return nil
	}
	// One receipts fetch, only for the rare block that opens a range.
	succeeded, err := m.succeeded(ctx, src, eth.BlockID{Hash: payload.BlockHash, Number: num})
	if err != nil {
		return fmt.Errorf("reading receipts for rendering block %d: %w", num, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if generation != m.generation {
		return fmt.Errorf("claim history changed while reading receipts")
	}
	for _, c := range candidates {
		if !succeeded[c.hash] {
			// A REVERTED postClaim never entered the registry's record, so there is no claim here
			// to follow. Under snap-to-commitment this is a skip and a metric, not a latch: the
			// sidecar's fail-stop was about keeping the RECOVERY record whole, which is not a
			// serving concern. See the package comment.
			m.log.Error("A postClaim transaction REVERTED; the range never entered the registry's record, skipping",
				"block", num, "tx", c.hash, "range", fmt.Sprintf("%d-%d", c.claim.FirstBlock, c.claim.LastBlock))
			m.metrics.RecordRejectedClaim("reverted")
			continue
		}
		m.pending = append(m.pending, &claim{
			carrier:  num,
			first:    c.claim.FirstBlock,
			last:     c.claim.LastBlock,
			terminal: c.claim.PrivateTerminalBlockHash,
			parent:   c.claim.PrivateTerminalParentHash,
		})
		m.metrics.RecordClaim()
		m.log.Info("Claim read from the rendering chain",
			"renderingBlock", num, "range", fmt.Sprintf("%d-%d", c.claim.FirstBlock, c.claim.LastBlock),
			"terminal", c.claim.PrivateTerminalBlockHash)
	}
	return nil
}

// decodeClaim recognises a claim in a registry-addressed transaction's calldata.
//
// The calldata is a selector followed by exactly what codec.Encode produces, and NOTHING re-encodes
// it on the way in — so canonical form is a property to CHECK, not to assume. codec.Decode is
// therefore the strict decoder, and a strict-decode failure on a registry-addressed transaction is
// a LOUD LOG AND A SKIP of that transaction, never a crash and never a lenient second attempt:
// anyone may send bytes to a contract address, and a module that fell over when they did would be a
// denial-of-service switch for the price of one transaction.
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

// complete turns claims into refs, and it is the whole of the "ref completion from public data"
// step. It runs after the scan because a claim LEADS its range: the rendering block the ref is read
// from is a full cadence ABOVE the block that carried the claim, so completion has to wait for the
// chain to derive it.
func (m *Module) complete(ctx context.Context, src Rendering, renderSafe uint64) error {
	for {
		m.mu.RLock()
		var next *claim
		for _, c := range m.pending {
			if !c.completed && c.invalidFrom == 0 && c.last <= renderSafe && c.last < m.next {
				next = c
				break
			}
		}
		generation := m.generation
		m.mu.RUnlock()
		if next == nil {
			return nil
		}
		ref, err := m.completeRef(ctx, src, next)
		if err != nil {
			// The chain says it has derived this height, so a failure here is transient. Leave the
			// claim pending and try again next poll rather than advancing past it.
			m.log.Warn("Cannot yet complete a claimed range's ref; retrying next poll",
				"renderingBlock", next.carrier, "lastBlock", next.last, "err", err)
			return err
		}
		m.mu.Lock()
		if generation != m.generation || next.invalidFrom != 0 {
			m.mu.Unlock()
			return fmt.Errorf("claim history changed during completion")
		}
		next.ref, next.completed = ref, true
		if !m.haveSafe || ref.Number > m.localSafe.Number {
			m.localSafe, m.haveSafe = ref, true
			m.metrics.RecordSafe(ref.Number)
			m.log.Info("Claim completed; advancing the private chain's safe head",
				"renderingBlock", next.carrier, "range", fmt.Sprintf("%d-%d", next.first, next.last), "safe", ref)
		}
		m.mu.Unlock()
	}
}

// completeRef assembles the private chain's L2BlockRef at a claim's lastBlock out of public data.
//
// Two fields come from the CLAIM and four from the supernode's OWN rendering block at the same
// height. The four are legitimate because of the ratified ORIGIN-COPY amendment: the batcher's
// transformation reuses each private block's own L1 origin as the rendering block's epoch, so the
// two chains' origins — and therefore their sequence numbers — are EQUAL BY CONSTRUCTION rather
// than merely expected to agree. Timestamps are equal for the same reason the numbers are: the
// rendering is block-for-block, one public block per private block at the same height and time.
//
// The height is read at or below the chain's SAFE view, never from its unsafe head, so the four
// borrowed fields are exactly as durable as the claim itself.
func (m *Module) completeRef(ctx context.Context, src Rendering, c *claim) (eth.L2BlockRef, error) {
	env, err := src.PayloadByNumber(ctx, c.last)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("reading rendering block %d: %w", c.last, err)
	}
	if env == nil || env.ExecutionPayload == nil {
		return eth.L2BlockRef{}, fmt.Errorf("rendering block %d came back without a payload", c.last)
	}
	m.mu.RLock()
	ref, scanned := m.history[c.last]
	m.mu.RUnlock()
	if !scanned || ref.Hash != env.ExecutionPayload.BlockHash {
		return eth.L2BlockRef{}, fmt.Errorf("claim terminal changed since its range was scanned")
	}
	renderRef, err := derive.PayloadToBlockRef(m.rollupCfg, env.ExecutionPayload)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("deriving the rendering ref at block %d: %w", c.last, err)
	}
	return eth.L2BlockRef{
		Hash:           c.terminal,
		ParentHash:     c.parent,
		Number:         c.last,
		Time:           renderRef.Time,
		L1Origin:       renderRef.L1Origin,
		SequenceNumber: renderRef.SequenceNumber,
	}, nil
}

// promote moves completed claims into the finalized label. The rendering's finality is L1-batch
// finality for the claim inside it, which is the whole of what "this private range is final" means
// on a chain with no derivation of its own.
//
// The gate is the claim's TERMINAL block, not the block that carried the claim, and the difference
// is load-bearing rather than conservative housekeeping. A served finalized ref is SIX fields, and
// four of them are borrowed from the rendering block at lastBlock — which sits a whole cadence
// ABOVE the carrier. Gating on the carrier would publish a finalized ref two of whose fields came
// from a finalized block and four from a block that was only safe, so an L1 reorg above the
// finalized height could change the borrowed four while the claim-borne hash stayed put. That is a
// changed finalized ref at an unchanged height, and the consumer's cache treats a finalized head as
// immutable at a height (engine_controller.applyFinalizedHeadCacheChecks: same ID returns the stale
// cached value, same height with a different hash PANICS). A finalized output must therefore be a
// pure function of finalized-depth inputs, and requiring lastBlock <= renderFinalized is what makes
// every one of its six fields that.
//
// It costs nothing in ordering: a claim leads its own range, so carrier == firstBlock <= lastBlock,
// and this gate strictly implies the carrier's own finality.
func (m *Module) promoteLocked(renderSafe, renderFinalized uint64) {
	// Cross-safety can retreat while the same locally derived chain remains.
	m.safe = m.finalized
	kept := m.pending[:0]
	for _, c := range m.pending {
		if c.completed && c.last <= renderSafe && c.ref.Number > m.safe.Number {
			m.safe = c.ref
		}
		if !c.completed || c.last > renderFinalized {
			if c.invalidFrom != 0 && c.last <= m.finalized.Number {
				continue
			}
			kept = append(kept, c)
			continue
		}
		if c.ref.Number > m.finalized.Number {
			m.finalized = c.ref
			m.metrics.RecordFinalized(c.ref.Number)
			m.log.Info("Claim finalized with the rendering block that carried it",
				"renderingBlock", c.carrier, "finalized", c.ref)
		}
	}
	m.pending = kept
}

// Run polls until the context is cancelled. A step that fails is logged and retried on the next
// tick: for a follow source, "nothing new this tick" is a normal state the consumer already
// tolerates, so there is no failure here worth taking a supernode down for.
func (m *Module) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := m.Step(ctx); err != nil {
			m.log.Warn("Claim follow module poll did not advance", "err", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
