package silhouette

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivparams "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// The injected data source: the single seam that turns a proof batch posted on L1 into stock
// OP-Stack derivation input.
//
// It sits exactly where a DA plugin sits, and it is the ONLY non-stock thing on the derivation path.
// Its output is channel frames — the same bytes a batcher would have posted — so everything from
// FrameQueue onward is unmodified derivation running for real. That is the whole architectural bet:
// not a parallel pipeline that imitates derivation, but real derivation fed a synthesised input.
//
// Reading order: OpenData rewinds chaining state to match the L1 cursor (G2 D5), the iterator does
// the work lazily, accept applies docs/SPEC-WIRE-V3.md's acceptance rules 1-5, and transcode turns accepted
// blocks into frames under the rendered-origin convention (G2 D4).

// L1Source is the L1 access this source needs. Note what is absent: receipts. P takes no deposits
// (DR-2) and its SystemConfig is frozen, so nothing on this path ever walks an origin's receipts.
type L1Source interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error)
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
}

// BlobSource fetches the blobs an envelope travelled in.
type BlobSource interface {
	GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error)
}

// DataSource implements derive.DataAvailabilitySource over a proof-batch stream.
type DataSource struct {
	log      log.Logger
	cfg      *Config
	rollup   *rollup.Config
	spec     *rollup.ChainSpec
	l1Chain  *params.ChainConfig
	sysCfg   eth.SystemConfig
	l1       L1Source
	blobs    BlobSource
	verifier Verifier
	facts    *FactStore
	signer   types.Signer

	// sink feeds accepted blocks' exported logs into the interop log database. It is nil until
	// AttachLogSink is called, and a nil sink is a no-op rather than an error: deriving P's chain
	// and making P's messages referenceable are separable, and G2's own tests exercise the first
	// without the second.
	sink *LogSink

	// lastOpened is the L1 block most recently opened. It is what distinguishes ordinary forward
	// progress from a reset or a reorg, so the canonicality walk only runs when something actually
	// moved backwards.
	lastOpened eth.L1BlockRef
}

var _ derive.DataAvailabilitySource = (*DataSource)(nil)

// NewDataSource builds the injected source. sysCfg is P's FROZEN genesis SystemConfig (DR-2).
func NewDataSource(logger log.Logger, cfg *Config, rollupCfg *rollup.Config, l1Chain *params.ChainConfig,
	sysCfg eth.SystemConfig, l1 L1Source, blobs BlobSource, verifier Verifier, facts *FactStore,
) *DataSource {
	return &DataSource{
		log:      logger,
		cfg:      cfg,
		rollup:   rollupCfg,
		spec:     rollup.NewChainSpec(rollupCfg),
		l1Chain:  l1Chain,
		sysCfg:   sysCfg,
		l1:       l1,
		blobs:    blobs,
		verifier: verifier,
		facts:    facts,
		signer:   types.LatestSignerForChainID(new(big.Int).SetUint64(cfg.L1ChainID)),
	}
}

// AttachLogSink gives the source the interop log store it seals exported logs into.
//
// It is separate from NewDataSource because the two concerns are separate: a source with no sink
// derives P's chain and serves its facts, which is everything a single-chain verifier needs. The
// sink is what additionally makes P's exported messages referenceable BY OTHER CHAINS, and it only
// exists in a supernode that holds P's interop database. Call it once, before derivation starts.
func (s *DataSource) AttachLogSink(sink *LogSink) error {
	if s.sink != nil {
		return errors.New("silhouette source already has a log sink")
	}
	s.sink = sink
	return nil
}

func (s *DataSource) forcedParams() ForcedParams {
	return ForcedParams{Rollup: s.rollup, L1Chain: s.l1Chain, SysCfg: s.sysCfg}
}

