package builder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivepar "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
)

// The range's FULL PRIVATE DERIVATION INPUT — the object whose keccak the range's claim carries as
// privateDataHash.
//
// It is a COMMITMENT, not a payload. Nothing publishes these bytes: private data reaches the
// operator's own followers over the firewalled p2p network, and the claim binds — for anyone who
// later holds the bytes, from a follower or from an operator's own backup — exactly which object is
// the range's real derivation input. So the ENCODING below is consensus-relevant even though the
// encoding's output never leaves the batcher: change it and every past claim commits to something
// nobody can reproduce.
//
// # What it is
//
// Exactly what a STOCK batcher for the private chain would have posted for these blocks: the
// derivation version byte, then stock channel frames over one span batch of the private blocks,
// with the private chain's own transactions in them. Not a bespoke archive format. The private
// chain is a real OP Stack chain, so the one description of it that stock software can consume is
// the one stock software writes, and a reader holding this object plus the private genesis can
// reproduce the private chain with no code of ours at all.
//
//	object = 0x00 ‖ frame₀ ‖ frame₁ ‖ …
//
// One version byte for the whole object, then frames back to back, which is what
// derive.ParseFrames expects of a single data item.
//
// # Why it is built here and not by a second batcher
//
// The private blocks pass through the terminal seam already: it loads them to RENDER them. Encoding
// them a second way, from the same payloads, in the same pass, is the only way the object and the
// rendering can be guaranteed to describe the same range. A separate stock batcher pointed at the
// same chain would produce an object for a range that has no relationship to any cadence boundary.
//
// # Determinism
//
// Same as the rendering's: a pure function of the private blocks plus the parent hash. That is why
// the channel ID is derived here too — derive.SpanChannelOut calls crypto/rand for it, which is
// correct for a chain whose batcher is the only writer and wrong for an object whose NAME is the
// hash of its bytes. A random channel ID would give the same range a new content address on every
// build, and the claim commits to that address.

// PrivateDataConfig is the static configuration for encoding a range's private input.
type PrivateDataConfig struct {
	// Rollup is the PRIVATE chain's rollup config — the chain the object describes. This is the one
	// place in this package where "the rollup config" does not mean the rendering's.
	Rollup *rollup.Config
	// MaxFrameSize caps a frame. Zero takes DefaultMaxFrameSize. Frames are not blobs here (the
	// object never goes on L1), but the same cap keeps the object parseable by anything that
	// enforces the stock 1 MB frame limit.
	MaxFrameSize uint64
	// Compression is the channel compression algorithm.
	Compression derive.CompressionAlgo
}

func (c *PrivateDataConfig) check() error {
	if c.Rollup == nil {
		return errors.New("no private rollup config")
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

// PrivateRange is one cadence's worth of PRIVATE blocks, as the stock conversion already produced
// them: the seam calls derive.PayloadToSingularBatch on every block it loads, and this is the
// result kept rather than discarded.
type PrivateRange struct {
	// FirstBlock is the range's first private block number. It seeds the channel ID; the batches
	// themselves carry timestamps, not numbers.
	FirstBlock uint64
	// ParentHash is the PRIVATE chain's block hash before FirstBlock: the span's 20-byte parent
	// check and the channel ID's other seed. Unlike the rendering's, it needs no follower — the
	// private chain is executed, and the seam loaded the block that names it.
	ParentHash common.Hash
	// Batches are the private blocks, ascending. SeqNums are their sequence numbers within their L1
	// origins, from each block's own L1-attributes deposit, and must be the same length.
	Batches []*derive.SingularBatch
	SeqNums []uint64
}

// PrivateDataHash is the object's content address: keccak256 of EncodePrivateData's output, and the
// value a range's claim carries as privateDataHash.
//
// It lives beside the encoding on purpose. The hash means nothing except "the keccak of THESE
// bytes", so the function that produces the bytes and the function that names them belong in one
// place, computed one way, by the builder that assembles the claim.
func PrivateDataHash(data []byte) common.Hash {
	return crypto.Keccak256Hash(data)
}

// EncodePrivateData builds the range's private derivation-input object.
func EncodePrivateData(cfg PrivateDataConfig, r *PrivateRange) ([]byte, error) {
	if err := cfg.check(); err != nil {
		return nil, err
	}
	if len(r.Batches) == 0 {
		return nil, fmt.Errorf("%w: no private blocks", ErrRange)
	}
	if len(r.Batches) != len(r.SeqNums) {
		return nil, fmt.Errorf("%w: %d private blocks but %d sequence numbers", ErrRange, len(r.Batches), len(r.SeqNums))
	}

	span := derive.NewSpanBatch(cfg.Rollup.Genesis.L2Time, cfg.Rollup.L2ChainID)
	for i, b := range r.Batches {
		if err := span.AppendSingularBatch(b, r.SeqNums[i]); err != nil {
			return nil, fmt.Errorf("appending private block %d to the span: %w", r.FirstBlock+uint64(i), err)
		}
	}
	raw, err := span.ToRawSpanBatch()
	if err != nil {
		return nil, fmt.Errorf("converting the private span batch: %w", err)
	}
	var rlpBuf bytes.Buffer
	if err := rlp.Encode(&rlpBuf, derive.NewBatchData(raw)); err != nil {
		return nil, fmt.Errorf("encoding the private span batch: %w", err)
	}
	channel, err := compressWith(cfg.Compression, rlpBuf.Bytes())
	if err != nil {
		return nil, err
	}

	out := []byte{derivepar.DerivationVersion0}
	for _, f := range frames(ChannelID(r.ParentHash, r.FirstBlock), channel, cfg.MaxFrameSize) {
		out = append(out, f...)
	}
	return out, nil
}
