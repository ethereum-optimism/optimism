package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// rpcCaller is the sliver of an RPC client this tool needs, which is also what makes the builder
// testable without any node.
type rpcCaller interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

// cursor is the submitter's memory between batches: where the last one ended. It is persisted so
// a restarted submitter chains onto its own history instead of re-posting a range the verifier
// has already accepted.
type cursor struct {
	LastBlock  uint64      `json:"lastBlock"`
	OutputRoot common.Hash `json:"outputRoot"`
}

func loadCursor(path string) (*cursor, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cursor %q: %w", path, err)
	}
	var c cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse cursor %q: %w", path, err)
	}
	return &c, nil
}

func saveCursor(path string, c *cursor) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write cursor %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace cursor %q: %w", path, err)
	}
	return nil
}

// builderConfig configures the attested envelope builder.
type builderConfig struct {
	// RollupConfigHash and DepSetHash are the commitments a prover would compute from what it derived
	// under. An attested batch derives nothing, so they are configuration here: they must be the values
	// the verifier is configured with, or it refuses every batch.
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	// MaxBlocks caps a batch's block count. 300 is one 10-minute cadence at 2s blocks.
	MaxBlocks uint64
	// L1Lag is how far below the L1 head the batch's claimed l1Head sits. It stands in for the
	// head a real derivation would have run against, and keeps the batch clear of an L1 reorg.
	L1Lag uint64
	// CursorPath persists the cursor between batches.
	CursorPath string
	// Head selects which of the chain's own heads bounds a batch. On a silhouette chain this MUST
	// be HeadUnsafe; see the type's doc for why anything else deadlocks.
	Head HeadSource
}

// HeadSource says which head the submitter batches up to.
//
// This is the one knob on this tool that is a correctness question rather than a tuning question,
// and it is a decision the alt-DA era did not have to make.
//
// On a chain WITH a batcher, the safe head is the right answer and was the original one: the batcher
// posts to L1, stock derivation makes blocks safe, and the submitter proves history that L1 already
// carries. Safe is also the conservative choice, because a safe block cannot be reorged away by the
// sequencer under it.
//
// On a SILHOUETTE chain the same choice deadlocks, and it deadlocks silently. There is no batcher.
// The only thing that advances P's safe head is derivation, P's only derivation source is the
// proof-batch inbox, and the only thing that writes the proof-batch inbox is THIS TOOL. So
// `safeHead == cursor.LastBlock` from the first cycle onward, `next()` takes its `safe <= LastBlock`
// early return forever, and the submitter sits in a healthy-looking loop posting nothing while P's
// public history never starts. The sequencer-side supernode makes this exact: in the `proven-head`
// posture (RUNBOOK §8) P's public labels are READ FROM the proven head, so safe is definitionally
// the last thing this tool proved.
//
// The unsafe head is the only head on a silhouette chain that moves for a reason external to the
// proof loop: it is what P's private sequencer has actually produced. Batching up to it is what
// makes the chain's public rendering follow its private execution instead of following itself.
//
// The cost is stated rather than hidden: an unsafe block CAN be reorged by its own sequencer (a
// crash before the head is persisted). If that happens after the block was proven, P's public
// history and its private history disagree, which is a P-level reorg — and DR-RESUME already rules
// on it (P reorgs and re-proves; the burned proof is wasted work by design). `HeadSafe` is kept
// reachable so the pre-rotation behaviour stays available and this change stays reversible by
// configuration.
type HeadSource uint8

const (
	// HeadUnsafe batches up to the chain's own unsafe head. The silhouette default.
	HeadUnsafe HeadSource = iota
	// HeadSafe batches up to the safe head. Correct only for a chain that has a batcher.
	HeadSafe
)

// ParseHeadSource resolves the CLI spelling. An unknown value is an error rather than a default,
// because both values are legal-looking and picking the wrong one is invisible until nothing
// batches.
func ParseHeadSource(s string) (HeadSource, error) {
	switch s {
	case "unsafe":
		return HeadUnsafe, nil
	case "safe":
		return HeadSafe, nil
	default:
		return 0, fmt.Errorf("unknown head source %q: want \"unsafe\" (silhouette) or \"safe\" (batcher-backed)", s)
	}
}

func (h HeadSource) String() string {
	if h == HeadSafe {
		return "safe"
	}
	return "unsafe"
}

