package sysgo

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// The in-test proof-batch submitter: the producer end of the pipe whose consumer end is a
// silhouette verifier.
//
// It is a port of op-supernode/cmd/proofbatch-submitter's attested builder rather than a
// reimplementation of it — same log-hash construction, same derivation of newOutputRoot from the
// last block's committed roots, same internal output-root cross-check — because the thing under
// test is the WIRE, and a harness that built the wire its own way would be testing agreement
// between two of its own opinions. The binary itself is package main, so the code cannot be shared;
// the codec it frames with (op-supernode/proofbatch) is, and is used directly here.
//
// Three deliberate differences from the CLI tool, all of them about being drivable from a test:
//
//   - The cursor is SEEDED from the configured anchor instead of self-anchoring on the current
//     head. The tool anchors itself because an operator has to discover an anchor to configure a
//     verifier with; a test writes both sides at once, so anchoring at P's genesis makes the whole
//     of P's history provable and the first batch start at block 1.
//   - Batches are bounded by an explicit block number, not only by the head. That is what lets a
//     test post P's early history and then WITHHOLD the batch covering a particular block, which is
//     the only way to make "cross-safe pins, then advances" a real assertion rather than a race.
//   - There is no persistence. A restarted submitter is not a scenario a single test process has.

// silhouetteHeadUnsafe is not a knob here, and the reason is worth stating where the code is:
// a silhouette chain's batches are built up to its UNSAFE head.
//
// On a chain WITH a batcher the safe head would be the conservative answer. On a silhouette chain it
// deadlocks, silently: there is no batcher, so the only thing that advances P's safe head is
// derivation, P's only derivation source is the proof-batch inbox, and the only thing that writes
// that inbox is this submitter. Batching on the safe head would make the bound equal to the last
// thing this code proved, forever, while every log line looked healthy.

// ProofBatchSubmitter builds silhouette proof batches from a live chain's own outputs and posts them
// to the L1 inbox as blob transactions.
type ProofBatchSubmitter struct {
	p      devtest.T
	log    log.Logger
	inbox  common.Address
	txMgr  txmgr.TxManager
	l1     client.RPC
	l2     client.RPC
	rollup client.RPC

	// rollupConfigHash and depSetHash are the commitments the real prover would compute from what it
	// derived under. Nothing here derives anything, so they are configuration — and they must be
	// exactly what the verifier is configured with or every batch is refused by acceptance rule 4.
	rollupConfigHash common.Hash
	depSetHash       common.Hash
	// l1Lag is how far below the L1 head a batch's claimed l1Head sits, standing in for the head a
	// real derivation would have run against.
	l1Lag uint64
	// maxBlocks caps one batch's block count.
	maxBlocks uint64
	// wireVersion is the envelope version this harness posts. Explicit rather than inherited from the
	// codec's current version, for the same reason the CLI submitter requires the flag: the version a
	// submitter posts decides whether every verifier CHECKS the chain's declared imports or trusts
	// them, and inheriting it from whichever version the binary was built with is how that changes
	// without anyone deciding it.
	wireVersion uint8

	mu         sync.Mutex
	lastBlock  uint64
	outputRoot common.Hash
	// mutate damages a batch after it is built and before it is checked, staying armed until the
	// callback reports that it APPLIED. It exists for exactly one kind of assertion: a verifier's
	// dependency check is only falsifiable if a test can post a batch whose declared dependency is
	// FALSE. Everything else about the batch stays as the honest builder made it, so the negative case
	// differs from the positive one in one field.
	//
	// "Until applied" rather than "next batch" because the batch a test cares about is the one
	// covering a particular L2 block, and with the cadence ticker running the test does not choose
	// which batch that is.
	mutate func(*proofbatch.ProofBatch) bool
	// proofBytesOnNext fills the next envelope's proof slot. See ProofBytesOnNext.
	proofBytesOnNext []byte
	// posted records every block export this submitter actually put on L1, by block number. It is how
	// a test asserts on THE WIRE — the object that went into the blob — rather than on what it hoped
	// the wire said, and it works whether the batch was posted by SubmitNext or by the ticker.
	posted map[uint64]proofbatch.BlockExport

	tickerCancel context.CancelFunc
	tickerDone   chan struct{}
}

// ProofBatchSubmitterConfig is everything the submitter needs that is not an endpoint.
type ProofBatchSubmitterConfig struct {
	Inbox            common.Address
	SubmitterKey     *ecdsa.PrivateKey
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	// Anchor is where proven history starts: the batch the submitter builds first extends this.
	AnchorBlock      uint64
	AnchorOutputRoot common.Hash
	MaxBlocks        uint64
	L1Lag            uint64
	// WireVersion is the envelope version to post; zero means the codec's current one.
	WireVersion uint8
}

