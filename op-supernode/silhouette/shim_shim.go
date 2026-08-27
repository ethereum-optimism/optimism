package silhouette

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// THE SHIM EXECUTION CLIENT: a service that speaks the Engine API and the eth_ query surface for a
// proof-carried chain, executes NOTHING, and serves proof-committed facts.
//
// It lives in this package rather than a subpackage of it, and the reason is worth one line because
// it looks like a layering mistake: the executable spec of the forced-extension convention and the
// whole synthetic-L1 + proof-batch harness are this package's own test files, and a Go file in
// package silhouette may not import a package that imports silhouette — that is an import cycle in
// the test binary, not a warning. A subpackage could not be driven by the harness the convention was
// written against, so the shim would have been tested against a second copy of it. See G3 D1.
//
// THE CLAIM (PLAN.md DR-1, ratified 2026-08-24). A proven chain is a chain whose execution client is
// a verifier. op-node treats the engine as the hash authority — `EngineClientDefaultConfig` builds
// its L2 client with `trustCache=true` ("engine is trusted, no need to recompute responses",
// op-service/sources/engine_client.go:21-26) and the block-hash recompute in
// op-service/sources/types.go:175-197 is gated on `!trustCache` — and the build dance
// fcU(attrs) → getPayload → newPayload → fcU never re-hashes a payload. So an execution client that
// serves the wire's real blockHash, stateRoot and messagePasserStorageRoot makes stock op-node code
// operate on the private chain's REAL identity: post-Isthmus `outputV0` reads the message-passer root
// straight out of the header's withdrawalsRoot (op-service/sources/l2_client.go:192-227), so
// `OutputV0AtBlockNumber` over these served headers returns P's real output roots, byte-identical to
// settlement claims, through code with no idea any of this is happening.
//
// THE HONEST PRICE (declared, not smoothed). This service speaks the execution vocabulary about
// things that never happened. "VALID" means continuity plus acceptance, not execution. The receipts
// root is the empty-receipts constant for a block that has a transaction. gasUsed is zero. The
// headers deliberately do not re-hash to the hashes served alongside them. Three things keep that
// from being a lie told to a machine that would believe it:
//
//   - SELF-DECLARATION AT THE SERVICE LAYER. The `silhouette_` namespace and web3_clientVersion say
//     what this is. The in-header marker PLAN.md originally asked for is dead (G2 D8): extraData is
//     the consensus-legal carrier of the eip-1559 parameters, and a marker there would have silently
//     reset the chain's fee market on the following block.
//   - FAIL-STOP GUARDS. getPayload refuses any block with no proven-or-forced fact; newPayload
//     refuses any hash that is not the fact hash at that height. Derivation can never outrun the
//     proof stream, and no path fabricates.
//   - THE HONESTY ASSERTION, KEPT IN CODE. P is proof-trusted: no judge invalidates it, so nothing
//     may reach the replacement-block synthesizer that mutates ExtraData and re-hashes
//     (chain_container/engine_controller/rewind.go:195 on the Cove branch). The shim's rejection of
//     unknown hashes is what converts a regression there into a loud halt instead of a quiet fork.
//
// WHAT THIS PACKAGE IS NOT. It holds no state, no EVM, no genesis allocation, and no transaction
// pool. It never returns SYNCING and never EL-syncs (`--syncmode=consensus-layer`). It does not feed
// LogsDB: rendering-device receipts are display-only and eth_getBlockReceipts is refused outright,
// because positional receipts ingestion would require filler logs with public preimages, i.e.
// forgeable initiating messages (PLAN.md's LogsDB rule).

// ClientVersion is the self-declaration a caller sees over web3_clientVersion. It names what this
// service is in the one place an operator or an integrator actually looks.
const ClientVersion = "op-silhouette-el/v1 (proof-rendered blocks; executes nothing)"

// Shim is the execution client for one silhouette chain.
type Shim struct {
	log    log.Logger
	params ForcedParams
	facts  *FactStore
	l1     L1Headers

	// onHalt, when set, is called once with the halt reason. G5 wires it to bring the node down: a
	// halted shim must not look like a slow one.
	onHalt func(error)

	mu   sync.Mutex
	jobs map[eth.PayloadID]*buildJob
	// replacements are real, deposits-only payloads prepared by P's private EL after the stock
	// supernode invalidates a proof fact. They live in FactStore so they survive an EL restart, while
	// remaining outside its canonical proof-fact slice until a corrected proof is posted.
	replacementBuilder ReplacementBuilder
	halted             error
}

// buildJob is an open block-building job: a parent, and the attributes the CL asked for.
//
// A job holds no partial block, because there is nothing to build. It is a promise to look one fact
// up when the CL comes back for it.
type buildJob struct {
	parent Fact
	attrs  *eth.PayloadAttributes
}