// OpenData implements derive.DataAvailabilitySource.
//
// The rewind here IS the chaining-state ↔ stock-reset mapping (G2 D5). L1Retrieval calls OpenData
// both when traversal advances and when the pipeline resets, and it does not say which — so rather
// than trying to tell them apart, the source makes the same statement unconditionally: before block
// `ref` is read, the state is "every L1 block below ref.Number has been processed". Forward progress
// makes that a no-op; a reset makes it the correct rewind.
func (s *DataSource) OpenData(ctx context.Context, ref eth.L1BlockRef, _ common.Address) (derive.DataIter, error) {
	movedBack := ref.Number <= s.lastOpened.Number || ref.ParentHash != s.lastOpened.Hash
	droppedAny := false
	if dropped := s.facts.rewindToL1Below(ref.Number); dropped > 0 {
		droppedAny = true
		s.log.Info("rewound proven history to the L1 cursor", "l1", ref, "dropped_batches", dropped)
	}
	if movedBack && s.lastOpened != (eth.L1BlockRef{}) {
		// Something went backwards: a reset, or an L1 reorg. The rewind above covers it whenever the
		// reorg is shallower than the stock reset lookback, which is stock op-node's own assumption;
		// this walk removes the reliance on that assumption for the carriers still in the window.
		dropped, err := s.facts.dropOrphanedCarriers(func(c carrier) (bool, error) {
			got, err := s.l1.L1BlockRefByNumber(ctx, c.L1.Number)
			if errors.Is(err, ethereum.NotFound) {
				return false, nil // L1 is shorter than it was: this carrier cannot be canonical
			}
			if err != nil {
				return false, derive.NewTemporaryError(fmt.Errorf("fetch L1 block %d while walking back proven history: %w", c.L1.Number, err))
			}
			return got.Hash == c.L1.Hash, nil
		})
		if err != nil {
			return nil, err
		}
		if dropped > 0 {
			droppedAny = true
			s.log.Warn("dropped proof batches orphaned by an L1 reorg", "l1", ref, "batches", dropped)
		}
	}
	if droppedAny {
		// The facts and the exported messages have to move together. A batch whose carrier is no
		// longer canonical was never posted, so the messages it made referenceable must stop being
		// referenceable — otherwise another chain could execute against an initiating message that
		// nothing on L1 proves any more, which is the one thing the log store exists to prevent.
		head := s.head()
		if err := s.sink.Rewind(eth.BlockID{Hash: head.Hash, Number: head.Number}); err != nil {
			return nil, derive.NewTemporaryError(fmt.Errorf("rewind exported logs to the proven head: %w", err))
		}
	}
	s.lastOpened = ref
	return &dataIter{src: s, ref: ref}, nil
}

// head is the current chaining head: the tip of proven-or-forced history, or the configured anchor
// when the table is empty.
//
// Deriving it from the fact table rather than holding it in a field is what keeps acceptance a pure
// function of L1: the rewind moves the table, and the head follows automatically.
func (s *DataSource) head() Fact {
	if h, ok := s.facts.Head(); ok {
		return h
	}
	return s.anchor()
}

func (s *DataSource) anchor() Fact {
	return Fact{
		Number:     s.cfg.Anchor.BlockNumber,
		Timestamp:  s.cfg.Anchor.Timestamp,
		Hash:       s.cfg.Anchor.BlockHash,
		OutputRoot: s.cfg.Anchor.OutputRoot,
		L1Origin:   s.cfg.Anchor.L1Origin,
	}
}

// dataIter yields one payload per accepted proof batch found in an L1 block.
type dataIter struct {
	src  *DataSource
	ref  eth.L1BlockRef
	data [][]byte
	// opened distinguishes "not fetched yet" from "fetched, nothing here", which a nil slice alone
	// cannot: an L1 block with no proof batch in it is the common case.
	opened bool
}

var _ derive.DataIter = (*dataIter)(nil)

// Next returns the next channel-frame payload from this L1 block, or io.EOF once it is drained.
// Fetching is lazy, exactly as BlobDataSource's is, so a pipeline reset does no L1 work.
func (it *dataIter) Next(ctx context.Context) (eth.Data, error) {
	if !it.opened {
		data, err := it.open(ctx)
		if err != nil {
			return nil, err
		}
		it.data, it.opened = data, true
	}
	if len(it.data) == 0 {
		return nil, io.EOF
	}
	next := it.data[0]
	it.data = it.data[1:]
	return next, nil
}

