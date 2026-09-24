package batcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Private Interop's terminal seam.
//
// Private payload loading, channel encoding, and the standard L1 transport are reused.
// The publication cursor skips positions already derived by the public projection,
// including fallback blocks after an outage, without promoting private safety.
// BlockEnricher fetches the receipts needed by the terminal encoder.
// ChannelOutFactory renders those blocks into ordinary sequencer batches and frames.

// BlockEnricher fetches, for each loaded L2 block, the side data an alternate terminal encoding
// needs and that an execution payload does not carry.
//
// The default batcher leaves it nil. It is deliberately narrow — it is given a payload and returns
// an error — because a hook in the block-loading path that can do more than "fetch and remember"
// is a hook that can change what blocks the batcher believes exist.
type BlockEnricher interface {
	PrepareBlock(ctx context.Context, payload *eth.ExecutionPayload) error
}

// PrivateReceipts fetches a private block's receipts. *sources.EthClient satisfies it, which is
// what the operator points at its own private EL.
type PrivateReceipts interface {
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, optypes.Receipts, error)
}

// RangeStart is everything about the PREVIOUS range that the next one continues from.
//
// All four are execution-derived facts about a chain the builder is writing the INPUT for, so none
// of them can be computed here. They come from a node following the rendering.
type RangeStart struct {
	Continuation projection.Continuation
	// PrevTerminalRenderingHash is the previous range's terminal rendering block hash: the span's
	// 20-byte parent check, and the channel ID's seed.
	PrevTerminalRenderingHash common.Hash
	// StartNonce is the standard batcher account's nonce for the range's first transaction.
	StartNonce uint64
}

// RangeSource supplies the asynchronous input a range needs.
//
// It is an interface because its one method is a WAIT on something outside this process — a node
// that has to have derived and executed the predecessor — and getting a wait wrong is a stalled
// chain rather than a wrong one. See privateInteropRangeSource for the production implementation.
//
// There is exactly ONE publication per cadence: the public batch transaction. Nothing else the
// operator does is a range input — the private chain's own safety comes from the leading claim
// inside that same batch, and the range's private derivation input is not published at all (it is
// committed to by hash; the bytes travel the operator's firewalled p2p network).
type RangeSource interface {
	// RangeStart returns the state the range beginning at firstBlock continues from. It fails, and
	// the batcher retries, while the predecessor is not yet derived: guessing would post a batch
	// every verifier drops.
	RangeStart(ctx context.Context, firstBlock uint64) (RangeStart, error)
}

// PrivateInteropTestHooks are programmatic test hooks for the terminal seam. They are never set by
// a CLI flag or by production code. The seam reads them when it builds each range, so a test may
// change or clear them while the batcher runs.
type PrivateInteropTestHooks struct {
	// SkipCarrierPreRun disables the carrier pre-run (static gas check and eth_call simulation),
	// so a test can publish a span whose carrier fails on the projection.
	SkipCarrierPreRun bool
	// GasPolicyOverride replaces the transaction builder's gas policy for the ranges built while
	// it is set.
	GasPolicyOverride *render.GasPolicy
	// MutateProof is applied to the producer's envelope after the producer verified it.
	MutateProof func([]byte) []byte
	// SkipAdmissionPreflight skips the post-proof admission preflight
	// (builder.Range.TestSkipAdmission).
	SkipAdmissionPreflight bool
	// ReplaceProof, when set and returning ok, replaces the producer for that range: the range is
	// published with the returned bytes as its proof, without running the proof command. Together
	// with SkipCarrierPreRun and SkipAdmissionPreflight it publishes a span the batcher itself
	// would refuse (for example an under-gassed carrier, which the producer's own admission
	// preflight rejects), so a test can check that derivation's admission drops it.
	ReplaceProof func(*builder.BuiltRange) (proof []byte, ok bool)
}

