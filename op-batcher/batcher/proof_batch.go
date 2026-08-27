package batcher

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

var errProofBatchFull = errors.New("silhouette proof batch reached its block limit")

// ProofBatchConfig changes only the normal batcher's terminal representation and destination.
// Block loading, reorg handling, channel retry, txmgr, receipts, and blob submission remain the
// stock batcher path.
type ProofBatchConfig struct {
	Inbox            common.Address
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	WireVersion      uint8
	MaxBlocks        uint64
	// TestHooks is used by in-process acceptance tests to observe or deliberately damage the
	// terminal representation. It never changes block loading, channel management, transaction
	// submission, or any other part of the normal batcher path.
	TestHooks *ProofBatchTestHooks
}

func (c ProofBatchConfig) Check() error {
	if c.Inbox == (common.Address{}) {
		return errors.New("silhouette proof-batch inbox is required")
	}
	if c.RollupConfigHash == (common.Hash{}) || c.DepSetHash == (common.Hash{}) {
		return errors.New("silhouette rollup-config and dependency-set hashes are required")
	}
	if err := proofbatch.CheckVersion(c.WireVersion); err != nil {
		return fmt.Errorf("silhouette wire version: %w", err)
	}
	if c.MaxBlocks == 0 {
		return errors.New("silhouette maximum blocks per proof batch must be non-zero")
	}
	return nil
}

type preparedProofBlock struct {
	export           proofbatch.BlockExport
	parentOutputRoot common.Hash
}

// ProofBatchEncoder is shared by the loader-side receipt/output enrichment and channel-side wire
// encoder. The map bridges those two existing batcher stages without teaching the generic
// derive.ChannelOut interface about receipts.
type ProofBatchEncoder struct {
	cfg ProofBatchConfig

	mu     sync.RWMutex
	blocks map[common.Hash]preparedProofBlock
}

// ProofBatchEnvelope is one proof batch as it entered the normal batcher's final wire encoder.
// Proof is normally empty; acceptance tests use it to exercise verifier refusal rules.
type ProofBatchEnvelope struct {
	Batch proofbatch.ProofBatch
	Proof []byte
}

// ProofBatchTestHooks provides observation and fault injection at the final encoding seam. It is
// intentionally passive: op-batcher remains the only component that loads, batches, and submits.
type ProofBatchTestHooks struct {
	mu         sync.Mutex
	mutate     func(*proofbatch.ProofBatch) bool
	proofNext  []byte
	proofAfter uint64
	pending    map[derive.ChannelID]pendingProofBatchEnvelope
	envelopes  []ProofBatchEnvelope
	changed    chan struct{}
}

type pendingProofBatchEnvelope struct {
	envelope ProofBatchEnvelope
	mutated  bool
}

func NewProofBatchTestHooks() *ProofBatchTestHooks {
	return &ProofBatchTestHooks{
		pending: make(map[derive.ChannelID]pendingProofBatchEnvelope),
		changed: make(chan struct{}, 1),
	}
}

func (h *ProofBatchTestHooks) MutateUntilApplied(fn func(*proofbatch.ProofBatch) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mutate = fn
}

func (h *ProofBatchTestHooks) ProofBytesOnNext(proof []byte) {
	h.ProofBytesAfter(proof, 0)
}

// ProofBytesAfter injects proof bytes into the next batch whose range advances beyond block.
// This lets integration tests ignore a normal batcher's already-buffered safe-head retry.
func (h *ProofBatchTestHooks) ProofBytesAfter(proof []byte, block uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.proofNext = append([]byte(nil), proof...)
	h.proofAfter = block
}

func (h *ProofBatchTestHooks) beforeEncode(batch *proofbatch.ProofBatch) ([]byte, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	mutated := false
	if h.mutate != nil {
		mutated = h.mutate(batch)
	}
	var proof []byte
	if len(batch.Blocks) > 0 && batch.Blocks[len(batch.Blocks)-1].Number > h.proofAfter {
		proof = append([]byte(nil), h.proofNext...)
	}
	return proof, mutated
}

func cloneProofBatch(in *proofbatch.ProofBatch) proofbatch.ProofBatch {
	out := *in
	out.Blocks = append([]proofbatch.BlockExport(nil), in.Blocks...)
	for i := range out.Blocks {
		out.Blocks[i].Logs = append([]proofbatch.LogExport(nil), in.Blocks[i].Logs...)
		out.Blocks[i].ExecMsgs = append([]proofbatch.ExecMsg(nil), in.Blocks[i].ExecMsgs...)
	}
	return out
}