func newProofBatchSubmitter(
	t devtest.T,
	logger log.Logger,
	cfg ProofBatchSubmitterConfig,
	l1UserRPC string,
	l2UserRPC string,
	rollupRPC string,
) *ProofBatchSubmitter {
	require := t.Require()
	require.NotNil(cfg.SubmitterKey, "proof-batch submitter needs an L1 key")
	require.NotEqual(common.Address{}, cfg.Inbox, "proof-batch submitter needs an inbox")
	require.NotEqual(common.Hash{}, cfg.AnchorOutputRoot, "proof-batch submitter needs an anchor output root")

	txCfg := setuputils.NewTxMgrConfig(endpoint.URL(l1UserRPC), cfg.SubmitterKey)
	require.NoError(txCfg.Check(), "invalid proof-batch submitter tx manager config")
	txMgr, err := txmgr.NewSimpleTxManager("proofbatch-submitter", logger, &txmetrics.NoopTxMetrics{}, txCfg)
	require.NoError(err, "failed to build proof-batch submitter tx manager")
	t.Cleanup(txMgr.Close)

	dial := func(name, url string) client.RPC {
		rpcCl, err := client.NewRPC(t.Ctx(), logger, url, client.WithLazyDial())
		require.NoErrorf(err, "failed to dial %s at %s for the proof-batch submitter", name, url)
		t.Cleanup(rpcCl.Close)
		return rpcCl
	}

	maxBlocks := cfg.MaxBlocks
	if maxBlocks == 0 {
		maxBlocks = 300
	}
	wireVersion := cfg.WireVersion
	if wireVersion == 0 {
		wireVersion = proofbatch.Version
	}
	require.NoError(proofbatch.CheckVersion(wireVersion), "proof-batch submitter wire version")
	s := &ProofBatchSubmitter{
		p:                t,
		log:              logger,
		inbox:            cfg.Inbox,
		txMgr:            txMgr,
		l1:               dial("L1", l1UserRPC),
		l2:               dial("L2 EL", l2UserRPC),
		rollup:           dial("rollup node", rollupRPC),
		rollupConfigHash: cfg.RollupConfigHash,
		depSetHash:       cfg.DepSetHash,
		l1Lag:            cfg.L1Lag,
		maxBlocks:        maxBlocks,
		wireVersion:      wireVersion,
		lastBlock:        cfg.AnchorBlock,
		outputRoot:       cfg.AnchorOutputRoot,
	}
	t.Cleanup(s.Stop)
	return s
}

// BatchedHead is the last L2 block this submitter has posted a batch for.
func (s *ProofBatchSubmitter) BatchedHead() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastBlock
}

// UnsafeHead is the chain's own unsafe head: the bound a batch may reach.
func (s *ProofBatchSubmitter) UnsafeHead() uint64 {
	head, err := s.head(s.p.Ctx())
	s.p.Require().NoError(err, "failed to read the silhouette chain's unsafe head")
	return head
}

// Start begins posting batches on a cadence, up to the unsafe head each time. It is the production
// shape; a test that needs to control WHEN a batch lands should leave the ticker alone and call
// SubmitUpTo instead.
func (s *ProofBatchSubmitter) Start(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tickerCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(s.p.Ctx())
	done := make(chan struct{})
	s.tickerCancel, s.tickerDone = cancel, done
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			if _, err := s.submit(ctx, 0); err != nil && ctx.Err() == nil {
				s.log.Warn("proof batch cycle failed, will retry", "err", err)
			}
		}
	}()
}

// Stop halts the cadence, if one was started.
func (s *ProofBatchSubmitter) Stop() {
	s.mu.Lock()
	cancel, done := s.tickerCancel, s.tickerDone
	s.tickerCancel, s.tickerDone = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// MutateUntilApplied arms a mutation that stays armed until the callback returns true.
//
// The mutation runs BEFORE the structural check, deliberately: a test must not be able to post a
// batch the wire codec would refuse, or "the verifier rejected it" would be evidence about the codec
// rather than about the judge.
func (s *ProofBatchSubmitter) MutateUntilApplied(fn func(*proofbatch.ProofBatch) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutate = fn
}

// ProofBytesOnNext makes the next posted envelope carry these bytes in its proof slot.
//
// It exists to make one v1 rule falsifiable end to end: an attested verifier requires proof_len == 0,
// and a batch that arrives carrying proof bytes must be REFUSED rather than accepted with the bytes
// ignored. Nothing here pretends the bytes are a proof — they are not, and the point is that a node
// which cannot check them must not behave as though it had.
//
// One batch rather than a latch, because after a refusal this submitter's cursor has still moved: the
// honest batches behind it chain onto history the verifier never accepted, so arming this for longer
// would test the chaining rule instead.
func (s *ProofBatchSubmitter) ProofBytesOnNext(proof []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proofBytesOnNext = proof
}

// takeProofBytes consumes the armed proof bytes, if any.
func (s *ProofBatchSubmitter) takeProofBytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	proof := s.proofBytesOnNext
	s.proofBytesOnNext = nil
	return proof
}