// builder assembles proof batches from a live chain's own outputs. It is the ATTESTED path, which is
// to say v1's path: the wire format, the blob transport and every binding a verifier checks are real,
// and the proof slot is empty because in this mode the operator's signature on the L1 transaction is
// what stands behind the batch. A prover would fill that slot and change nothing else.
type builder struct {
	cfg    builderConfig
	l1     rpcCaller
	l2     rpcCaller
	rollup rpcCaller
	cur    *cursor
}

func newBuilder(cfg builderConfig, l1, l2, rollup rpcCaller) (*builder, error) {
	if cfg.MaxBlocks == 0 {
		return nil, fmt.Errorf("max blocks must be non-zero")
	}
	cur, err := loadCursor(cfg.CursorPath)
	if err != nil {
		return nil, err
	}
	return &builder{cfg: cfg, l1: l1, l2: l2, rollup: rollup, cur: cur}, nil
}

// next builds the batch that extends the cursor, or nil if the chain has produced nothing new.
func (b *builder) next(ctx context.Context) (*proofbatch.ProofBatch, error) {
	head, err := b.head(ctx)
	if err != nil {
		return nil, err
	}
	cur := b.cur
	if cur == nil {
		// First batch: anchor on the current head so the first range is small and the anchor a
		// verifier is configured with is unambiguous.
		root, err := b.outputRoot(ctx, head)
		if err != nil {
			return nil, err
		}
		cur = &cursor{LastBlock: head, OutputRoot: root}
		b.cur = cur
		if err := b.persist(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("anchored at L2 block %d (output root %s); "+
			"configure the verifier with this anchor, then batches follow", head, root)
	}
	if head <= cur.LastBlock {
		return nil, nil
	}
	last := head
	if last-cur.LastBlock > b.cfg.MaxBlocks {
		last = cur.LastBlock + b.cfg.MaxBlocks
	}

	l1Head, err := b.l1Head(ctx)
	if err != nil {
		return nil, err
	}
	batch := &proofbatch.ProofBatch{
		PrevOutputRoot:   cur.OutputRoot,
		L1Head:           l1Head,
		RollupConfigHash: b.cfg.RollupConfigHash,
		DepSetHash:       b.cfg.DepSetHash,
		ExportPolicyHash: proofbatch.ExportPolicyAllHashes,
	}
	for n := cur.LastBlock + 1; n <= last; n++ {
		export, err := b.blockExport(ctx, n)
		if err != nil {
			return nil, err
		}
		batch.Blocks = append(batch.Blocks, *export)
	}
	// The head root is DERIVED from the last block's committed roots rather than asked for
	// separately: that is what a v2 verifier checks, so building it any other way would produce
	// batches that only a lenient verifier accepts.
	batch.NewOutputRoot = batch.Blocks[len(batch.Blocks)-1].OutputRoot()
	if err := batch.CheckStructure(); err != nil {
		return nil, fmt.Errorf("built an invalid batch: %w", err)
	}
	return batch, nil
}

// commit advances the cursor after a batch has landed on L1.
func (b *builder) commit(batch *proofbatch.ProofBatch) error {
	b.cur = &cursor{
		LastBlock:  batch.Blocks[len(batch.Blocks)-1].Number,
		OutputRoot: batch.NewOutputRoot,
	}
	return b.persist()
}

func (b *builder) persist() error {
	if b.cfg.CursorPath == "" {
		return nil
	}
	if dir := filepath.Dir(b.cfg.CursorPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return saveCursor(b.cfg.CursorPath, b.cur)
}

// head reads the configured head from the chain's own sync status. Both heads are read from the one
// response, so the log line below can report the gap between them — on a healthy silhouette chain
// that gap IS the proof lag, and on a misconfigured one it is the symptom.
func (b *builder) head(ctx context.Context) (uint64, error) {
	var status struct {
		UnsafeL2 struct {
			Number uint64 `json:"number"`
		} `json:"unsafe_l2"`
		SafeL2 struct {
			Number uint64 `json:"number"`
		} `json:"safe_l2"`
	}
	if err := b.rollup.CallContext(ctx, &status, "optimism_syncStatus"); err != nil {
		return 0, fmt.Errorf("fetch sync status: %w", err)
	}
	if b.cfg.Head == HeadSafe {
		return status.SafeL2.Number, nil
	}
	if status.UnsafeL2.Number < status.SafeL2.Number {
		// Not a condition the chain should ever be in; refuse rather than build a batch from a
		// head that is behind the safe head we would be chaining onto.
		return 0, fmt.Errorf("unsafe head %d is below the safe head %d",
			status.UnsafeL2.Number, status.SafeL2.Number)
	}
	return status.UnsafeL2.Number, nil
}

func (b *builder) outputRoot(ctx context.Context, block uint64) (common.Hash, error) {
	out, err := b.outputAtBlock(ctx, block)
	if err != nil {
		return common.Hash{}, err
	}
	return out.OutputRoot, nil
}

// rpcOutput is the part of optimism_outputAtBlock this tool reads. One call answers everything a
// v2 block export needs about a block except its logs: its identity, its timestamp, and the two
// roots the output root derives from.
type rpcOutput struct {
	OutputRoot            common.Hash `json:"outputRoot"`
	StateRoot             common.Hash `json:"stateRoot"`
	WithdrawalStorageRoot common.Hash `json:"withdrawalStorageRoot"`
	BlockRef              struct {
		Hash common.Hash `json:"hash"`
		Time uint64      `json:"timestamp"`
	} `json:"blockRef"`
}

func (b *builder) outputAtBlock(ctx context.Context, block uint64) (*rpcOutput, error) {
	var out rpcOutput
	if err := b.rollup.CallContext(ctx, &out, "optimism_outputAtBlock", hexutil.Uint64(block)); err != nil {
		return nil, fmt.Errorf("fetch output at %d: %w", block, err)
	}
	return &out, nil
}

// blockExport reads one L2 block's committed facts and its logs, hashing every log the way the
// interop LogsDB does. Log indices are taken from the node's own numbering rather than from the
// position a log happens to occupy in the response, because that index is what an executing
// message references and what a curated export policy would preserve.
func (b *builder) blockExport(ctx context.Context, number uint64) (*proofbatch.BlockExport, error) {
	out, err := b.outputAtBlock(ctx, number)
	if err != nil {
		return nil, err
	}
	var receipts []rpcReceipt
	if err := b.l2.CallContext(ctx, &receipts, "eth_getBlockReceipts", hexutil.Uint64(number)); err != nil {
		return nil, fmt.Errorf("fetch receipts of L2 block %d: %w", number, err)
	}
	export := &proofbatch.BlockExport{
		Number:                   number,
		Timestamp:                out.BlockRef.Time,
		Hash:                     out.BlockRef.Hash,
		StateRoot:                out.StateRoot,
		MessagePasserStorageRoot: out.WithdrawalStorageRoot,
	}
	// The node published this block's output root itself, so deriving it from the three roots
	// this batch will commit to is a free check that the export is internally consistent — and it
	// is exactly the check the verifier will run.
	if derived := export.OutputRoot(); derived != out.OutputRoot {
		return nil, fmt.Errorf("block %d: derived output root %s but the node reports %s",
			number, derived, out.OutputRoot)
	}
	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			export.Logs = append(export.Logs, proofbatch.LogExport{
				Index: uint32(l.LogIndex),
				Hash: messages.LogToLogHash(&types.Log{
					Address: l.Address,
					Topics:  l.Topics,
					Data:    l.Data,
				}),
				// The v2 default policy exports hashes only; a preimage-bearing policy is a
				// config change on both sides, not a change to this tool's wire.
			})
		}
	}
	return export, nil
}