func (h *ProofBatchTestHooks) prepared(id derive.ChannelID, batch *proofbatch.ProofBatch, proof []byte, mutated bool) {
	h.mu.Lock()
	h.pending[id] = pendingProofBatchEnvelope{
		envelope: ProofBatchEnvelope{
			Batch: cloneProofBatch(batch),
			Proof: append([]byte(nil), proof...),
		},
		mutated: mutated,
	}
	h.mu.Unlock()
}

// RecordProofBatchSubmission moves terminal encodings into the observable history only when the
// normal batcher queues their transaction with txmgr. Encoding a channel is not enough: the normal
// channel manager may discard and rebuild it during safe-head reconciliation.
func (h *ProofBatchTestHooks) RecordProofBatchSubmission(ids []derive.ChannelID) {
	h.mu.Lock()
	changed := false
	for _, id := range ids {
		pending, ok := h.pending[id]
		if !ok {
			continue
		}
		h.envelopes = append(h.envelopes, pending.envelope)
		delete(h.pending, id)
		if pending.mutated {
			h.mutate = nil
		}
		if len(pending.envelope.Proof) > 0 {
			h.proofNext = nil
			h.proofAfter = 0
		}
		changed = true
	}
	h.mu.Unlock()
	if !changed {
		return
	}
	select {
	case h.changed <- struct{}{}:
	default:
	}
}

func (h *ProofBatchTestHooks) Envelopes() []ProofBatchEnvelope {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]ProofBatchEnvelope, len(h.envelopes))
	copy(out, h.envelopes)
	return out
}

func (h *ProofBatchTestHooks) Changed() <-chan struct{} { return h.changed }

func NewProofBatchEncoder(cfg ProofBatchConfig) (*ProofBatchEncoder, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return &ProofBatchEncoder{cfg: cfg, blocks: make(map[common.Hash]preparedProofBlock)}, nil
}

// PrepareBlock enriches the payload already fetched by the normal batcher with precisely the facts
// omitted from a transaction-stripped batch: output roots and receipt logs/imports.
func (e *ProofBatchEncoder) PrepareBlock(ctx context.Context, payloads dial.PayloadSource, rollupClient dial.RollupClientInterface, payload *eth.ExecutionPayload) error {
	if err := e.prepareBlock(ctx, payloads, rollupClient, payload); err != nil {
		return err
	}
	// A verifier-generated Holocene replacement is already local-safe when the private LightCL
	// follows it, so the stock batcher quite correctly starts loading at the following block. Keep
	// the immediate parent available to the terminal encoder as a one-block overlap: this lets the
	// replacement itself become proof-committed without changing stock sync/reorg bookkeeping.
	number := uint64(payload.BlockNumber)
	if number <= 1 {
		return nil
	}
	parent, err := payloads.PayloadByNumber(ctx, number-1)
	if err != nil {
		return fmt.Errorf("fetch proof-batch overlap block %d: %w", number-1, err)
	}
	if parent == nil || parent.ExecutionPayload == nil {
		return fmt.Errorf("proof-batch overlap block %d is empty", number-1)
	}
	if parent.ExecutionPayload.BlockHash != payload.ParentHash {
		return fmt.Errorf("proof-batch block %d parent is %s, canonical block %d is %s",
			number, payload.ParentHash, number-1, parent.ExecutionPayload.BlockHash)
	}
	return e.prepareBlock(ctx, payloads, rollupClient, parent.ExecutionPayload)
}

func (e *ProofBatchEncoder) prepareBlock(ctx context.Context, payloads dial.PayloadSource, rollupClient dial.RollupClientInterface, payload *eth.ExecutionPayload) error {
	if payload == nil {
		return errors.New("cannot prepare a nil proof-batch payload")
	}
	if _, ok := e.prepared(payload.BlockHash); ok {
		return nil
	}
	number := uint64(payload.BlockNumber)
	if number == 0 {
		return errors.New("genesis is an anchor, not a proof-batch block")
	}
	output, err := rollupClient.OutputAtBlock(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch output at block %d: %w", number, err)
	}
	parentOutput, err := rollupClient.OutputAtBlock(ctx, number-1)
	if err != nil {
		return fmt.Errorf("fetch parent output at block %d: %w", number-1, err)
	}
	_, receipts, err := payloads.FetchReceipts(ctx, payload.BlockHash)
	if err != nil {
		return fmt.Errorf("fetch receipts of block %d: %w", number, err)
	}
	export, err := proofbatch.ExportBlock(payload, output, receipts)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.blocks[payload.BlockHash] = preparedProofBlock{export: export, parentOutputRoot: common.Hash(parentOutput.OutputRoot)}
	e.mu.Unlock()
	return nil
}