// PostedExport returns the export this submitter posted for an L2 block, if it has posted one.
func (s *ProofBatchSubmitter) PostedExport(number uint64) (proofbatch.BlockExport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	blk, ok := s.posted[number]
	return blk, ok
}

// WaitBatched blocks until proven history covers `block`, whoever posted it.
func (s *ProofBatchSubmitter) WaitBatched(block uint64, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for s.BatchedHead() < block {
		s.p.Require().False(time.Now().After(deadline),
			"timed out waiting for a batch covering block %d (reached %d)", block, s.BatchedHead())
		time.Sleep(500 * time.Millisecond) // nosemgrep: flake-sleep-in-test -- waiting on the proof cadence, which is the thing under test
	}
}

// SubmitNext posts one batch covering everything up to the chain's unsafe head, and returns it. It
// returns nil when the chain has produced nothing new.
func (s *ProofBatchSubmitter) SubmitNext() *proofbatch.ProofBatch {
	batch, err := s.submit(s.p.Ctx(), 0)
	s.p.Require().NoError(err, "failed to submit a proof batch")
	return batch
}

// SubmitUpTo posts batches until proven history reaches lastBlock, waiting for the chain to produce
// the blocks if it has not yet. It is the control a test uses to decide exactly which of P's blocks
// are provable at a given moment.
func (s *ProofBatchSubmitter) SubmitUpTo(lastBlock uint64) {
	require := s.p.Require()
	deadline := time.Now().Add(5 * time.Minute)
	for s.BatchedHead() < lastBlock {
		require.False(time.Now().After(deadline),
			"timed out batching the silhouette chain up to block %d (reached %d)", lastBlock, s.BatchedHead())
		batch, err := s.submit(s.p.Ctx(), lastBlock)
		require.NoError(err, "failed to submit a proof batch up to block %d", lastBlock)
		if batch == nil {
			// The chain has not produced the blocks yet. Its sequencer is live, so this is a wait
			// rather than a stall.
			time.Sleep(500 * time.Millisecond) // nosemgrep: flake-sleep-in-test -- waiting on the private sequencer to produce blocks the batch needs
		}
	}
}

// submit builds and posts the batch that extends proven history, bounded by upTo when non-zero and
// by the chain's unsafe head always. A nil batch with a nil error means there was nothing to post.
func (s *ProofBatchSubmitter) submit(ctx context.Context, upTo uint64) (*proofbatch.ProofBatch, error) {
	batch, err := s.build(ctx, upTo)
	if err != nil || batch == nil {
		return nil, err
	}
	// The proof slot is EMPTY in attested mode, which is v1: the operator's signature on the
	// transaction below is what stands behind this batch. A test can arm bytes here to assert that a
	// verifier refuses a proof it cannot check.
	proof := s.takeProofBytes()
	payload, err := proofbatch.EncodeAs(batch, proof, s.wireVersion)
	if err != nil {
		return nil, fmt.Errorf("encode envelope at wire version %d: %w", s.wireVersion, err)
	}
	blobs, err := proofbatch.ToBlobs(payload)
	if err != nil {
		return nil, fmt.Errorf("pack blobs: %w", err)
	}
	last := batch.Blocks[len(batch.Blocks)-1]
	s.log.Info("submitting proof batch", "inbox", s.inbox, "wire_version", s.wireVersion,
		"bytes", len(payload), "blobs", len(blobs),
		"blocks", len(batch.Blocks), "first", batch.Blocks[0].Number, "last", last.Number,
		"output_root", batch.NewOutputRoot)
	receipt, err := s.txMgr.Send(ctx, txmgr.TxCandidate{To: &s.inbox, Blobs: blobs})
	if err != nil {
		return nil, fmt.Errorf("submit proof batch: %w", err)
	}
	s.log.Info("proof batch landed", "tx", receipt.TxHash, "l1_block", receipt.BlockNumber,
		"first", batch.Blocks[0].Number, "last", last.Number)

	// The cursor moves only after the batch is on L1, so a send failure re-proposes the same range
	// rather than skipping it — the verifier's chaining rule would reject everything after a gap.
	s.mu.Lock()
	s.lastBlock, s.outputRoot = last.Number, batch.NewOutputRoot
	if s.posted == nil {
		s.posted = make(map[uint64]proofbatch.BlockExport)
	}
	for _, blk := range batch.Blocks {
		s.posted[blk.Number] = blk
	}
	s.mu.Unlock()
	return batch, nil
}