func (it *dataIter) open(ctx context.Context) ([][]byte, error) {
	s := it.src
	_, txs, err := s.l1.InfoAndTxsByHash(ctx, it.ref.Hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// The block this pipeline is standing on is gone: that is a reorg, not a hiccup.
			return nil, derive.NewResetError(fmt.Errorf("failed to open silhouette data source at %s: %w", it.ref, err))
		}
		return nil, derive.NewTemporaryError(fmt.Errorf("failed to open silhouette data source at %s: %w", it.ref, err))
	}

	var out [][]byte
	for _, tx := range txs {
		if !s.isProofBatchTx(tx) {
			continue
		}
		payload, err := s.readEnvelope(ctx, it.ref, tx)
		if err != nil {
			// A failure to READ L1 must not be mistaken for a verdict about the batch: returning it
			// makes the step loop retry rather than pass over data it never saw.
			return nil, err
		}
		if payload == nil {
			continue // rejected and logged
		}
		frames, err := s.accept(ctx, payload, it.ref, tx.Hash())
		if err != nil {
			if isRetryable(err) {
				return nil, derive.NewResetError(fmt.Errorf("could not decide proof batch in tx %s: %w", tx.Hash(), err))
			}
			// Rule 2/3/4/5 rejection. A bad envelope from the submitter must NEVER halt a verifier:
			// skip it, log it, and keep deriving. The proven head only advances on accept, so a
			// duplicate or a replay is a no-op by the same rule that rejects a forgery.
			s.log.Warn("rejected proof batch", "l1", it.ref, "tx", tx.Hash(),
				"proven_head", s.head().Number, "err", err)
			continue
		}
		out = append(out, frames)
	}
	return out, nil
}

// readEnvelope fetches and blob-decodes one candidate. A nil payload with a nil error means the
// transaction was skipped and logged.
func (s *DataSource) readEnvelope(ctx context.Context, ref eth.L1BlockRef, tx *types.Transaction) ([]byte, error) {
	hashes := tx.BlobHashes()
	if len(hashes) == 0 {
		s.log.Warn("proof-batch transaction carries no blobs", "l1", ref, "tx", tx.Hash())
		return nil, nil
	}
	blobs, err := s.blobs.GetBlobsByHash(ctx, ref.Time, hashes)
	if errors.Is(err, ethereum.NotFound) {
		// The L1 block was readable, so its blobs should be too; if they are not, this node's view of
		// L1 is not one it can derive from.
		return nil, derive.NewResetError(fmt.Errorf("failed to fetch blobs of tx %s: %w", tx.Hash(), err))
	} else if err != nil {
		return nil, derive.NewTemporaryError(fmt.Errorf("failed to fetch blobs of tx %s: %w", tx.Hash(), err))
	}
	payload, err := proofbatch.FromBlobs(blobs)
	if err != nil {
		// Blob-encoding damage is the submitter's problem, not L1's: skip it and keep going, the same
		// way a malformed envelope is skipped a stage later.
		s.log.Warn("ignoring proof-batch transaction with undecodable blobs", "l1", ref, "tx", tx.Hash(), "err", err)
		return nil, nil
	}
	return payload, nil
}

// isProofBatchTx is acceptance rule 1: type-3, to the inbox, from the submitter. Nothing else can
// carry a proof batch, and no receipt status is consulted — a reverted transaction still put its
// blobs on L1, and the blobs are the data.
//
// The batcherAddr the pipeline passes to OpenData is deliberately ignored: it comes from a
// SystemConfig, and the submitter here is configuration rather than chain state. Giving this chain a
// real SystemConfig address is the hook that would make submitter rotation an L1 governance action.
func (s *DataSource) isProofBatchTx(tx *types.Transaction) bool {
	if tx.Type() != types.BlobTxType {
		return false
	}
	if to := tx.To(); to == nil || *to != s.cfg.Inbox {
		return false
	}
	sender, err := types.Sender(s.signer, tx)
	if err != nil {
		s.log.Warn("proof-batch inbox transaction has an invalid signature", "tx", tx.Hash(), "err", err)
		return false
	}
	return sender == s.cfg.Submitter
}

// retryableError marks a failure to read L1 rather than a failure of the batch itself. The
// distinction is the whole reject-and-log rule: a rejected batch is skipped forever, a retryable one
// is retried against the same L1 block.
type retryableError struct{ error }

func isRetryable(err error) bool {
	var r retryableError
	return errors.As(err, &r)
}

