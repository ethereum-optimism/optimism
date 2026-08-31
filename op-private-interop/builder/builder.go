// Package builder synthesizes the rendering's batches.
//
// There is no public sequencer. The rendering exists only as derivation input until a public node
// derives and executes it, so the operator's builder must produce, for each cadence range, exactly
// the bytes a stock batcher would have produced had a stock sequencer built the rendering: one span
// batch, stock frames, stock blob encoding, derivation version 0x00, posted to the normal inbox. A
// public verifier cannot tell this apart from any other chain's batcher output, and that is the
// requirement, not a nicety.
//
// The architecture is op-private-interop/docs/DESIGN.md, section "Batch construction". This
// package owns the range: origins, the span batch, the deterministic
// channel, the frames and the blobs. It does not own the block-level transformation (that is
// op-private-interop/render) and it does not own the batcher's lifecycle (that is op-batcher, which
// calls in here at its terminal seam).
//
// # Byte-determinism is the point
//
// The whole payload must be a pure function of private-chain data plus two inputs the operator
// supplies: the previous range's terminal RENDERING hash, and the operator EOA's starting nonce.
// Given those, building the same range twice from fresh state produces identical span-batch bytes,
// identical frames and identical blobs. TestRangeIsByteDeterministic is the gate.
//
// This is why the channel ID is derived rather than random. The stock batcher calls
// crypto/rand for it (derive.SpanChannelOut.setRandomID), which is correct for a chain whose batcher
// is the only writer and wrong for a payload that must be reproducible by anyone holding the same
// private data. Normatively:
//
//	channelID = keccak256(prevRangeTerminalRenderingHash ‖ uint64be(firstBlock))[:16]
//
// Everything else about the channel is stock: the same RLP encoding of the same RawSpanBatch, the
// same zlib settings, the same frame layout and the same blob packing that
// derive.SpanChannelOut/op-batcher produce. This package reimplements the *drive* of that pipeline
// (because SpanChannelOut has no seam for a fixed channel ID) and reuses every *encoder* in it.
//
// # Origins are copied, not chosen
//
// A rendering block's L1 epoch is the private block's OWN L1 origin, read from that block's L1-info
// deposit and carried through unchanged, along with its sequence number.
//
// This replaced an independent origin-selection rule (newest L1 block with time <= the block's
// timestamp, clamped to one epoch per block, drift-checked), and deleting that machinery is the
// point rather than a side effect. The private sequencer already chose origins under the identical
// stock rules at the identical timestamps, so re-deriving them could only ever agree with the
// private chain or be wrong — and it needed a live L1 view, a confirmation-depth buffer, and a
// past-range view to do it. Copying needs none of that: the batch payload is now a pure function of
// private-chain data with NO live L1 input at all, span validity holds by construction because the
// copied origins are origins a stock sequencer already produced, and the rendering's sequence
// numbers equal the private chain's, which is what lets the public supernode serve follow refs.
//
// The builder is therefore an effectively stock batch submitter: it strips the private
// transactions, inserts the claim transaction and the replay transactions, and alters nothing about
// block progression.
//
// # l1Head
//
// The claim's l1Head is the TERMINAL BLOCK'S OWN L1 ORIGIN — derived here, not supplied.
//
// It used to be an operator-supplied "L1 head the range was derived under", read off the same view
// origins were chosen from. That view is gone, and rather than keep it alive for one field: since
// origin-copy, the newest L1 block a range actually depends on IS its terminal block's origin. That
// is the honest reading of the same field, it costs no L1 access, and unlike a supplied value a
// verifier can CHECK it — the rendering's own terminal block carries that origin. A value merely
// ahead of it names L1 the range never consumed.
//
// # What the builder cannot compute
//
// The previous range's terminal rendering block hash. It is an execution-derived field of a chain
// this package is writing the input for, so the CALLER supplies it in Range.PrevTerminalRenderingHash,
// having read it off a node following the rendering the operator already posted (op-batcher's
// RangeSource is that caller). It has two jobs: it is the span's 20-byte parent check, and it seeds
// the channel ID. A wrong one is a plain BatchDrop at every verifier: a loud stall attributable to
// the operator, never a divergence and never a reset.
package builder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	derivepar "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	// ErrRange is returned when a range's blocks are not what a span batch can describe.
	ErrRange = errors.New("invalid rendering range")
	// ErrNoOrigin is returned when a block arrives without the private L1 origin the rendering must
	// copy. It is never a choice this package could make instead — see "Origins are copied".
	ErrNoOrigin = errors.New("private block has no L1 origin to copy")
)