func (s *ProofBatchSubmitter) build(ctx context.Context, upTo uint64) (*proofbatch.ProofBatch, error) {
	head, err := s.head(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	from, prevRoot := s.lastBlock, s.outputRoot
	s.mu.Unlock()

	last := head
	if upTo != 0 && upTo < last {
		last = upTo
	}
	if last <= from {
		return nil, nil
	}
	if last-from > s.maxBlocks {
		last = from + s.maxBlocks
	}

	l1Head, err := s.l1Head(ctx)
	if err != nil {
		return nil, err
	}
	batch := &proofbatch.ProofBatch{
		PrevOutputRoot:   prevRoot,
		L1Head:           l1Head,
		RollupConfigHash: s.rollupConfigHash,
		DepSetHash:       s.depSetHash,
		ExportPolicyHash: proofbatch.ExportPolicyAllHashes,
	}
	for n := from + 1; n <= last; n++ {
		export, err := s.blockExport(ctx, n)
		if err != nil {
			return nil, err
		}
		batch.Blocks = append(batch.Blocks, *export)
	}
	// DERIVED from the last block's committed roots rather than asked for separately: that is what a
	// v2 verifier checks, so building it any other way would produce batches only a lenient verifier
	// accepts.
	batch.NewOutputRoot = batch.Blocks[len(batch.Blocks)-1].OutputRoot()
	// Taken OUT under the lock, not merely read under it. Both SubmitNext/SubmitUpTo and the cadence
	// ticker call build(), so a read-invoke-relock sequence lets two concurrent builds each observe the
	// same non-nil hook and apply it twice — "until applied" applying to two batches. Whoever takes it
	// owns it, and puts it back if it did not fire.
	s.mu.Lock()
	mutate := s.mutate
	s.mutate = nil
	s.mu.Unlock()
	if mutate != nil && !mutate(batch) {
		s.mu.Lock()
		// Only re-arm if nothing else claimed it in the meantime.
		if s.mutate == nil {
			s.mutate = mutate
		}
		s.mu.Unlock()
	}
	if err := batch.CheckStructure(); err != nil {
		return nil, fmt.Errorf("built an invalid batch: %w", err)
	}
	// The verifier's acceptance rule, checked HERE rather than discovered on the far side: a real
	// prover refuses to prove a same-timestamp import in-circuit (G7R D10), so a harness that could
	// post one would be manufacturing a batch no prover can produce.
	if err := batch.CheckNoSameTimestampImports(); err != nil {
		return nil, fmt.Errorf("built a batch a verifier will refuse: %w", err)
	}
	return batch, nil
}

// head reads the chain's UNSAFE head. Both heads come out of the one response so the gap between
// them can be logged: on a healthy silhouette chain that gap IS the proof lag.
func (s *ProofBatchSubmitter) head(ctx context.Context) (uint64, error) {
	var status struct {
		UnsafeL2 struct {
			Number uint64 `json:"number"`
		} `json:"unsafe_l2"`
		SafeL2 struct {
			Number uint64 `json:"number"`
		} `json:"safe_l2"`
	}
	if err := s.rollup.CallContext(ctx, &status, "optimism_syncStatus"); err != nil {
		return 0, fmt.Errorf("fetch sync status: %w", err)
	}
	if status.UnsafeL2.Number < status.SafeL2.Number {
		return 0, fmt.Errorf("unsafe head %d is below the safe head %d",
			status.UnsafeL2.Number, status.SafeL2.Number)
	}
	return status.UnsafeL2.Number, nil
}

// silhouetteRPCOutput is the part of optimism_outputAtBlock a v2 block export needs: the block's
// identity, its timestamp, and the two roots its output root derives from.
type silhouetteRPCOutput struct {
	OutputRoot            common.Hash `json:"outputRoot"`
	StateRoot             common.Hash `json:"stateRoot"`
	WithdrawalStorageRoot common.Hash `json:"withdrawalStorageRoot"`
	BlockRef              struct {
		Hash common.Hash `json:"hash"`
		Time uint64      `json:"timestamp"`
	} `json:"blockRef"`
}

func (s *ProofBatchSubmitter) outputAtBlock(ctx context.Context, block uint64) (*silhouetteRPCOutput, error) {
	var out silhouetteRPCOutput
	if err := s.rollup.CallContext(ctx, &out, "optimism_outputAtBlock", hexutil.Uint64(block)); err != nil {
		return nil, fmt.Errorf("fetch output at %d: %w", block, err)
	}
	return &out, nil
}

// blockExport reads one L2 block's committed facts and its logs, hashing every log the way the
// interop LogsDB does. Log indices come from the node's own numbering, because that index is what an
// executing message references.
func (s *ProofBatchSubmitter) blockExport(ctx context.Context, number uint64) (*proofbatch.BlockExport, error) {
	out, err := s.outputAtBlock(ctx, number)
	if err != nil {
		return nil, err
	}
	var receipts []silhouetteRPCReceipt
	if err := s.l2.CallContext(ctx, &receipts, "eth_getBlockReceipts", hexutil.Uint64(number)); err != nil {
		return nil, fmt.Errorf("fetch receipts of L2 block %d: %w", number, err)
	}
	export := &proofbatch.BlockExport{
		Number:                   number,
		Timestamp:                out.BlockRef.Time,
		Hash:                     out.BlockRef.Hash,
		StateRoot:                out.StateRoot,
		MessagePasserStorageRoot: out.WithdrawalStorageRoot,
	}
	// The node published this block's output root itself, so deriving it from the three roots this
	// batch commits to is a free check that the export is internally consistent — and it is exactly
	// the check the verifier will run.
	if derived := export.OutputRoot(); derived != out.OutputRoot {
		return nil, fmt.Errorf("block %d: derived output root %s but the node reports %s",
			number, derived, out.OutputRoot)
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			log := &types.Log{Address: l.Address, Topics: l.Topics, Data: l.Data}
			logs = append(logs, log)
			export.Logs = append(export.Logs, proofbatch.LogExport{
				Index: uint32(l.LogIndex),
				Hash:  messages.LogToLogHash(log),
				// The v2 default policy exports hashes only; a preimage-bearing policy is a config
				// change on both sides, not a change to this code's wire.
			})
		}
	}
	// THE IMPORT LIST (wire v3), through the SHARED canonical extraction rather than a second copy of
	// it. A real prover does this in-circuit from the same receipts; this harness must not be able to
	// produce an import list the node would not — filter, order, dedup and the abort-on-malformed rule
	// all live in one place (proofbatch.ExecMsgsFromLogs).
	imports, err := proofbatch.ExecMsgsFromLogs(logs)
	if err != nil {
		return nil, fmt.Errorf("extract the import list of L2 block %d: %w", number, err)
	}
	export.ExecMsgs = imports
	return export, nil
}

