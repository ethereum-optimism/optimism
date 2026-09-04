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
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Private Interop's terminal seam.
//
// The batcher's whole lifecycle is reused unchanged: unsafe-head polling, reorg detection against
// the PRIVATE chain, channel retry, the txmgr, blob submission, throttling. What changes is only
// the last stage — the blocks it loaded are private, and the bytes it posts describe the RENDERING.
//
// The seam is the ChannelOutFactory hook that already exists in channelManager, plus one small
// addition to the block-loading path (BlockEnricher) so that receipts, which an execution payload
// does not carry and which the transformation needs, are fetched alongside each block. That shape
// is HARVESTED from an earlier lane's batcher work, whose terminal encoding is otherwise retired:
// only the lifecycle seam carries over.
//
// Private Interop posts nothing foreign: the frames this file emits are stock frames with a
// deterministic channel ID, and op-batcher's own txData.Blobs() puts the 0x00 derivation version
// byte in front exactly as it does for any chain. Stock op-batcher files are therefore untouched
// apart from the enricher hook.

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

// PrivateInteropConfig configures the terminal seam.
type PrivateInteropConfig struct {
	// Rollup is the RENDERING's rollup config. It is NOT the private chain's: the timestamps and
	// numbers coincide block-for-block, but the genesis, chain ID and drift the span batch is
	// encoded against belong to the chain being described.
	Rollup *rollup.Config
	// PrivateRollup is the PRIVATE chain's rollup config — the batcher's own, from --rollup-rpc. It
	// is needed for exactly one thing: encoding the range's private derivation input, which is a
	// description of the private chain and must be encoded against the private chain.
	PrivateRollup *rollup.Config
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

	mu       sync.Mutex
	prepared map[common.Hash]optypes.Receipts
}

var (
	_ BlockEnricher = (*PrivateInteropEncoder)(nil)
)

func NewPrivateInteropEncoder(cfg PrivateInteropConfig) (*PrivateInteropEncoder, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return &PrivateInteropEncoder{cfg: cfg, prepared: make(map[common.Hash]optypes.Receipts)}, nil
}

// PrepareBlock fetches the private block's receipts.
//
// It runs in the block-LOADING stage rather than in the ChannelOut, because the ChannelOut is
// called under the channel-manager mutex and must not do network I/O; and because a receipt fetch
// that fails should fail the load, which the batcher already knows how to retry.
func (e *PrivateInteropEncoder) PrepareBlock(ctx context.Context, payload *eth.ExecutionPayload) error {
	_, receipts, err := e.cfg.Receipts.FetchReceipts(ctx, payload.BlockHash)
	if err != nil {
		return fmt.Errorf("fetching private receipts for %s: %w", payload.BlockHash, err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
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

// renderChannelOut is a derive.ChannelOut whose input is PRIVATE blocks and whose output is the
// RENDERING's stock span batch.
//
// It implements the interface rather than wrapping a stock SpanChannelOut for one reason: the
// channel ID must be a function of the range, and SpanChannelOut randomizes it with no seam to
// inject one. Every ENCODER is still stock — the span batch, its RLP, the compressor, the frame
// layout — they are just driven from op-private-interop/builder instead of incrementally from here.
type renderChannelOut struct {
	enc     *PrivateInteropEncoder
	builder *builder.Builder
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
	privBatches []*derive.SingularBatch
	privSeqNums []uint64
	privParent  common.Hash
	// privDataHash is the range's privateDataHash once the object has been encoded and hashed;
	// privDataHashed says it has been, so a Close retried after a later failure does not re-run the
	// range's one expensive compression to arrive at the same bytes.
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
	c.blocks, c.hashes = nil, nil
	c.privBatches, c.privSeqNums, c.privParent = nil, nil, common.Hash{}
	c.privDataHashed, c.privDataHash = false, common.Hash{}
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
	size := 0
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
// Everything expensive happens exactly once, here, because a span batch is not compressible
// incrementally anyway (see derive.SpanChannelOut.compress).
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
	}

	built, err := c.builder.Build(&builder.Range{
		Blocks:                    c.blocks,
		PrevTerminalRenderingHash: c.start.PrevTerminalRenderingHash,
		Claim: &builder.ClaimInput{
			RollupConfigHash: c.enc.cfg.RollupConfigHash,
			DepSetHash:       c.enc.cfg.DepSetHash,
			PrivateDataHash:  c.privDataHash,
			// v1 is attested, never proven: the registry rejects a non-empty slot.
			Proof: nil,
		},
		StartNonce: c.start.StartNonce,
	})
	if err != nil {
		return fmt.Errorf("building the rendering range %d-%d: %w", first.Number, last.Number, err)
	}
	c.built = built
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

// DiscardCompressor is a no-op: this encoder holds no compressor between ranges. The stock one
// exists to release a long-lived buffer, and there is none here.
func (c *renderChannelOut) DiscardCompressor() {}

// BuiltRange exposes the encoded range, for tests and for operator tooling that wants to see what
// was posted. It is nil until Close.
func (c *renderChannelOut) BuiltRange() *builder.BuiltRange { return c.built }