func retryable(err error) error { return retryableError{err} }

// accept applies docs/SPEC-WIRE-V3.md's verifier acceptance rules and, only if all of them hold,
// records the batch's facts and returns the channel frames it transcodes to.
func (s *DataSource) accept(ctx context.Context, payload []byte, l1 eth.L1BlockRef, txHash common.Hash) ([]byte, error) {
	// Rule 2: the envelope, at EXACTLY the version this node is configured for. The version is part
	// of the acceptance rule rather than a parsing detail, because it decides whether this chain's
	// import list exists at all — and therefore whether its dependencies are checked or trusted.
	env, err := proofbatch.DecodeAs(payload, s.cfg.wireVersion())
	if err != nil {
		return nil, err
	}
	b := &env.Batch
	if err := b.CheckStructure(); err != nil {
		return nil, err
	}
	deniedIndex := -1
	for i, blk := range b.Blocks {
		denied, err := s.facts.Denied(blk.Number, blk.Hash)
		if err != nil {
			return nil, retryable(fmt.Errorf("check durable denial for block %d (%s): %w", blk.Number, blk.Hash, err))
		}
		if denied {
			deniedIndex = i
			break
		}
	}
	// Acceptance rule (G7G D2 / G7R D10): no block may consume a message stamped at or above its own
	// timestamp. The wire format PERMITS such a batch — it is well-formed — and this node refuses it,
	// because a same-timestamp import carries no position and the stock same-timestamp cycle graph
	// orders executing messages by position. Refusing here makes the failure a rejected batch, which
	// is recoverable by the prover posting a correct one, without first replacing an L2 block.
	if err := b.CheckNoSameTimestampImports(); err != nil {
		return nil, err
	}

	// Rule 3: chaining. A batch extends the chaining head or it is not this chain's history — which
	// is also what makes a replayed or duplicated batch a no-op.
	//
	// The head may be a FORCED head: if the prover stalled long enough for the window to expire,
	// stock derivation force-generated blocks and the resuming batch extends the last of them. Those
	// blocks are never on the wire (G1 D8.1) — both sides compute them — so the forced extension is
	// derived here, from the batch's OWN l1Head rather than from this node's live cursor, which is
	// the only race-free reading available (G2 F4).
	head := s.head()
	supersedes := false
	if b.PrevOutputRoot != head.OutputRoot {
		base, ok, err := s.facts.SupersessionBase(b.PrevOutputRoot)
		if err != nil {
			return nil, fmt.Errorf("check durable denial for proof supersession: %w", err)
		}
		if ok {
			head, supersedes = base, true
		} else {
			anchor := s.anchor()
			if b.PrevOutputRoot == anchor.OutputRoot {
				allowed, err := s.facts.AnchorSupersession(anchor)
				if err != nil {
					return nil, fmt.Errorf("check durable anchor denial for proof supersession: %w", err)
				}
				if allowed {
					head, supersedes = anchor, true
				}
			}
		}
		if !supersedes {
			resumed, err := s.resumeHead(ctx, head, b, l1)
			if err != nil {
				return nil, err
			}
			head = resumed
		}
	}
	if b.NewOutputRoot == (common.Hash{}) {
		return nil, errors.New("batch claims a zero output root")
	}
	if b.NewOutputRoot == b.PrevOutputRoot {
		return nil, errors.New("batch does not advance the output root")
	}
	if want := head.Number + 1; b.Blocks[0].Number != want {
		return nil, fmt.Errorf("batch starts at block %d, the chaining head expects %d", b.Blocks[0].Number, want)
	}
	// The last block's three committed roots derive an output root; if that is not newOutputRoot the
	// batch is internally inconsistent whatever the proof says about it.
	last := b.Blocks[len(b.Blocks)-1]
	if got := last.OutputRoot(); got != b.NewOutputRoot {
		return nil, fmt.Errorf("batch claims new output root %s but block %d's committed roots derive %s",
			b.NewOutputRoot, last.Number, got)
	}
	// G2 D6: the whole envelope must validate and prove together, so timestamps are checked for exact
	// block-time spacing here rather than left to the stock batch queue, which would silently drop
	// mistimed batches downstream and leave the chaining head advanced past blocks nothing derived.
	// Only after all checks pass may replay omit a suffix carrying an already-denied block.
	prevTime := head.Timestamp
	for i := range b.Blocks {
		if b.Blocks[i].Hash == (common.Hash{}) {
			return nil, fmt.Errorf("block %d exports a zero block hash", b.Blocks[i].Number)
		}
		if want := prevTime + s.rollup.BlockTime; b.Blocks[i].Timestamp != want {
			return nil, fmt.Errorf("block %d has timestamp %d, expected %d (%ds block time)",
				b.Blocks[i].Number, b.Blocks[i].Timestamp, want, s.rollup.BlockTime)
		}
		prevTime = b.Blocks[i].Timestamp
	}

	// Rule 4: config binding. A proof of a stock derivation is only a proof of THIS chain's history
	// if it derived under this chain's rollup config and dependency set; without these two, a valid
	// proof of some other chain would be accepted as P's.
	if want := s.cfg.RollupConfigHash; b.RollupConfigHash != want {
		return nil, fmt.Errorf("batch derived rollup config %s, this node requires %s", b.RollupConfigHash, want)
	}
	if want := s.cfg.DepSetHash; b.DepSetHash != want {
		return nil, fmt.Errorf("batch derived dependency set %s, this node requires %s", b.DepSetHash, want)
	}
	if want := s.cfg.ExportPolicy(); b.ExportPolicyHash != want {
		return nil, fmt.Errorf("batch commits to export policy %s, this node requires %s", b.ExportPolicyHash, want)
	}
	if err := s.checkL1Head(ctx, b.L1Head, l1); err != nil {
		return nil, err
	}

	// Rule 5: the proof itself, over the exact bytes that were posted — never a re-encoding.
	if err := s.verifier.Verify(env.PublicValues, env.Proof); err != nil {
		return nil, err
	}

	// An L1 replay after replacement still encounters the original proof before the corrected one.
	// Keep its verified prefix so stock derivation reaches the parent of the denied block, but never
	// reintroduce that block or anything descended from it. The whole original envelope is verified
	// above before its suffix is omitted, so every retained fact remains proof-committed.
	blocks := b.Blocks
	newOutputRoot := b.NewOutputRoot
	if deniedIndex >= 0 {
		denied := b.Blocks[deniedIndex]
		if deniedIndex == 0 {
			return nil, fmt.Errorf("block %d (%s) was denied by the cross-safety judge", denied.Number, denied.Hash)
		}
		blocks = b.Blocks[:deniedIndex]
		last = blocks[len(blocks)-1]
		newOutputRoot = last.OutputRoot()
		s.log.Warn("omitting denied proof suffix during L1 replay",
			"denied_block", denied.Number, "denied_hash", denied.Hash,
			"retained_first", blocks[0].Number, "retained_last", last.Number)
	}

	// Accepted. Transcode BEFORE recording anything: the transcode assigns each block its rendered
	// origin, and those origins are part of the facts. A failure here is not the batch's fault, so
	// it must not be recorded as a rejection either — it is returned as retryable.
	frames, origins, err := s.transcode(ctx, head, blocks, l1, b.PrevOutputRoot, newOutputRoot)
	if err != nil {
		return nil, err
	}
	if supersedes {
		oldHead := s.head()
		if err := s.sink.Rewind(eth.BlockID{Number: head.Number, Hash: head.Hash}); err != nil {
			return nil, retryable(fmt.Errorf("rewind exported logs for proof supersession: %w", err))
		}
		if head.Number == s.cfg.Anchor.BlockNumber {
			s.facts.replaceAllForSupersession()
		} else if err := s.facts.ReplaceSuffix(head); err != nil {
			return nil, err
		}
		s.log.Warn("accepted a replacement proof for a denied suffix",
			"from", head.Number+1, "old_head", oldHead.Number, "new_head", last.Number)
	}
	// Seal the exported logs BEFORE the facts move. The fact table IS the proven head, so this
	// ordering is what makes "the head can never claim history the log store does not hold" true
	// rather than merely intended: a sink failure returns before RecordProven, leaving the head
	// where it was, and the pipeline derives the same batch again.
	if err := s.sink.Accept(blocks, head.Hash); err != nil {
		return nil, fmt.Errorf("seal exported logs for blocks %d-%d: %w", blocks[0].Number, last.Number, err)
	}
	for i, blk := range blocks {
		s.facts.RecordProven(blk, origins[i].origin, origins[i].seqNumber, env.Version)
	}
	s.facts.recordCarrier(carrier{
		L1: l1.ID(), L1Time: l1.Time,
		FirstBlock: blocks[0].Number, LastBlock: last.Number, LastHash: last.Hash,
		ParentHash: head.Hash, PrevOutputRoot: b.PrevOutputRoot, NewOutputRoot: newOutputRoot,
	})
	s.log.Info("accepted proof batch", "l1", l1, "tx", txHash, "blocks", len(blocks),
		"first", blocks[0].Number, "last", last.Number, "last_hash", last.Hash,
		"output_root", newOutputRoot, "proof_type", s.verifier.ProofType())
	return frames, nil
}