// rpcReceipt / rpcLog read only the fields a log export needs: the three a log hash covers, plus
// the block-level index the node assigns.
type rpcReceipt struct {
	Logs []rpcLog `json:"logs"`
}

type rpcLog struct {
	Address  common.Address `json:"address"`
	Topics   []common.Hash  `json:"topics"`
	Data     hexutil.Bytes  `json:"data"`
	LogIndex hexutil.Uint64 `json:"logIndex"`
}

// l1Head picks the L1 block a batch claims to have been derived against: the L1 head minus the
// configured lag, so the batch stays clear of a reorg the verifier has not seen.
func (b *builder) l1Head(ctx context.Context) (common.Hash, error) {
	var head hexutil.Uint64
	if err := b.l1.CallContext(ctx, &head, "eth_blockNumber"); err != nil {
		return common.Hash{}, fmt.Errorf("fetch L1 head: %w", err)
	}
	if uint64(head) < b.cfg.L1Lag {
		return common.Hash{}, fmt.Errorf("L1 head %d is below the configured lag %d", head, b.cfg.L1Lag)
	}
	number := uint64(head) - b.cfg.L1Lag

	var header struct {
		Hash common.Hash `json:"hash"`
	}
	if err := b.l1.CallContext(ctx, &header, "eth_getBlockByNumber", hexutil.Uint64(number), false); err != nil {
		return common.Hash{}, fmt.Errorf("fetch L1 block %d: %w", number, err)
	}
	return header.Hash, nil
}