func (e *ProofBatchEncoder) ChannelOut(channelCfg ChannelConfig, rollupCfg *rollup.Config) (derive.ChannelOut, error) {
	return newProofChannelOut(e, channelCfg, rollupCfg)
}

func (e *ProofBatchEncoder) prepared(hash common.Hash) (preparedProofBlock, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	b, ok := e.blocks[hash]
	return b, ok
}

func (e *ProofBatchEncoder) RecordProofBatchSubmission(ids []derive.ChannelID) {
	if e.cfg.TestHooks != nil {
		e.cfg.TestHooks.RecordProofBatchSubmission(ids)
	}
}

// ConfigureProofBatches applies the proof encoder to a normal DriverSetup.
func ConfigureProofBatches(setup *DriverSetup, cfg ProofBatchConfig) error {
	encoder, err := NewProofBatchEncoder(cfg)
	if err != nil {
		return err
	}
	setup.BlockEnricher = encoder
	setup.ChannelOutFactory = encoder.ChannelOut
	setup.SubmissionInbox = &encoder.cfg.Inbox
	return nil
}

type proofChannelOut struct {
	encoder *ProofBatchEncoder
	rollup  *rollup.Config
	id      derive.ChannelID
	carrier derive.ChannelOut

	blocks   []proofbatch.BlockExport
	prevRoot common.Hash
	l1Head   common.Hash
	input    int
	fullErr  error
	closed   bool
	payload  []byte
	offset   int
	frame    uint16
	lastRaw  bool
}

func newProofChannelOut(encoder *ProofBatchEncoder, channelCfg ChannelConfig, rollupCfg *rollup.Config) (*proofChannelOut, error) {
	// The proof envelope is followed by an ordinary derivation channel containing the same blocks
	// with every batch-submitted transaction removed. A stock op-node ignores the KCPB blobs as an
	// unknown derivation format and consumes these carrier frames normally.
	carrierCfg := channelCfg
	carrierCfg.BatchType = derive.SingularBatchType
	carrier, err := NewChannelOut(carrierCfg, rollupCfg)
	if err != nil {
		return nil, fmt.Errorf("create empty-batch carrier: %w", err)
	}
	c := &proofChannelOut{encoder: encoder, rollup: rollupCfg, carrier: carrier}
	if _, err := rand.Read(c.id[:]); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *proofChannelOut) ID() derive.ChannelID { return c.id }

func (c *proofChannelOut) Reset() error {
	if err := c.carrier.Reset(); err != nil {
		return err
	}
	*c = proofChannelOut{encoder: c.encoder, rollup: c.rollup, carrier: c.carrier}
	_, err := rand.Read(c.id[:])
	return err
}

func (c *proofChannelOut) AddBlock(cfg *rollup.Config, payload *eth.ExecutionPayload) (*derive.L1BlockInfo, error) {
	if c.closed {
		return nil, derive.ErrChannelOutAlreadyClosed
	}
	prepared, ok := c.encoder.prepared(payload.BlockHash)
	if !ok {
		return nil, fmt.Errorf("block %s was not enriched for proof-batch encoding", payload.ID())
	}
	_, l1Info, err := derive.PayloadToSingularBatch(cfg, payload)
	if err != nil {
		return nil, fmt.Errorf("read L1 origin from block %s: %w", payload.ID(), err)
	}
	prepared.export.L1Origin = eth.BlockID{Hash: l1Info.BlockHash, Number: l1Info.Number}
	prepared.export.SequenceNumber = l1Info.SequenceNumber
	c.encoder.mu.Lock()
	c.encoder.blocks[payload.BlockHash] = prepared
	c.encoder.mu.Unlock()
	if len(c.blocks) == 0 {
		if prepared.export.Number > 1 {
			parent, ok := c.encoder.prepared(payload.ParentHash)
			if !ok {
				return nil, fmt.Errorf("proof-batch overlap parent %s was not enriched", payload.ParentID())
			}
			if parent.export.Number+1 != prepared.export.Number || common.Hash(parent.export.OutputRoot()) != prepared.parentOutputRoot {
				return nil, fmt.Errorf("proof-batch overlap block %d does not extend into block %d",
					parent.export.Number, prepared.export.Number)
			}
			c.prevRoot = parent.parentOutputRoot
			c.blocks = append(c.blocks, parent.export)
			c.input += proofBlockInputSize(parent.export)
		} else {
			c.prevRoot = prepared.parentOutputRoot
		}
	} else {
		prev := c.blocks[len(c.blocks)-1]
		if prepared.export.Number != prev.Number+1 || prepared.parentOutputRoot != common.Hash(prev.OutputRoot()) {
			return nil, fmt.Errorf("proof block %d does not extend proof block %d", prepared.export.Number, prev.Number)
		}
	}
	c.blocks = append(c.blocks, prepared.export)
	// Preserve only deposits long enough for PayloadToSingularBatch to read the mandatory L1-info
	// transaction. It excludes all deposits from the encoded batch, while the verifier's stock
	// attributes builder derives real portal deposits from L1 in the usual way.
	carrierPayload := *payload
	carrierPayload.Transactions = make([]eth.Data, 0, len(payload.Transactions))
	for _, tx := range payload.Transactions {
		if len(tx) > 0 && tx[0] == optypes.DepositTxType {
			carrierPayload.Transactions = append(carrierPayload.Transactions, tx)
		}
	}
	if _, err := c.carrier.AddBlock(cfg, &carrierPayload); err != nil {
		return nil, fmt.Errorf("add block %s to empty-batch carrier: %w", payload.ID(), err)
	}
	c.l1Head = l1Info.BlockHash
	c.input += proofBlockInputSize(prepared.export)
	if uint64(len(c.blocks)) >= c.encoder.cfg.MaxBlocks {
		c.fullErr = errProofBatchFull
	}
	return l1Info, nil
}