// resumeHead handles a batch whose prevOutputRoot is not the current head: it may be resuming over a
// forced extension. The forced blocks are computed, and accepted only if the last of them is exactly
// what the batch claims to extend — a prefix match is not enough, because the resume anchor is
// defined to be the forced HEAD.
func (s *DataSource) resumeHead(ctx context.Context, head Fact, b *proofbatch.ProofBatch, l1 eth.L1BlockRef) (Fact, error) {
	// The forced extension is measured at the batch's own l1Head, capped at the carrying block: the
	// wire value is what both sides can agree on, and it is already checked canonical and
	// depth-bounded by rule 4.
	l1HeadRef, err := s.l1.L1BlockRefByHash(ctx, b.L1Head)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return Fact{}, fmt.Errorf("batch extends %s but the chaining head is %s, and its l1Head %s is unknown",
				b.PrevOutputRoot, head.OutputRoot, b.L1Head)
		}
		return Fact{}, retryable(fmt.Errorf("fetch l1Head %s: %w", b.L1Head, err))
	}
	pipelineOrigin := min(l1HeadRef.Number, l1.Number)

	forced, err := ForcedExtension(ctx, s.forcedParams(), s.l1, head, pipelineOrigin)
	if err != nil {
		return Fact{}, retryable(fmt.Errorf("compute forced extension above block %d: %w", head.Number, err))
	}
	if len(forced) == 0 {
		return Fact{}, fmt.Errorf("batch extends %s but the chaining head is %s and no forced extension is due",
			b.PrevOutputRoot, head.OutputRoot)
	}
	tip := forced[len(forced)-1]
	if tip.OutputRoot != b.PrevOutputRoot {
		return Fact{}, fmt.Errorf("batch extends %s; the chaining head is %s and the forced extension of %d block(s) reaches %s",
			b.PrevOutputRoot, head.OutputRoot, len(forced), tip.OutputRoot)
	}
	for _, f := range forced {
		s.facts.Record(f)
	}
	s.log.Info("resumed over a forced extension", "forced_blocks", len(forced),
		"from", head.Number, "to", tip.Number, "output_root", tip.OutputRoot)
	return tip, nil
}