// Config is the builder's static configuration.
type Config struct {
	// Rollup is the RENDERING's rollup config. Only the genesis timestamp, the chain ID, the block
	// time and the sequencer drift are read, and all four are frozen for the chain's life.
	Rollup *rollup.Config
	// Emitters is the rendering's emitter set.
	Emitters render.EmitterSet
	// MaxFrameSize caps a frame. The default is one byte under a blob's capacity, because the blob
	// carries the derivation version byte in front of the frame.
	MaxFrameSize uint64
	// Compression is the channel compression algorithm. Stock batchers default to zlib, and both
	// zlib and brotli are deterministic functions of (input, level).
	Compression derive.CompressionAlgo
}

// DefaultMaxFrameSize is one frame per blob, which is what the stock batcher targets for blob DA.
const DefaultMaxFrameSize = uint64(eth.MaxBlobDataSize - 1)

func (c *Config) check() error {
	if c.Rollup == nil {
		return errors.New("no rollup config")
	}
	if c.MaxFrameSize == 0 {
		c.MaxFrameSize = DefaultMaxFrameSize
	}
	if c.MaxFrameSize < derive.FrameV0OverHeadSize {
		return fmt.Errorf("max frame size %d is below the frame overhead", c.MaxFrameSize)
	}
	if c.Compression == "" {
		c.Compression = derive.Zlib
	}
	return nil
}

// Range is one cadence's worth of input.
type Range struct {
	// Blocks are the rendered blocks, contiguous and ascending in number and timestamp.
	Blocks []*render.RenderedBlock
	// PrevTerminalRenderingHash is the previous range's terminal rendering block hash, read off a
	// node following the rendering. It is the span's parent check and the channel ID's seed.
	PrevTerminalRenderingHash common.Hash
	// Claim carries the operator-supplied part of THIS range's claim. It is required: every range,
	// including the chain's first, opens with its own claim.
	Claim *ClaimInput
	// StartNonce is the operator EOA's nonce for the first transaction of the range.
	StartNonce uint64
}

// BuiltBlock is one rendering block as the batch describes it.
type BuiltBlock struct {
	Number    uint64
	Timestamp uint64
	Origin    eth.BlockID
	SeqNum    uint64
	// Txs are the block's non-deposit transactions: one replay transaction per rendered log, in
	// rendered order, preceded — in a range's FIRST block only — by that range's claim
	// transaction. The claim emits no log, so the k-th REPLAY transaction still emits rendering
	// log k with no exceptions anywhere in a range.
	Txs []hexutil.Bytes
}

// ClaimInput is the part of a range's claim the builder cannot derive from the range itself: two
// configuration hashes, a content address, and the proof slot.
//
// Everything else — firstBlock, lastBlock, privateTerminalBlockHash, privateTerminalParentHash and
// l1Head — is derived here rather than accepted, because the builder holds the range and would only
// be checking the caller's copy against its own. Assembling them itself makes "the claim describes
// the range it opens" true by construction rather than by agreement, which matters for a record
// that is the permanent claim about that range.
type ClaimInput struct {
	// RollupConfigHash and DepSetHash pin which chain and which dependency set the claim speaks for.
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	// PrivateDataHash is the content address of the range's full private derivation input: a
	// COMMITMENT, not a pointer. Nothing publishes the object — the bytes are hashed and dropped,
	// and they reach every legitimate reader over the operator's private p2p network.
	PrivateDataHash common.Hash
	// Proof is empty in v1, where the registry refuses a non-empty slot. In proven mode it attests
	// the claim series that follows it in the same block.
	Proof []byte
}