func proofBlockInputSize(block proofbatch.BlockExport) int {
	return 256 + len(block.Logs)*36 + len(block.ExecMsgs)*192
}

func (c *proofChannelOut) InputBytes() int { return c.input }
func (c *proofChannelOut) ReadyBytes() int {
	if !c.closed {
		return 0
	}
	return len(c.payload) - c.offset + c.carrier.ReadyBytes()
}
func (c *proofChannelOut) Flush() error { return c.carrier.Flush() }
func (c *proofChannelOut) FullErr() error {
	if c.fullErr != nil {
		return c.fullErr
	}
	return c.carrier.FullErr()
}

func (c *proofChannelOut) Close() error {
	if c.closed {
		return derive.ErrChannelOutAlreadyClosed
	}
	if len(c.blocks) == 0 {
		return errors.New("cannot close an empty proof batch")
	}
	if err := c.carrier.Close(); err != nil {
		return fmt.Errorf("close empty-batch carrier: %w", err)
	}
	batch := &proofbatch.ProofBatch{
		PrevOutputRoot:   c.prevRoot,
		NewOutputRoot:    common.Hash(c.blocks[len(c.blocks)-1].OutputRoot()),
		L1Head:           c.l1Head,
		RollupConfigHash: c.encoder.cfg.RollupConfigHash,
		DepSetHash:       c.encoder.cfg.DepSetHash,
		ExportPolicyHash: proofbatch.ExportPolicyAllHashes,
		Blocks:           append([]proofbatch.BlockExport(nil), c.blocks...),
	}
	var proof []byte
	var mutated bool
	if c.encoder.cfg.TestHooks != nil {
		proof, mutated = c.encoder.cfg.TestHooks.beforeEncode(batch)
	}
	if err := batch.CheckStructure(); err != nil {
		return fmt.Errorf("built invalid proof batch: %w", err)
	}
	if err := batch.CheckNoSameTimestampImports(); err != nil {
		return fmt.Errorf("built proof batch every verifier would reject: %w", err)
	}
	payload, err := proofbatch.EncodeAs(batch, proof, c.encoder.cfg.WireVersion)
	if err != nil {
		return err
	}
	maxPayload := eth.MaxBlobDataSize * maxBlobsPerBlock
	if len(payload) > maxPayload {
		return fmt.Errorf("proof batch is %d bytes, exceeds one transaction's %d-byte blob capacity", len(payload), maxPayload)
	}
	if c.encoder.cfg.TestHooks != nil {
		c.encoder.cfg.TestHooks.prepared(c.id, batch, proof, mutated)
	}
	c.payload = payload
	c.closed = true
	return nil
}

func (c *proofChannelOut) OutputFrame(out *bytes.Buffer, maxSize uint64) (uint16, error) {
	if !c.closed {
		return c.frame, io.EOF
	}
	frame := c.frame
	if c.offset < len(c.payload) {
		end := min(c.offset+eth.MaxBlobDataSize, len(c.payload))
		_, _ = out.Write(c.payload[c.offset:end])
		c.frame++
		c.offset = end
		c.lastRaw = true
		// Even when this is the final KCPB blob, the stock carrier frames still follow.
		return frame, nil
	}
	_, err := c.carrier.OutputFrame(out, maxSize)
	c.frame++
	c.lastRaw = false
	return frame, err
}

func (c *proofChannelOut) DiscardCompressor() {}

// RawFrames is true only for KCPB chunks. The following empty-batch carrier frames receive the
// normal derivation-version prefix and are consumed by an unmodified op-node.
func (c *proofChannelOut) RawFrames() bool { return c.lastRaw }