// checkL1Head requires the L1 head the batch's own derivation ran against to be a block on the chain
// this node follows, no deeper than the configured depth and no later than the block that carried
// the batch.
//
// This is the whole of the batch's L1-side binding: everything the chain consumed from L1 was read
// under this head by whatever produced the batch, so pinning the head to this node's canonical chain
// pins all of it. Depth is measured against the CARRYING block rather than this node's live head so
// acceptance is a function of L1 alone — proven history is re-derived from l1StartBlock on every
// restart, and a rule that referred to "now" would accept a batch on first sight and reject the same
// batch on replay.
func (s *DataSource) checkL1Head(ctx context.Context, headHash common.Hash, l1 eth.L1BlockRef) error {
	head, err := s.l1.L1BlockRefByHash(ctx, headHash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return fmt.Errorf("l1Head %s is unknown to this node", headHash)
		}
		return retryable(fmt.Errorf("fetch l1Head %s: %w", headHash, err))
	}
	if head.Number > l1.Number {
		return fmt.Errorf("l1Head %d is above the L1 block %d that carried the batch", head.Number, l1.Number)
	}
	if l1.Number-head.Number > s.cfg.l1HeadMaxDepth() {
		return fmt.Errorf("l1Head %d is %d blocks below the carrying block, max %d",
			head.Number, l1.Number-head.Number, s.cfg.l1HeadMaxDepth())
	}
	// Ancestry: an L1 block known by hash may still be off this node's canonical chain, so the
	// canonical block at that height must be the same block.
	canonical, err := s.l1.L1BlockRefByNumber(ctx, head.Number)
	if err != nil {
		return retryable(fmt.Errorf("fetch canonical L1 block %d: %w", head.Number, err))
	}
	if canonical.Hash != headHash {
		return fmt.Errorf("l1Head %s is not canonical: block %d is %s", headHash, head.Number, canonical.Hash)
	}
	return nil
}