// PrivateInteropConfig configures the terminal seam.
type PrivateInteropConfig struct {
	// Prove produces the range's proof envelope before any frames may be emitted. Publication
	// blocks until it succeeds: there is no fallback verifier and no unproven publication. Every
	// verifier except insecure-stub-v1 requires it.
	Prove func(context.Context, *builder.BuiltRange, []byte, RangeStart) ([]byte, error)
	// Caller simulates carrier transactions on the public projection before proving (§E.3). It
	// is required whenever Prove is set, unless the pre-run is disabled by a test hook.
	Caller ProjectionCaller
	// TestHooks are programmatic test hooks; nil in production.
	TestHooks *PrivateInteropTestHooks
	// Rollup is the RENDERING's rollup config. It is NOT the private chain's: the timestamps and
	// numbers coincide block-for-block, but the genesis, chain ID and drift the span batch is
	// encoded against belong to the chain being described.
	Rollup *rollup.Config
	// PrivateRollup is the PRIVATE chain's rollup config — the batcher's own, from --rollup-rpc. It
	// is needed for exactly one thing: encoding the range's private derivation input, which is a
	// description of the private chain and must be encoded against the private chain.
	PrivateRollup *rollup.Config
	// Batcher is the account signing the projection transactions.
	Batcher common.Address
	// Emitters is the rendering's emitter set.
	Emitters render.EmitterSet
	// MaxBlocksPerRange is the cadence — ~300 blocks at 2 s is one span batch every ten minutes.
	MaxBlocksPerRange uint64
	// MaxRangeBytes closes a range once its conservative uncompressed rendering-size estimate
	// reaches the producer budget.
	MaxRangeBytes uint64
	// RollupConfigHash and DepSetHash are the claim's two configuration commitments: which chain
	// and which dependency set the claim speaks for. They are frozen configuration, identical for
	// every range, which is why they live here rather than being fetched per range.
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	// Receipts is the private EL's receipt source.
	Receipts PrivateReceipts
	// Ranges supplies the previous range's terminal state.
	Ranges RangeSource
	// Txs builds the standard batcher's signed replay and claim transactions.
	Txs render.ReplayTxBuilder
	// MaxFrameSize caps a frame; zero takes the builder's blob-sized default.
	MaxFrameSize uint64
}

func (c *PrivateInteropConfig) Check() error {
	if c.Rollup == nil {
		return errors.New("private interop: no rendering rollup config")
	}
	if c.PrivateRollup == nil {
		return errors.New("private interop: no private rollup config")
	}
	if pp := c.Rollup.PrivateProjection; pp != nil && pp.Verifier != projection.InsecureStub && c.Prove == nil {
		return fmt.Errorf("private interop: %s admission requires a proof producer (--private-interop.proof-command)", pp.Verifier)
	}
	if c.Prove != nil && c.Caller == nil && (c.TestHooks == nil || !c.TestHooks.SkipCarrierPreRun) {
		return errors.New("private interop: the carrier pre-run needs a public-projection execution client")
	}
	if c.MaxBlocksPerRange == 0 {
		return errors.New("private interop: no cadence configured")
	}
	if c.MaxRangeBytes == 0 {
		return errors.New("private interop: no range byte budget configured")
	}
	if c.RollupConfigHash == (common.Hash{}) {
		return errors.New("private interop: no rollup config hash for the range claim")
	}
	if c.DepSetHash == (common.Hash{}) {
		return errors.New("private interop: no dependency set hash for the range claim")
	}
	if c.Receipts == nil {
		return errors.New("private interop: no private receipt source")
	}
	if c.Ranges == nil {
		return errors.New("private interop: no range source")
	}
	if c.Txs == nil {
		return errors.New("private interop: no replay transaction builder")
	}
	if c.Batcher == (common.Address{}) {
		return errors.New("private interop: no batcher signer address")
	}
	if _, err := renderingBlockGasBudget(c.Rollup); err != nil {
		return err
	}
	return nil
}

// renderingBlockGasBudget reserves half of the EIP-1559 target for the mandatory attributes
// deposit and any protocol upgrade transactions. Synthetic claim/replay transactions must fit in
// the other half. Since actual gas used cannot exceed declared gas, staying within this budget
// keeps a zero base fee at zero.
func renderingBlockGasBudget(cfg *rollup.Config) (uint64, error) {
	if cfg == nil || cfg.ChainOpConfig == nil || cfg.ChainOpConfig.EIP1559Elasticity == 0 {
		return 0, errors.New("private interop: rendering rollup config has no EIP-1559 elasticity")
	}
	target := cfg.Genesis.SystemConfig.GasLimit / cfg.ChainOpConfig.EIP1559Elasticity
	if target < 2 {
		return 0, errors.New("private interop: rendering EIP-1559 gas target is too small")
	}
	return target / 2, nil
}