// BuiltRange is everything the batcher needs to post one cadence.
type BuiltRange struct {
	FirstBlock, LastBlock uint64
	// Claim is the full claim this range posted, as assembled and encoded. Exposed so the operator
	// can record what it committed to without re-deriving it.
	Claim     *codec.RangeClaim
	ChannelID derive.ChannelID
	Blocks    []BuiltBlock
	SpanBatch *derive.SpanBatch
	// ChannelData is the compressed channel payload: the frames' concatenated content.
	ChannelData []byte
	// Frames are stock frame encodings, ready to be prefixed with the derivation version byte.
	Frames [][]byte
	// Blobs are the frames as blobs, each with the 0x00 derivation version byte in front — exactly
	// what op-batcher's txData.Blobs() produces from stock frames.
	Blobs []*eth.Blob
	// NextNonce is the operator EOA's nonce after the range, which is the next range's StartNonce.
	NextNonce uint64
}

// Builder synthesizes ranges. It holds configuration and a transaction builder; the transaction
// builder's nonce is repositioned at every Build, so a Builder has no state that survives a range
// and building the same range twice from the same Builder gives the same bytes.
type Builder struct {
	cfg Config
	txs render.ReplayTxBuilder
}

// New builds a range builder.
func New(cfg Config, txs render.ReplayTxBuilder) (*Builder, error) {
	if err := cfg.check(); err != nil {
		return nil, err
	}
	if txs == nil {
		return nil, errors.New("no replay transaction builder")
	}
	return &Builder{cfg: cfg, txs: txs}, nil
}