// rendered is one block's rendered L1 origin assignment.
type rendered struct {
	origin    eth.BlockID
	seqNumber uint64
}

// maxFrameSize bounds one emitted frame. The payload this source hands the pipeline is shaped like
// a batcher's calldata transaction, so the bound only has to be large enough to be efficient and
// small enough to stay inside the frame format's length field.
const maxFrameSize = 120_000

// transcode turns proven blocks into a stock channel: one empty singular batch per block, framed
// exactly as a batcher would have framed them.
//
// Three things are doing the work here.
//
// The batches' transaction lists are EMPTY. The stock attributes builder prepends the L1-info
// deposit and P has no user deposits (DR-2), so a rendered block is single-transaction and its
// interior stays private by construction (G2 D6) — there is nothing to put in the list, and that is
// the design rather than a limitation.
//
// Each batch's ParentHash is the previous proven block's REAL hash off the wire. That is what binds
// the rendered chain to the proof-committed hashes: the stock batch queue checks it against the
// engine's head, so real hash in, real hash out, with nothing re-hashing a payload anywhere.
//
// And the batch is built by handing the STOCK converter a payload rather than by constructing a
// SingularBatch directly. That costs a synthesised L1-info transaction and buys the thing worth
// having: the L1-info bytes are parsed straight back by PayloadToSingularBatch, so the epoch this
// source renders is by construction the epoch the stock CL will read out of those same bytes. An
// origin-mapping disagreement cannot survive the round trip, which is exactly where a
// hand-rolled encoder would have been free to drift.
func (s *DataSource) transcode(ctx context.Context, head Fact, blocks []proofbatch.BlockExport, carrier eth.L1BlockRef, prevRoot, newRoot common.Hash) ([]byte, []rendered, error) {
	compressor, err := newChannelCompressor()
	if err != nil {
		return nil, nil, retryable(err)
	}
	out, err := derive.NewSingularChannelOut(compressor, s.spec)
	if err != nil {
		return nil, nil, retryable(fmt.Errorf("open channel: %w", err))
	}

	origins := make([]rendered, len(blocks))
	parentHash := head.Hash
	origin, err := s.l1.L1BlockRefByNumber(ctx, head.L1Origin.Number)
	if err != nil {
		return nil, nil, retryable(fmt.Errorf("fetch rendered origin %d: %w", head.L1Origin.Number, err))
	}
	seqNumber := head.SeqNumber

	for i, blk := range blocks {
		origin, seqNumber, err = s.advanceOrigin(ctx, origin, seqNumber, blk.Timestamp, carrier)
		if err != nil {
			return nil, nil, err
		}
		info, err := s.l1.InfoByHash(ctx, origin.Hash)
		if err != nil {
			return nil, nil, retryable(fmt.Errorf("fetch rendered origin %s: %w", origin, err))
		}
		l1InfoTx, err := derive.L1InfoDepositBytes(s.rollup, s.l1Chain, s.sysCfg, seqNumber, info, blk.Timestamp)
		if err != nil {
			return nil, nil, fmt.Errorf("build L1-info tx for block %d: %w", blk.Number, err)
		}
		origins[i] = rendered{origin: origin.ID(), seqNumber: seqNumber}
		if _, err := out.AddBlock(s.rollup, &eth.ExecutionPayload{
			ParentHash:   parentHash,
			BlockNumber:  hexutil.Uint64(blk.Number),
			BlockHash:    blk.Hash,
			Timestamp:    hexutil.Uint64(blk.Timestamp),
			Transactions: []hexutil.Bytes{l1InfoTx},
		}); err != nil {
			return nil, nil, retryable(fmt.Errorf("add block %d to channel: %w", blk.Number, err))
		}
		parentHash = blk.Hash
	}

	if err := out.Close(); err != nil {
		return nil, nil, retryable(fmt.Errorf("close channel: %w", err))
	}
	// One payload carries the whole channel, the way a single batcher transaction would. Because an
	// envelope is one L1 transaction and a channel never spans L1 blocks here, nothing is ever
	// half-read across a reset — which is why the reset lookback in G2 D5 is zero.
	framed := bytes.NewBuffer([]byte{derivparams.DerivationVersion0})
	for {
		if _, err := out.OutputFrame(framed, maxFrameSize); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, retryable(fmt.Errorf("output frame: %w", err))
		}
	}

	// Restamp the frames with a DETERMINISTIC channel ID.
	//
	// NewSingularChannelOut picks a random one, which is right for a batcher — two concurrent
	// channels must not collide — and wrong here. This source must be a pure function of L1: the same
	// L1 block re-read after a reset has to produce the same bytes, and G6's superroot program has to
	// be able to derive the same frames in-circuit. A random ID makes both impossible. Deriving it
	// from the batch's own output roots gives an ID that is unique per batch (the roots chain, so no
	// two batches share a pair) and computable by anyone re-deriving the same history.
	frames, err := derive.ParseFrames(framed.Bytes())
	if err != nil {
		return nil, nil, retryable(fmt.Errorf("re-read own frames: %w", err))
	}
	id := channelIDFor(prevRoot, newRoot)
	payload := bytes.NewBuffer([]byte{derivparams.DerivationVersion0})
	for i := range frames {
		frames[i].ID = id
		if err := frames[i].MarshalBinary(payload); err != nil {
			return nil, nil, retryable(fmt.Errorf("re-frame: %w", err))
		}
	}
	return payload.Bytes(), origins, nil
}