// PrivateInteropEncoder is the terminal stage: it remembers each loaded private block's receipts,
// and hands the channel manager a ChannelOut that renders them.
type PrivateInteropEncoder struct {
	cfg PrivateInteropConfig
	// baseGas is the transaction builder's configured gas policy, restored after a test hook's
	// override is cleared. It is nil when the builder does not expose its policy.
	baseGas *render.GasPolicy

	mu             sync.Mutex
	prepared       map[common.Hash]optypes.Receipts
	outputs        map[common.Hash][2]common.Hash
	lastOutputHash common.Hash
	lastOutputRoot common.Hash
}

var (
	_ BlockEnricher = (*PrivateInteropEncoder)(nil)
)

func NewPrivateInteropEncoder(cfg PrivateInteropConfig) (*PrivateInteropEncoder, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	enc := &PrivateInteropEncoder{cfg: cfg, prepared: make(map[common.Hash]optypes.Receipts), outputs: make(map[common.Hash][2]common.Hash)}
	if g, ok := cfg.Txs.(gasPolicySetter); ok {
		base := g.GasPolicy()
		enc.baseGas = &base
	}
	return enc, nil
}

// gasPolicySetter is the part of render.BatcherTxBuilder the gas-policy test hook drives.
type gasPolicySetter interface {
	GasPolicy() render.GasPolicy
	SetGasPolicy(render.GasPolicy)
}

// applyGasHook installs the test hook's gas policy override, or restores the configured policy
// once it is cleared. Production (no hooks) never touches the policy.
func (e *PrivateInteropEncoder) applyGasHook() {
	hooks := e.cfg.TestHooks
	g, ok := e.cfg.Txs.(gasPolicySetter)
	if hooks == nil || !ok || e.baseGas == nil {
		return
	}
	if hooks.GasPolicyOverride != nil {
		g.SetGasPolicy(*hooks.GasPolicyOverride)
	} else {
		g.SetGasPolicy(*e.baseGas)
	}
}