// NewShim builds a shim over a fact store.
//
// rollupCfg and sysCfg are P's rollup config and its FROZEN genesis SystemConfig (DR-2); l1Chain is
// the settlement chain's config, read only for the L1-info transaction's blob-fee field. l1 is the
// headers-only L1 access the forced-extension convention needs — nothing here reads a receipt.
func NewShim(logger log.Logger, rollupCfg *rollup.Config, l1Chain *params.ChainConfig, sysCfg eth.SystemConfig,
	l1 L1Headers, facts *FactStore,
) *Shim {
	return &Shim{
		log:    logger,
		params: ForcedParams{Rollup: rollupCfg, L1Chain: l1Chain, SysCfg: sysCfg},
		facts:  facts,
		l1:     l1,
		jobs:   make(map[eth.PayloadID]*buildJob),
	}
}

// SetReplacementBuilder connects the Silhouette EL to P's private Engine API for the single
// operation it cannot perform itself: executing stock deposits-only replacement attributes.
func (s *Shim) SetReplacementBuilder(builder ReplacementBuilder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replacementBuilder = builder
}

// OnHalt registers the callback invoked when the shim halts.
func (s *Shim) OnHalt(fn func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onHalt = fn
}

// ErrHalted is what every method returns after a halt.
var ErrHalted = errors.New("silhouette shim halted")

// halt is the loud stop. It is reached from exactly one place — a newPayload whose hash contradicts
// the facts — and it is permanent by design: the alternative to stopping is describing a block that
// no proof covers, which is the one thing this service exists not to do.
func (s *Shim) halt(reason error) {
	s.mu.Lock()
	first := s.halted == nil
	if first {
		s.halted = reason
	}
	fn, halted := s.onHalt, s.halted
	s.mu.Unlock()
	if !first {
		return
	}
	// Deliberately Error and not Crit: geth's Crit calls os.Exit(1) (op-geth log/logger.go:271-273),
	// and killing the process is the supernode's decision to make through OnHalt, not this method's.
	// A halted shim refuses every request, which is the part that must not be negotiable.
	s.log.Error("SILHOUETTE SHIM HALTED — the engine was asked to accept a block outside the proven "+
		"facts. This is the honesty assertion of DR-1 firing: no path may reach invalidation or "+
		"replacement-block synthesis for a proof-trusted chain. Refusing every further request.",
		"reason", halted)
	if fn != nil {
		fn(halted)
	}
}

// Halted reports the halt reason, if the shim has halted.
func (s *Shim) Halted() (error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.halted, s.halted != nil
}

func (s *Shim) checkLive() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.halted != nil {
		return fmt.Errorf("%w: %w", ErrHalted, s.halted)
	}
	return nil
}

// genesisFact is the rollup genesis block as a fact-shaped row.
//
// Genesis is the one block in the shim's universe that is neither proven nor forced: it is
// configuration. Its hash, number and timestamp are real (they are what the chain is defined by) and
// its roots are NOT KNOWN to this node — there is no wire record of them and no state to read them
// from. They are served as zero, which fails closed rather than quietly: an output root computed from
// a genesis header is keccak(0‖0‖0‖hash), a value that matches no real output root and no settlement
// claim, so a consumer that wrongly reached for one gets a mismatch rather than a plausible answer.
// See G3 D6.
func (s *Shim) genesisFact() Fact {
	g := s.params.Rollup.Genesis
	return Fact{
		Number:    g.L2.Number,
		Timestamp: s.params.Rollup.Genesis.L2Time,
		Hash:      g.L2.Hash,
		L1Origin:  g.L1,
		SeqNumber: 0,
	}
}

func (s *Shim) isGenesis(number uint64) bool {
	return number == s.params.Rollup.Genesis.L2.Number
}

// factByNumber resolves a block number to its facts, genesis included.
func (s *Shim) factByNumber(number uint64) (Fact, bool) {
	if s.isGenesis(number) {
		return s.genesisFact(), true
	}
	if fact, ok := s.facts.ByNumber(number); ok {
		return fact, true
	}
	return s.facts.ReplacementByNumber(number)
}

// factByHash resolves a block hash to its facts, genesis included.
func (s *Shim) factByHash(hash common.Hash) (Fact, bool) {
	if hash == s.params.Rollup.Genesis.L2.Hash {
		return s.genesisFact(), true
	}
	if fact, ok := s.facts.ByHash(hash); ok {
		return fact, true
	}
	if replacement, ok := s.facts.ReplacementByHash(hash); ok {
		return replacement, true
	}
	if rewind, ok := s.facts.RewindFact(hash); ok {
		return rewind, true
	}
	return Fact{}, false
}

func (s *Shim) isRewindFact(hash common.Hash) bool {
	return s.facts.IsRewindFact(hash)
}

func (s *Shim) recordRewindFact(fact Fact) {
	s.facts.RecordRewindFact(fact)
}

func (s *Shim) clearRewindFacts() {
	s.facts.ClearRewindFacts()
}

// parentOf returns a block's parent facts. A block whose parent has fallen out of the fact window is
// refused rather than served with a fabricated parent hash: "not here any more" and "not proven" are
// different statements and the shim must not conflate them.
func (s *Shim) parentOf(fact Fact) (Fact, error) {
	if s.isGenesis(fact.Number) {
		return Fact{}, fmt.Errorf("genesis has no parent")
	}
	parent, ok := s.factByNumber(fact.Number - 1)
	if !ok {
		return Fact{}, fmt.Errorf("block %d is in this node's fact window but its parent %d "+
			"is not, so its parent hash is unknown; proven history below the window is re-derived from "+
			"L1 rather than remembered", fact.Number, fact.Number-1)
	}
	return parent, nil
}