// channelIDFor derives a channel's ID from the batch it carries, so that re-deriving the same L1
// history reproduces the same frames byte for byte.
func channelIDFor(prevRoot, newRoot common.Hash) derive.ChannelID {
	var id derive.ChannelID
	copy(id[:], crypto.Keccak256(prevRoot[:], newRoot[:])[:len(id)])
	return id
}

// advanceOrigin is the rendered-origin rule (G2 D4): the greedy rule a stock sequencer uses, which is
// to adopt the next origin as soon as its timestamp is not in the future relative to the block being
// built. It is a convention rather than a wire fact — the wire carries no per-block origin — and it
// is chosen because it minimises drift and is therefore the assignment most likely to be stock-legal
// for every block.
//
// The advance is capped at the carrying L1 block: a batch is read while the pipeline stands on its
// carrier, so an origin above it would name a block the pipeline has not traversed. In practice the
// cap never binds, because a proof is posted after the blocks it proves.
func (s *DataSource) advanceOrigin(ctx context.Context, origin eth.L1BlockRef, seqNumber uint64, blockTime uint64, carrier eth.L1BlockRef) (eth.L1BlockRef, uint64, error) {
	advanced := false
	for origin.Number < carrier.Number {
		next, err := s.l1.L1BlockRefByNumber(ctx, origin.Number+1)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				break
			}
			return origin, 0, retryable(fmt.Errorf("fetch candidate origin %d: %w", origin.Number+1, err))
		}
		if next.Time > blockTime {
			break // adopting it would point this L2 block at a future L1 block
		}
		origin, advanced = next, true
	}
	if advanced {
		seqNumber = 0
	} else {
		seqNumber++
	}
	if origin.Time > blockTime {
		return origin, 0, fmt.Errorf("block at %d precedes its rendered origin %d (time %d): "+
			"the batch's blocks are older than the head's own origin", blockTime, origin.Number, origin.Time)
	}
	if drift := blockTime - origin.Time; drift > s.spec.MaxSequencerDrift(origin.Time) {
		return origin, 0, fmt.Errorf("block at %d drifts %ds past rendered origin %d, max %ds",
			blockTime, drift, origin.Number, s.spec.MaxSequencerDrift(origin.Time))
	}
	return origin, seqNumber, nil
}