// silhouetteRPCReceipt / silhouetteRPCLog read only the fields a log export needs: the three a log
// hash covers, plus the block-level index the node assigns.
type silhouetteRPCReceipt struct {
	Logs []silhouetteRPCLog `json:"logs"`
}

type silhouetteRPCLog struct {
	Address  common.Address `json:"address"`
	Topics   []common.Hash  `json:"topics"`
	Data     hexutil.Bytes  `json:"data"`
	LogIndex hexutil.Uint64 `json:"logIndex"`
}

// l1Head picks the L1 block a batch claims to have been derived against: the L1 head minus the
// configured lag, so the batch stays clear of a reorg the verifier has not seen.
func (s *ProofBatchSubmitter) l1Head(ctx context.Context) (common.Hash, error) {
	var head hexutil.Uint64
	if err := s.l1.CallContext(ctx, &head, "eth_blockNumber"); err != nil {
		return common.Hash{}, fmt.Errorf("fetch L1 head: %w", err)
	}
	if uint64(head) < s.l1Lag {
		return common.Hash{}, fmt.Errorf("L1 head %d is below the configured lag %d", head, s.l1Lag)
	}
	number := uint64(head) - s.l1Lag

	var header struct {
		Hash common.Hash `json:"hash"`
	}
	if err := s.l1.CallContext(ctx, &header, "eth_getBlockByNumber", hexutil.Uint64(number), false); err != nil {
		return common.Hash{}, fmt.Errorf("fetch L1 block %d: %w", number, err)
	}
	return header.Hash, nil
}