// PrepareBlock fetches the private block's receipts.
//
// It runs in the block-LOADING stage rather than in the ChannelOut, because the ChannelOut is
// called under the channel-manager mutex and must not do network I/O; and because a receipt fetch
// that fails should fail the load, which the batcher already knows how to retry.
func (e *PrivateInteropEncoder) PrepareBlock(ctx context.Context, payload *eth.ExecutionPayload) error {
	// A rotated key cannot sign claims/replays for old-key epochs. Publishing
	// them would leave sequencer transactions without an accepted commitment,
	// which cannot be replayed as deposit-only recovery. Wait for expiry instead.
	_, info, err := derive.PayloadToSingularBatch(e.cfg.PrivateRollup, payload)
	if err != nil {
		return fmt.Errorf("reading private batcher authorization: %w", err)
	}
	if info.BatcherAddr != e.cfg.Batcher {
		return fmt.Errorf("private block %d authorizes batcher %s, configured signer is %s; waiting for publication cursor recovery",
			payload.BlockNumber, info.BatcherAddr, e.cfg.Batcher)
	}
	_, receipts, err := e.cfg.Receipts.FetchReceipts(ctx, payload.BlockHash)
	if err != nil {
		return fmt.Errorf("fetching private receipts for %s: %w", payload.BlockHash, err)
	}
	root, err := projection.PrivateOutput(payload)
	if err != nil {
		return err
	}
	e.mu.Lock()
	parentHash, parentRoot := e.lastOutputHash, e.lastOutputRoot
	e.mu.Unlock()
	if parentHash != payload.ParentHash {
		if payload.ParentHash == e.cfg.PrivateRollup.Genesis.L2.Hash && uint64(payload.BlockNumber) == e.cfg.PrivateRollup.Genesis.L2.Number+1 && e.cfg.Rollup.PrivateProjection != nil {
			parentRoot = e.cfg.Rollup.PrivateProjection.GenesisOutputRoot
		} else {
			info, _, err := e.cfg.Receipts.FetchReceipts(ctx, payload.ParentHash)
			if err != nil {
				return fmt.Errorf("private publication parent output: %w", err)
			}
			if info == nil || info.Hash() != payload.ParentHash || info.WithdrawalsRoot() == nil {
				return fmt.Errorf("private parent output unavailable")
			}
			parentRoot = common.Hash(eth.OutputRoot(&eth.OutputV0{StateRoot: eth.Bytes32(info.Root()), MessagePasserStorageRoot: eth.Bytes32(*info.WithdrawalsRoot()), BlockHash: info.Hash()}))
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.outputs[payload.BlockHash] = [2]common.Hash{root, parentRoot}
	e.lastOutputHash, e.lastOutputRoot = payload.BlockHash, root
	e.prepared[payload.BlockHash] = receipts
	return nil
}

func (e *PrivateInteropEncoder) take(hash common.Hash) (optypes.Receipts, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r, ok := e.prepared[hash]
	return r, ok
}

// forget drops a block's receipts once its range has been encoded. The map would otherwise grow for
// the process's life, and a re-loaded block (after a reorg) is re-enriched anyway.
func (e *PrivateInteropEncoder) forget(hashes []common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, h := range hashes {
		delete(e.prepared, h)
		delete(e.outputs, h)
	}
}

// ChannelOut satisfies ChannelOutFactory.
func (e *PrivateInteropEncoder) ChannelOut(channelCfg ChannelConfig, rollupCfg *rollup.Config) (derive.ChannelOut, error) {
	maxFrame := e.cfg.MaxFrameSize
	if maxFrame == 0 {
		maxFrame = uint64(channelCfg.MaxFrameSize)
	}
	compression := channelCfg.CompressorConfig.CompressionAlgo
	maxBlockGas, err := renderingBlockGasBudget(e.cfg.Rollup)
	if err != nil {
		return nil, err
	}
	b, err := builder.New(builder.Config{
		Rollup:       e.cfg.Rollup,
		Emitters:     e.cfg.Emitters,
		MaxFrameSize: maxFrame,
		Compression:  compression,
		MaxBlockGas:  maxBlockGas,
	}, e.cfg.Txs)
	if err != nil {
		return nil, err
	}
	return &renderChannelOut{enc: e, builder: b, maxFrame: maxFrame, compression: compression}, nil
}

var errPrivateProofPending = errors.New("private execution proof pending")

type projectionProofResult struct {
	proof []byte
	err   error
}

// renderChannelOut is a derive.ChannelOut whose input is PRIVATE blocks and whose output is the
// RENDERING's stock span batch.
//
// It implements the interface rather than wrapping a stock SpanChannelOut for one reason: the
// channel ID must be a function of the range, and SpanChannelOut randomizes it with no seam to
// inject one. Every ENCODER is still stock — the span batch, its RLP, the compressor, the frame
// layout — they are just driven from op-private-interop/builder instead of incrementally from here.
type renderChannelOut struct {
	proofResult <-chan projectionProofResult
	cancelProof context.CancelFunc
	// carrierErr is the range's carrier pre-run verdict once a carrier failed. It is sticky: the
	// range is never published, and Close keeps returning it until the channel is reset.
	carrierErr error
	enc        *PrivateInteropEncoder
	builder    *builder.Builder
	// maxFrame and compression are the channel settings this range was created with, resolved once
	// so that the private derivation-input object is framed and compressed exactly like the
	// rendering's own channel.
	maxFrame    uint64
	compression derive.CompressionAlgo

	blocks   []*render.RenderedBlock
	hashes   []common.Hash
	start    RangeStart
	haveID   bool
	id       derive.ChannelID
	inputLen int

	// privBatches and privSeqNums are the PRIVATE blocks as the stock conversion produced them,
	// kept for the range's private derivation-input object. privParent is the private chain's block
	// hash before the range. They cost one retained slice and are the only way the object and the
	// rendering are guaranteed to describe the same blocks.
	privBatches      []*derive.SingularBatch
	privSeqNums      []uint64
	privParent       common.Hash
	parentOutputRoot common.Hash
	// privDataHash is the range's privateDataHash once the object has been encoded and hashed;
	// privDataHashed says it has been, so a Close retried after a later failure does not re-run the
	// range's one expensive compression to arrive at the same bytes.
	privData       []byte
	privDataHash   common.Hash
	privDataHashed bool

	closed   bool
	full     error
	built    *builder.BuiltRange
	frameIdx int
}

var _ derive.ChannelOut = (*renderChannelOut)(nil)

func (c *renderChannelOut) ID() derive.ChannelID { return c.id }

func (c *renderChannelOut) Reset() error {
	if c.cancelProof != nil {
		c.cancelProof()
	}
	c.cancelProof, c.proofResult, c.carrierErr = nil, nil, nil
	c.blocks, c.hashes = nil, nil
	c.parentOutputRoot = common.Hash{}
	c.privBatches, c.privSeqNums, c.privParent = nil, nil, common.Hash{}
	c.privDataHashed, c.privDataHash, c.privData = false, common.Hash{}, nil
	c.haveID, c.id = false, derive.ChannelID{}
	c.inputLen = 0
	c.closed, c.full, c.built, c.frameIdx = false, nil, nil, 0
	return nil
}

// AddBlock renders one private block.
//
// The payload is the PRIVATE block, exactly as the stock loader produced it — reorg detection and
// queue bookkeeping upstream of here operate on the private chain, which is the chain the
// sequencer actually built and the only one anything can reorg.
func (c *renderChannelOut) AddBlock(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (*derive.L1BlockInfo, error) {
	if c.closed {
		return nil, derive.ErrChannelOutAlreadyClosed
	}
	if c.full != nil {
		return nil, c.full
	}
	// The private block's own L1 info, for the batcher's origin bookkeeping and timeouts. It is
	// also the check that the payload has the attributes deposit a real block always has.
	//
	// rollupCfg is the PRIVATE chain's: the channel manager passes the batcher's own config, and
	// the payload being converted is a private block. The singular batch it produces is kept, not
	// discarded, because it is exactly what the range's private derivation-input object is made of.
	privBatch, l1Info, err := derive.PayloadToSingularBatch(rollupCfg, payload)
	if err != nil {
		return l1Info, fmt.Errorf("reading the private block's L1 info: %w", err)
	}
	// The private block's own ref, from the same private config and the same L1-info deposit. Its
	// L1Origin becomes the rendering block's epoch VERBATIM — origins are copied, not chosen — and
	// its Hash and ParentHash are what the range claim publishes for its terminal block.
	ref, err := derive.PayloadToBlockRef(rollupCfg, payload)
	if err != nil {
		return l1Info, fmt.Errorf("reading the private block's ref: %w", err)
	}

	receipts, ok := c.enc.take(payload.BlockHash)
	if !ok {
		return l1Info, fmt.Errorf("no receipts prepared for private block %s", payload.BlockHash)
	}
	rendered, err := render.RenderBlock(render.PrivateBlock{
		Header:   &types.Header{Number: new(big.Int).SetUint64(uint64(payload.BlockNumber)), Time: uint64(payload.Timestamp)},
		Receipts: receipts.Geth(),
		Ref:      ref,
	}, c.enc.cfg.Emitters)
	if err != nil {
		return l1Info, fmt.Errorf("rendering private block %d: %w", payload.BlockNumber, err)
	}

	c.enc.mu.Lock()
	roots, haveRoots := c.enc.outputs[payload.BlockHash]
	c.enc.mu.Unlock()
	if !haveRoots {
		return l1Info, fmt.Errorf("private output was not prepared")
	}
	rendered.OutputRoot = roots[0]

	// Leave an overflowing block queued for the next range. Consuming it first
	// can leave Close unable to encode the range within its size limit.
	nextInput := c.inputLen + estimatedRenderedBlockBytes(rendered)
	if uint64(nextInput) > c.enc.cfg.MaxRangeBytes {
		if len(c.blocks) > 0 {
			c.full = derive.ErrCompressorFull
			return l1Info, c.full
		}
		return l1Info, fmt.Errorf("private block %d cannot fit a claim: %d range bytes (limit %d)", payload.BlockNumber, nextInput, c.enc.cfg.MaxRangeBytes)
	}

	if !c.haveID {
		// The range's identity is fixed by its first block, so this is where it is resolved — and
		// where the wait for the predecessor's execution lands. Failing here means the batcher
		// retries the block, which is the correct behaviour: a guessed parent check is a batch
		// every verifier drops.
		start, err := c.enc.cfg.Ranges.RangeStart(context.Background(), rendered.Number)
		if err != nil {
			return l1Info, fmt.Errorf("resolving the range starting at %d: %w", rendered.Number, err)
		}
		c.start = start
		c.id = builder.ChannelID(start.PrevTerminalRenderingHash, rendered.Number)
		c.privParent = privBatch.ParentHash
		c.parentOutputRoot = roots[1]
		c.haveID = true
	}

	c.blocks = append(c.blocks, rendered)
	c.hashes = append(c.hashes, payload.BlockHash)
	c.privBatches = append(c.privBatches, privBatch)
	c.privSeqNums = append(c.privSeqNums, l1Info.SequenceNumber)
	c.inputLen += estimatedRenderedBlockBytes(rendered)
	if uint64(len(c.blocks)) >= c.enc.cfg.MaxBlocksPerRange || uint64(c.inputLen) >= c.enc.cfg.MaxRangeBytes {
		c.full = derive.ErrCompressorFull
	}
	return l1Info, nil
}

// estimatedRenderedBlockBytes conservatively bounds the signed rendering transactions before the
// range is built. The 512-byte per-action allowance covers typed-transaction/signature and ABI
// overhead; topic and data bytes are counted exactly. Ending early on an overestimate is harmless.
func estimatedRenderedBlockBytes(block *render.RenderedBlock) int {
	const transactionOverhead = 512
	size := transactionOverhead
	for _, action := range block.Actions {
		size += transactionOverhead + len(action.Topics)*common.HashLength + len(action.Data)
	}
	return size
}

// Close commits to the range's private derivation input and then builds the range.
//
// # The ordering
//
//  1. the private derivation-input object is encoded from the blocks just added, and its content
//     hash computed — the ONLY place privateDataHash comes from;
//  2. only then is the claim assembled;
//  3. only then does Build produce the channel data, the frames and the blobs.
//
// Nothing before step 3 can reach L1, because a frame does not exist until step 3: ReadyBytes is
// zero and OutputFrame returns io.EOF while c.built is nil, so the batcher has literally nothing to
// send. A range whose object could not even be encoded therefore cannot be posted.
//
// The bytes are hashed and dropped. Nothing stores or serves them: the claim is a COMMITMENT to the
// range's derivation input, and the input itself reaches the operator's own followers over the
// firewalled p2p network (an off-chain archive, if an operator wants one, is an external sidecar
// reading its own private node — not this process's business).
//
// A failure at any step leaves c.closed false, so the stock retry path (ChannelBuilder.OutputFrames
// on a full channel calls Close again) re-runs it. The encoding is not repeated: the object is a
// pure function of the blocks, so a hash once computed stays right, and recompressing identical
// bytes after a later step failed would be work for nothing.
//
// Private-input compression is cached across retries. The projection candidate is
// reconstructed for admission; native execution runs asynchronously outside the caller's lock.
func (c *renderChannelOut) Close() error {
	if c.closed {
		return derive.ErrChannelOutAlreadyClosed
	}
	if len(c.blocks) == 0 {
		c.closed = true
		return nil
	}
	first, last := c.blocks[0], c.blocks[len(c.blocks)-1]

	if !c.privDataHashed {
		data, err := builder.EncodePrivateData(builder.PrivateDataConfig{
			Rollup:       c.enc.cfg.PrivateRollup,
			MaxFrameSize: c.maxFrame,
			Compression:  c.compression,
		}, &builder.PrivateRange{
			FirstBlock: first.Number,
			ParentHash: c.privParent,
			Batches:    c.privBatches,
			SeqNums:    c.privSeqNums,
		})
		if err != nil {
			return fmt.Errorf("encoding the private input for range %d-%d: %w", first.Number, last.Number, err)
		}
		c.privDataHash, c.privDataHashed = builder.PrivateDataHash(data), true
		if c.enc.cfg.Prove != nil {
			c.privData = data
		}
	}

	request := &builder.Range{
		Blocks:                    c.blocks,
		PrevTerminalRenderingHash: c.start.PrevTerminalRenderingHash,
		Continuation:              c.start.Continuation,
		Claim: &builder.ClaimInput{
			ParentOutputRoot: c.parentOutputRoot,
			RollupConfigHash: c.enc.cfg.RollupConfigHash,
			DepSetHash:       c.enc.cfg.DepSetHash,
			PrivateDataHash:  c.privDataHash,
			// Always the producer's envelope. Only insecure-stub-v1 publishes without one.
			Proof: []byte{},
		},
		StartNonce: c.start.StartNonce,
	}
	hooks := c.enc.cfg.TestHooks
	if hooks != nil {
		request.TestSkipAdmission = hooks.SkipAdmissionPreflight
	}
	c.enc.applyGasHook()
	if c.enc.cfg.Prove != nil {
		request.Prove = func(candidate *builder.BuiltRange) ([]byte, error) {
			if c.carrierErr != nil {
				return nil, c.carrierErr
			}
			// Close runs under the channel-manager mutex. Simulate and prove outside that
			// lock, retain the candidate, and retry without frames.
			if c.proofResult == nil {
				jobCtx, cancel := context.WithCancel(context.Background())
				result := make(chan projectionProofResult, 1)
				c.cancelProof, c.proofResult = cancel, result
				start, privateData, prove := c.start, c.privData, c.enc.cfg.Prove
				caller, batcher := c.enc.cfg.Caller, c.enc.cfg.Batcher
				skipPreRun := hooks != nil && hooks.SkipCarrierPreRun
				var mutate func([]byte) []byte
				var replace func(*builder.BuiltRange) ([]byte, bool)
				if hooks != nil {
					mutate, replace = hooks.MutateProof, hooks.ReplaceProof
				}
				go func() {
					if !skipPreRun {
						if err := preRunCarriers(jobCtx, caller, batcher, start.PrevTerminalRenderingHash, candidate); err != nil {
							result <- projectionProofResult{err: err}
							return
						}
					}
					if replace != nil {
						if proof, ok := replace(candidate); ok {
							result <- projectionProofResult{proof: proof}
							return
						}
					}
					proof, err := prove(jobCtx, candidate, privateData, start)
					if err == nil && mutate != nil {
						proof = mutate(proof)
					}
					result <- projectionProofResult{proof: proof, err: err}
				}()
			}
			select {
			case result := <-c.proofResult:
				c.cancelProof()
				c.cancelProof, c.proofResult = nil, nil
				if errors.Is(result.err, ErrCarrierPreRun) {
					c.carrierErr = result.err
				}
				return result.proof, result.err
			default:
				return nil, errPrivateProofPending
			}
		}
	}
	built, err := c.builder.Build(request)
	if err != nil {
		return fmt.Errorf("building the rendering range %d-%d: %w", first.Number, last.Number, err)
	}
	// The published claim carries the proof, so its calldata and gas limit differ from the
	// candidate's: statically re-check the carriers that are actually published.
	if c.enc.cfg.Prove != nil && (hooks == nil || !hooks.SkipCarrierPreRun) {
		if err := checkRangeCarrierGas(built); err != nil {
			c.carrierErr = err
			return fmt.Errorf("building the rendering range %d-%d: %w", first.Number, last.Number, err)
		}
	}
	c.built = built
	c.privData = nil
	c.closed = true
	c.enc.forget(c.hashes)
	return nil
}

// OutputFrame hands over one already-built stock frame.
//
// maxSize is honoured by construction: the frames were built with the channel config's
// MaxFrameSize. It is still checked, because a caller asking for less than we built would otherwise
// get an oversized frame silently.
func (c *renderChannelOut) OutputFrame(w *bytes.Buffer, maxSize uint64) (uint16, error) {
	if maxSize < derive.FrameV0OverHeadSize {
		return 0, derive.ErrMaxFrameSizeTooSmall
	}
	if !c.closed {
		return 0, io.EOF
	}
	if c.built == nil || c.frameIdx >= len(c.built.Frames) {
		return 0, io.EOF
	}
	frame := c.built.Frames[c.frameIdx]
	if uint64(len(frame)) > maxSize {
		return 0, fmt.Errorf("frame %d is %d bytes but the caller allows %d", c.frameIdx, len(frame), maxSize)
	}
	n := uint16(c.frameIdx)
	c.frameIdx++
	if _, err := w.Write(frame); err != nil {
		return n, err
	}
	if c.frameIdx == len(c.built.Frames) {
		return n, io.EOF
	}
	return n, nil
}

// ReadyBytes is zero until Close: a range is encoded as a whole, so there is nothing to emit before
// its last block has arrived. After Close it is the bytes still to be handed over.
func (c *renderChannelOut) ReadyBytes() int {
	if !c.closed || c.built == nil {
		return 0
	}
	var n int
	for _, f := range c.built.Frames[c.frameIdx:] {
		n += len(f)
	}
	return n
}

func (c *renderChannelOut) InputBytes() int { return c.inputLen }
func (c *renderChannelOut) FullErr() error  { return c.full }
func (c *renderChannelOut) Flush() error    { return nil }

// DiscardCompressor releases retained proof input and cancels work for abandoned ranges.
func (c *renderChannelOut) DiscardCompressor() {
	if c.cancelProof != nil {
		c.cancelProof()
	}
	c.cancelProof, c.proofResult, c.privData = nil, nil, nil
}

// BuiltRange exposes the encoded range, for tests and for operator tooling that wants to see what
// was posted. It is nil until Close.
func (c *renderChannelOut) BuiltRange() *builder.BuiltRange { return c.built }