// ref turns facts into the L2BlockRef the CL's own parsers would produce from the served block.
func (s *Shim) ref(fact Fact) (eth.L2BlockRef, error) {
	var parentHash common.Hash
	if !s.isGenesis(fact.Number) {
		parent, err := s.parentOf(fact)
		if err != nil {
			return eth.L2BlockRef{}, err
		}
		parentHash = parent.Hash
	}
	return eth.L2BlockRef{
		Hash:           fact.Hash,
		Number:         fact.Number,
		ParentHash:     parentHash,
		Time:           fact.Timestamp,
		L1Origin:       fact.L1Origin,
		SequenceNumber: fact.SeqNumber,
	}, nil
}

// head is the block the head cursor points at, defaulting to genesis.
//
// A brand-new shim genuinely IS at genesis. A persistent standalone EL restores its cursor before
// constructing the shim, while a zero-value test store has no forkchoice opinion yet.
func (s *Shim) head() Fact {
	c := s.facts.Cursors()
	if c.Unsafe == (eth.L2BlockRef{}) {
		return s.genesisFact()
	}
	if fact, ok := s.factByHash(c.Unsafe.Hash); ok {
		return fact
	}
	return s.genesisFact()
}

// rendering returns the header and body this node serves for a block: the stored rendering from the
// build job when there is one, otherwise the deterministic reconstruction.
//
// The reconstruction is sound because a silhouette block's body is a function of configuration: the
// transcoder emits EMPTY singular batches (G2 D6) and P takes no user deposits (DR-2), so the stock
// attributes builder produces exactly one transaction — the origin-accurate L1-info deposit — and
// that is the whole body. It refuses on a fork-activation block, where the stored rendering is the
// only authority (see RenderedBody).
func (s *Shim) rendering(ctx context.Context, fact Fact) (Rendering, error) {
	if r, ok := s.facts.Rendering(fact.Hash); ok {
		return r, nil
	}
	if s.isGenesis(fact.Number) {
		hdr, err := s.genesisHeader()
		if err != nil {
			return Rendering{}, err
		}
		return Rendering{Header: hdr, Txs: nil, Hash: fact.Hash}, nil
	}
	// A forced block carries its own header: the convention built it, and keeping it is what lets a
	// hash disagreement between implementations be diagnosed by diffing headers.
	parent, err := s.parentOf(fact)
	if err != nil {
		return Rendering{}, err
	}
	txs, origin, err := RenderedBody(ctx, s.params, s.l1, fact)
	if err != nil {
		return Rendering{}, err
	}
	if fact.Forced && fact.Header != nil {
		return Rendering{Header: fact.Header, Txs: txs, Hash: fact.Hash}, nil
	}
	hdr, err := RenderHeader(s.params, HeaderInputs{
		Parent:                   parent,
		Number:                   fact.Number,
		Timestamp:                fact.Timestamp,
		StateRoot:                fact.StateRoot,
		MessagePasserStorageRoot: fact.MessagePasserStorageRoot,
		Origin:                   origin,
		Txs:                      txs,
	})
	if err != nil {
		return Rendering{}, err
	}
	return Rendering{Header: hdr, Txs: txs, Hash: fact.Hash}, nil
}

func (s *Shim) genesisHeader() (*types.Header, error) {
	return RenderGenesisHeader(s.params)
}

// originFromAttributes reads the L1 origin and sequence number out of the attributes' first
// transaction, with the STOCK parser.
//
// This is the same code the CL uses for origin mapping and resets (`derive.L1BlockInfoFromBytes`,
// via the rule that kona's `L2BlockInfo::from_header_and_first_tx` is called and never
// reimplemented). Reading it back rather than taking the origin from somewhere else is what makes
// the rendered origin self-consistent across everything that reads it: the epoch the shim records is
// by construction the epoch the CL will read out of the block it serves.
func originFromAttributes(cfg *rollup.Config, attrs *eth.PayloadAttributes) (eth.BlockID, uint64, error) {
	if len(attrs.Transactions) == 0 {
		return eth.BlockID{}, 0, errors.New("payload attributes carry no transactions: an OP block's " +
			"first transaction is always the L1-info deposit")
	}
	dep, err := optypes.UnmarshalDepositTx(attrs.Transactions[0])
	if err != nil {
		return eth.BlockID{}, 0, fmt.Errorf("decode the L1-info deposit: %w", err)
	}
	info, err := derive.L1BlockInfoFromBytes(cfg, uint64(attrs.Timestamp), dep.Data)
	if err != nil {
		return eth.BlockID{}, 0, fmt.Errorf("parse the L1-info deposit: %w", err)
	}
	return eth.BlockID{Hash: info.BlockHash, Number: info.Number}, info.SequenceNumber, nil
}