// Build turns one range of rendered blocks into a posted cadence's worth of bytes.
func (b *Builder) Build(r *Range) (*BuiltRange, error) {
	if len(r.Blocks) == 0 {
		return nil, fmt.Errorf("%w: no blocks", ErrRange)
	}
	first, last := r.Blocks[0], r.Blocks[len(r.Blocks)-1]
	if err := b.checkContiguous(r.Blocks); err != nil {
		return nil, err
	}
	if r.Claim == nil {
		return nil, fmt.Errorf("%w: no claim for range %d-%d", ErrRange, first.Number, last.Number)
	}

	// The claim is assembled here rather than accepted from the caller, so that it cannot disagree
	// with the range it opens. PrivateTerminalBlockHash comes straight from the range's last
	// rendered block: it is the PRIVATE chain's hash, a fact that already existed before any of
	// this ran, which is exactly what makes leading placement non-circular.
	claim := &codec.RangeClaim{
		FirstBlock:                first.Number,
		LastBlock:                 last.Number,
		PrivateTerminalBlockHash:  last.PrivateRef.Hash,
		PrivateTerminalParentHash: last.PrivateRef.ParentHash,
		// L1Head is DERIVED, not supplied: it is the terminal block's own L1 origin, which since
		// origin-copy is the newest L1 block the range actually depends on. See "l1Head" below.
		L1Head:           last.PrivateRef.L1Origin.Hash,
		RollupConfigHash: r.Claim.RollupConfigHash,
		DepSetHash:       r.Claim.DepSetHash,
		PrivateDataHash:  r.Claim.PrivateDataHash,
		Proof:            r.Claim.Proof,
	}
	if claim.PrivateTerminalBlockHash == (common.Hash{}) {
		// A zero hash means the range's blocks were built without their private identity, which
		// would post a claim committing to nothing at all.
		return nil, fmt.Errorf("%w: block %d has no private block hash", ErrRange, last.Number)
	}

	out := &BuiltRange{
		FirstBlock: first.Number,
		LastBlock:  last.Number,
		Claim:      claim,
		ChannelID:  ChannelID(r.PrevTerminalRenderingHash, first.Number),
		Blocks:     make([]BuiltBlock, 0, len(r.Blocks)),
	}

	b.txs.Reset(r.StartNonce)
	span := derive.NewSpanBatch(b.cfg.Rollup.Genesis.L2Time, b.cfg.Rollup.L2ChainID)

	for i, blk := range r.Blocks {
		// ORIGIN-COPY. The rendering block's epoch is the private block's own, verbatim.
		origin, seqNum := blk.PrivateRef.L1Origin, blk.PrivateRef.SequenceNumber
		if origin == (eth.BlockID{}) {
			return nil, fmt.Errorf("%w: block %d", ErrNoOrigin, blk.Number)
		}

		txs, err := b.blockTxs(blk, i == 0, claim)
		if err != nil {
			return nil, err
		}
		out.Blocks = append(out.Blocks, BuiltBlock{
			Number: blk.Number, Timestamp: blk.Timestamp, Origin: origin, SeqNum: seqNum, Txs: txs,
		})

		sb := &derive.SingularBatch{
			EpochNum:     rollup.Epoch(origin.Number),
			EpochHash:    origin.Hash,
			Timestamp:    blk.Timestamp,
			Transactions: txs,
		}
		if i == 0 {
			// AppendSingularBatch reads ParentHash only for the first element: a span batch carries
			// ONE 20-byte parent check for the whole range, which is precisely why the rendering can
			// be batched at all. Intra-range parent hashes are unknowable before execution.
			sb.ParentHash = r.PrevTerminalRenderingHash
		}
		if err := span.AppendSingularBatch(sb, seqNum); err != nil {
			return nil, fmt.Errorf("appending block %d to the span: %w", blk.Number, err)
		}
	}
	out.SpanBatch = span
	out.NextNonce = b.txs.Nonce()

	raw, err := span.ToRawSpanBatch()
	if err != nil {
		return nil, fmt.Errorf("converting the span batch: %w", err)
	}
	var rlpBuf bytes.Buffer
	if err := rlp.Encode(&rlpBuf, derive.NewBatchData(raw)); err != nil {
		return nil, fmt.Errorf("encoding the span batch: %w", err)
	}
	if out.ChannelData, err = compressWith(b.cfg.Compression, rlpBuf.Bytes()); err != nil {
		return nil, err
	}
	out.Frames = frames(out.ChannelID, out.ChannelData, b.cfg.MaxFrameSize)
	if out.Blobs, err = blobs(out.Frames); err != nil {
		return nil, err
	}
	return out, nil
}

// ChannelID is the normative deterministic channel ID.
func ChannelID(prevTerminalRenderingHash common.Hash, firstBlock uint64) derive.ChannelID {
	var seed [40]byte
	copy(seed[:32], prevTerminalRenderingHash[:])
	binary.BigEndian.PutUint64(seed[32:], firstBlock)
	var id derive.ChannelID
	copy(id[:], crypto.Keccak256(seed[:])[:16])
	return id
}

// blockTxs builds one block's transaction list: the range's claim first, in its first block only,
// then one replay transaction per rendered log in RenderedLogs order.
//
// The claim can lead because the registry emits no log. A logging registry would put a
// rendering-only log at index 0 of every range-opening block and push every message in it up by
// one, which is the one thing the canonical-position rule cannot survive.
func (b *Builder) blockTxs(blk *render.RenderedBlock, isFirst bool, claim *codec.RangeClaim) ([]hexutil.Bytes, error) {
	var out []hexutil.Bytes
	appendTx := func(tx *types.Transaction) error {
		raw, err := tx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("encoding a rendering transaction in block %d: %w", blk.Number, err)
		}
		if len(raw) > 0 && raw[0] == optypes.DepositTxType {
			// Stock validation drops any batch containing a deposit-type transaction, and the
			// rendering has no deposits by construction. Catch it here rather than at a verifier.
			return fmt.Errorf("%w: block %d contains a deposit-type transaction", ErrRange, blk.Number)
		}
		out = append(out, raw)
		return nil
	}
	if isFirst {
		tx, err := b.txs.ClaimTx(claim)
		if err != nil {
			return nil, fmt.Errorf("block %d, claim for range %d-%d: %w", blk.Number, claim.FirstBlock, claim.LastBlock, err)
		}
		if err := appendTx(tx); err != nil {
			return nil, err
		}
	}
	for _, act := range blk.Actions {
		tx, err := b.txs.ReplayTx(act)
		if err != nil {
			return nil, fmt.Errorf("block %d, rendered index %d: %w", blk.Number, act.RenderedLogIndex, err)
		}
		if err := appendTx(tx); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *Builder) checkContiguous(blocks []*render.RenderedBlock) error {
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Number != blocks[i-1].Number+1 {
			return fmt.Errorf("%w: block %d follows %d", ErrRange, blocks[i].Number, blocks[i-1].Number)
		}
		if want := blocks[i-1].Timestamp + b.cfg.Rollup.BlockTime; blocks[i].Timestamp != want {
			return fmt.Errorf("%w: block %d has timestamp %d, expected %d",
				ErrRange, blocks[i].Number, blocks[i].Timestamp, want)
		}
	}
	return nil
}

// compressWith is the stock channel compression, shared by the rendering's range and by the private
// derivation-input object so that both are the same stock pipeline.
//
// It is a FRESH compressor per call: span batches are not compressed incrementally (see
// derive.SpanChannelOut.compress), so a fresh one is both correct and the thing that makes the
// output independent of the builder's history.
func compressWith(algo derive.CompressionAlgo, rlpBytes []byte) ([]byte, error) {
	c, err := derive.NewChannelCompressor(algo)
	if err != nil {
		return nil, fmt.Errorf("creating the channel compressor: %w", err)
	}
	if _, err := c.Write(rlpBytes); err != nil {
		return nil, fmt.Errorf("compressing the channel: %w", err)
	}
	if err := c.Close(); err != nil {
		return nil, fmt.Errorf("closing the channel compressor: %w", err)
	}
	return append([]byte(nil), c.GetCompressed().Bytes()...), nil
}

// frames chunks a compressed channel into stock frames, the same way
// derive.createEmptyFrame + Frame.MarshalBinary do.
func frames(id derive.ChannelID, data []byte, maxSize uint64) [][]byte {
	maxData := int(maxSize - derive.FrameV0OverHeadSize)
	var out [][]byte
	for n, off := 0, 0; ; n++ {
		end := min(off+maxData, len(data))
		f := derive.Frame{
			ID:          id,
			FrameNumber: uint16(n),
			Data:        data[off:end],
			IsLast:      end == len(data),
		}
		var buf bytes.Buffer
		// MarshalBinary only ever fails on a failing writer, and a bytes.Buffer does not fail.
		_ = f.MarshalBinary(&buf)
		out = append(out, buf.Bytes())
		off = end
		if f.IsLast {
			return out
		}
	}
}

// blobs packs frames exactly as op-batcher's txData.Blobs() does: one blob per frame, each frame
// prefixed with the 0x00 derivation version byte. There is no raw or skipped blob in this design.
func blobs(frames [][]byte) ([]*eth.Blob, error) {
	out := make([]*eth.Blob, 0, len(frames))
	for i, f := range frames {
		var blob eth.Blob
		if err := blob.FromData(append([]byte{derivepar.DerivationVersion0}, f...)); err != nil {
			return nil, fmt.Errorf("encoding frame %d as a blob: %w", i, err)
		}
		out = append(out, &blob)
	}
	return out, nil
}
