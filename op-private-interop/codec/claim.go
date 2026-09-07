// Package codec defines the version-2 private range claim. Its canonical ABI
// encoding binds private block identity, derivation inputs, proof bytes, and the
// publicly available opaque write records. Attested mode requires an empty proof.
// Old version-1 encodings are deliberately rejected; activation requires a new
// projection genesis and matching producer/readers.
package codec

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-private-interop/writes"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ClaimVersion is the version this package encodes and the only one it decodes. Any field
	// change in RangeClaim bumps it.
	//
	// It is a field of the ABI struct rather than a byte in front of it, so a consumer that got
	// hold of the value without its framing — out of a log, out of a trace, out of a proof's
	// public inputs — still knows what it is holding.
	ClaimVersion uint8 = 2

	// EncodedSizeEmptyProof is the minimum encoding with empty proof and writes:
	// the outer offset, eleven head words, and two dynamic length words.
	//
	// It is also the minimum length of ANY valid encoding, which is why it is the first thing
	// Decode checks.
	EncodedSizeEmptyProof = (headWords + 3) * 32

	// MaxProofSize caps the proof slot at 64 KiB, enforced by BOTH Encode and Decode.
	//
	// The cap is a bound on the wire, not a prediction about proof systems: the claim rides in
	// an ordinary L2 transaction, so an uncapped slot makes the cost and the block-space footprint
	// of a cadence unbounded, and makes a decoder's memory footprint a function of whatever a
	// caller hands it. 64 KiB is comfortably above any succinct proof this design would carry and
	// far below anything that would strain a block.
	//
	// Enforced on the producing side as well, for the same reason the inverted-range rule is: a
	// producer that can emit what its own decoder refuses will do it eventually, and at the worst
	// possible time.
	MaxProofSize = 65536

	// MaxEncodedSize bounds both dynamic payloads, allowing their ABI padding.
	// Decode checks it before allocating from untrusted offsets.
	MaxEncodedSize = EncodedSizeEmptyProof + MaxProofSize + writes.MaxEncodedSize + 32

	// headWords counts nine static fields and the offsets to proof and writes.
	headWords = 11
	// proofOffset is the byte offset of the proof's length word, measured from the start of the
	// TUPLE (that is, from the word after the outer offset) — the value the proof offset word must
	// carry in a canonical encoding.
	proofOffset = headWords * 32
)

var (
	ErrTruncated     = errors.New("truncated range claim")
	ErrBadVersion    = errors.New("unsupported range claim version")
	ErrInvertedRange = errors.New("range claim covers an inverted block range")
	ErrProofNotEmpty = errors.New("attested-mode verifier refuses a non-empty proof slot")
	ErrProofTooLarge = errors.New("range claim proof slot exceeds the maximum size")
	ErrNonCanonical  = errors.New("range claim is not in canonical ABI form")
)

// RangeClaim is a range's claim: everything about the range it opens that the public chain's own
// blocks and receipts cannot say.
//
// Read it as three groups. FirstBlock, LastBlock and PrivateTerminalBlockHash identify the RANGE
// and commit to the private chain state it will render. L1Head, RollupConfigHash and DepSetHash are
// the BINDING to a particular L1 view and a particular chain configuration, so a claim cannot be
// read as speaking for a different chain or a different dependency set. PrivateDataHash is the
// CONTENT ADDRESS of the full private derivation input — the single field connecting the public
// record to the object a recovery or audit replay actually fetches.
//
// The wire's `version` field is not a Go field. There is exactly one accepted value, and a struct
// member for it would be a second source of truth: an encoder could then emit a version its own
// decoder refuses, which is a bug that compiles. Encode always writes ClaimVersion and Decode
// accepts only it.
type RangeClaim struct {
	// FirstBlock and LastBlock are the inclusive block range this claim describes — the range the
	// claim itself opens, not a previous one. Public block numbers are private block numbers: the
	// chains are block-for-block.
	FirstBlock uint64
	LastBlock  uint64
	// PrivateTerminalBlockHash is the PRIVATE chain's block hash at LastBlock.
	//
	// Private, not public, and that is what makes a claim able to describe its own range. The
	// private chain runs ahead of its public rendering, so when the operator builds this range's
	// leading block the private blocks the range will render already exist: their terminal hash is
	// a KNOWN PAST FACT at build time. The public terminal hash could not be here — it is a
	// function of a range that contains this very transaction, which is circular.
	//
	// With it, a claim has the prevRoot -> newRoot chaining shape the proof-batch wire had: each
	// claim pins the private state its range ends at, and the next claim starts from a range whose
	// endpoint is already on the public record. That is what gives a proof something to be a proof
	// ABOUT, and it is what an auditor walks.
	//
	// It is a deliberate disclosure: this publishes exactly one commitment to private chain state
	// per range. One 32-byte hash per cadence, of a block whose contents stay private, was judged
	// the right price for a chain of claims that means anything. Writes additionally publishes
	// opaque changed-state identifiers and value commitments.
	//
	// Note this is NOT the span-batch parent check. That check is the previous PUBLIC block's hash,
	// truncated to 20 bytes, and the batching service reads it from the public chain it is building
	// rather than from any claim.
	PrivateTerminalBlockHash common.Hash
	// PrivateTerminalParentHash is that same block's parent hash, and it is here for exactly one
	// reason: it is THE ONE FIELD of the private chain's terminal L2BlockRef that public data cannot
	// supply.
	//
	// Since origin-copy, a rendering block carries the private block's own L1 origin, so number,
	// timestamp, L1 origin and sequence number are all readable off the rendering by anyone; hash
	// comes from the field above. A private block's parent is a private block, and no public
	// artifact names it. Publishing it completes the six-field ref the follow protocol needs, which
	// is what lets the public supernode serve follow refs from public data alone and lets the
	// follower binary be deleted.
	//
	// It discloses nothing the field above does not: both name blocks of the same private chain, and
	// the parent of a range's terminal block is the terminal block of nothing — it is an interior
	// block whose contents stay private exactly as every other block's do. One more 32-byte hash per
	// cadence.
	PrivateTerminalParentHash common.Hash
	// L1Head is the L1 block the operator derived the range under.
	L1Head common.Hash
	// RollupConfigHash and DepSetHash pin which chain and which dependency set this claim
	// speaks for.
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	// PrivateDataHash is the content address of the range's full private derivation input. It is a
	// COMMITMENT, not a pointer: nothing publishes the object, which reaches every legitimate
	// reader over the operator's private p2p network. It appears exactly once on chain — here, in
	// the public record — which is what makes the claim the read-side authority for every object:
	// there is no second commitment anywhere that a reader could resolve instead.
	PrivateDataHash common.Hash
	// Proof fills the proof slot. It is EMPTY under attested mode (v2), where a non-empty slot is
	// refused outright rather than carried. The slot itself is unconditional and is the upgrade
	// path: a proving system fills it, and nothing else about the wire changes.
	Proof []byte
	// Writes is a canonical sorted set of opaque writes, including their last-write block.
	Writes []writes.Record
}

// Mode is the proof posture a decoder is configured for. Its zero value is the strict one.
type Mode uint8

const (
	// ModeAttested is the v2 posture: the claim's authority is the operator's signature on the
	// carrying transaction, there is no proof system, and a non-empty proof slot is therefore a
	// claim this verifier cannot evaluate. It refuses it. See the package comment.
	ModeAttested Mode = iota
	// ModeProven is for a verifier that has a proof system wired up, plus the tooling that must be
	// able to read a claim whatever its slot contains. It performs NO verification — checking
	// a proof is the caller's job and is not a codec concern — it only stops refusing.
	ModeProven
)

func (m Mode) String() string {
	switch m {
	case ModeAttested:
		return "attested"
	case ModeProven:
		return "proven"
	default:
		return fmt.Sprintf("mode(%d)", uint8(m))
	}
}

// claimArgs encodes RangeClaim as a single ABI value, which is what `abi.encode(claim)` in
// Solidity produces for a struct with a dynamic member: a head offset word followed by the tuple.
var claimArgs = abi.Arguments{{Type: claimType}}

// ClaimTupleType is the claim's canonical ABI tuple string, e.g.
// "(uint8,uint64,uint64,bytes32,...,bytes)".
//
// It exists so that NOBODY HAND-WRITES THE FIELD LIST AGAIN. The registry's function signature --
// and therefore the 4-byte selector every claim transaction is sent with -- is "postClaim(" + this
// + ")", and a hand-written copy of it is a second source of truth that a field addition silently
// desynchronises: the producer and every Go reader keep agreeing with each other while the CHAIN
// stops recognising the call at all. That is not hypothetical. Adding privateTerminalParentHash
// left a stale nine-field signature in op-private-interop/render, so the batcher sent selector
// 0x41a02b4d to a registry that answers 0x4db071ca; the call hit a contract with no fallback and
// every postClaim reverted, while the follow module -- which compared against the same stale
// constant -- decoded the very same transactions happily.
//
// Deriving it from the encoder's own type makes the two impossible to separate.
func ClaimTupleType() string { return claimType.String() }

// claimType is the tuple the encoder and decoder share.
var claimType = mustClaimType()

func mustClaimType() abi.Type {
	t, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "version", Type: "uint8"},
		{Name: "firstBlock", Type: "uint64"},
		{Name: "lastBlock", Type: "uint64"},
		{Name: "privateTerminalBlockHash", Type: "bytes32"},
		{Name: "privateTerminalParentHash", Type: "bytes32"},
		{Name: "l1Head", Type: "bytes32"},
		{Name: "rollupConfigHash", Type: "bytes32"},
		{Name: "depSetHash", Type: "bytes32"},
		{Name: "privateDataHash", Type: "bytes32"},
		{Name: "proof", Type: "bytes"},
		{Name: "writes", Type: "bytes"},
	})
	if err != nil {
		panic(fmt.Errorf("range claim v%d ABI type: %w", ClaimVersion, err))
	}
	return t
}

// abiRangeClaim mirrors the Solidity struct. Field names are the CamelCase of the ABI component
// names, which is how go-ethereum binds tuple components positionally.
type abiRangeClaim struct {
	Version                   uint8
	FirstBlock                uint64
	LastBlock                 uint64
	PrivateTerminalBlockHash  common.Hash
	PrivateTerminalParentHash common.Hash
	L1Head                    common.Hash
	RollupConfigHash          common.Hash
	DepSetHash                common.Hash
	PrivateDataHash           common.Hash
	Proof                     []byte
	Writes                    []byte
}

// Encode ABI-encodes a claim at the current version.
//
// It refuses an inverted range rather than emitting one, because a producer that can post bytes its
// own decoder rejects will eventually post them at 3am. See CheckStructure.
func Encode(e *RangeClaim) ([]byte, error) {
	if err := e.CheckStructure(); err != nil {
		return nil, err
	}
	if len(e.Proof) > MaxProofSize {
		return nil, fmt.Errorf("%w: proof is %d bytes, the maximum is %d", ErrProofTooLarge, len(e.Proof), MaxProofSize)
	}
	return encodeAtVersion(e, ClaimVersion)
}

// encodeAtVersion is Encode with the version byte lifted out. It is unexported and exists for one
// reason: the refusal corpus needs REAL encodings at wrong versions — bytes a mis-versioned
// producer would genuinely emit — rather than a valid encoding with a word poked, which tests a
// slightly different thing (see the version fixtures).
func encodeAtVersion(e *RangeClaim, version uint8) ([]byte, error) {
	proof := e.Proof
	if proof == nil {
		proof = []byte{}
	}
	writeData, err := writes.Encode(e.Writes)
	if err != nil {
		return nil, err
	}
	out, err := claimArgs.Pack(abiRangeClaim{
		Version:                   version,
		FirstBlock:                e.FirstBlock,
		LastBlock:                 e.LastBlock,
		PrivateTerminalBlockHash:  e.PrivateTerminalBlockHash,
		PrivateTerminalParentHash: e.PrivateTerminalParentHash,
		L1Head:                    e.L1Head,
		RollupConfigHash:          e.RollupConfigHash,
		DepSetHash:                e.DepSetHash,
		PrivateDataHash:           e.PrivateDataHash,
		Proof:                     proof,
		Writes:                    writeData,
	})
	if err != nil {
		return nil, fmt.Errorf("pack range claim: %w", err)
	}
	return out, nil
}

// Decode parses a claim in attested mode: exactly version 2, a non-inverted range, canonical
// ABI form, and an empty proof slot. It is the decoder a v2 verifier wants, and it is what the
// zero value of Mode selects.
func Decode(data []byte) (*RangeClaim, error) { return DecodeMode(data, ModeAttested) }

// DecodeMode parses a claim under an explicit proof posture.
//
// It accepts EXACTLY one version — never a set, never "1 or newer". A verifier's configuration
// pins the layout it accepts and a producer commits to one layout; a decoder that guessed would be
// applying an acceptance rule nobody configured, and a future field addition would be read out of
// the wrong offset by something that reported success. Rotating the wire is rotating a config, with
// two strict decoders in flight, not one lenient one.
//
// It also requires CANONICAL FORM: the bytes must be exactly what re-encoding the decoded value
// produces. ABI decoders are famously permissive — trailing data after the last field, dirty high
// bits above a uint8, a `bytes` member reached through a non-minimal offset — and each of those is
// a way for two readers of the same transaction to disagree about what it said. Here they are one
// rule with one error, and the check is a comparison rather than a checklist, so it cannot fall
// behind the format.
func DecodeMode(data []byte, mode Mode) (*RangeClaim, error) {
	if len(data) < EncodedSizeEmptyProof {
		return nil, fmt.Errorf("%w: %d bytes, need at least %d", ErrTruncated, len(data), EncodedSizeEmptyProof)
	}
	// Refused on length alone, before anything is unpacked or allocated. No canonical encoding of
	// a claim is this long, so there is nothing to learn from decoding it.
	if len(data) > MaxEncodedSize {
		return nil, fmt.Errorf("%w: %d bytes, no valid encoding exceeds %d", ErrProofTooLarge, len(data), MaxEncodedSize)
	}
	values, err := claimArgs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpack range claim: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("expected 1 ABI value in a range claim, got %d", len(values))
	}
	d := abi.ConvertType(values[0], new(abiRangeClaim)).(*abiRangeClaim)
	if d.Version != ClaimVersion {
		return nil, fmt.Errorf("%w: %d, this decoder accepts exactly %d", ErrBadVersion, d.Version, ClaimVersion)
	}
	// An absent proof decodes to an empty slice; normalise it to nil so a decoded claim compares
	// equal to the value that was encoded and re-encodes to the same bytes.
	proof := d.Proof
	if len(proof) == 0 {
		proof = nil
	}
	records, err := writes.Decode(d.Writes)
	if err != nil {
		return nil, err
	}
	e := &RangeClaim{
		FirstBlock:                d.FirstBlock,
		LastBlock:                 d.LastBlock,
		PrivateTerminalBlockHash:  d.PrivateTerminalBlockHash,
		PrivateTerminalParentHash: d.PrivateTerminalParentHash,
		L1Head:                    d.L1Head,
		RollupConfigHash:          d.RollupConfigHash,
		DepSetHash:                d.DepSetHash,
		PrivateDataHash:           d.PrivateDataHash,
		Proof:                     proof,
		Writes:                    records,
	}
	if err := e.CheckStructure(); err != nil {
		return nil, err
	}
	// The combined allocation bound does not replace the individual proof-size bound.
	if len(e.Proof) > MaxProofSize {
		return nil, fmt.Errorf("%w: proof is %d bytes, the maximum is %d", ErrProofTooLarge, len(e.Proof), MaxProofSize)
	}
	if mode == ModeAttested && len(e.Proof) != 0 {
		return nil, fmt.Errorf("%w: proof slot carries %d bytes", ErrProofNotEmpty, len(e.Proof))
	}
	canonical, err := encodeAtVersion(e, d.Version)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, data) {
		return nil, fmt.Errorf("%w: %d bytes decode to a value that re-encodes to %d",
			ErrNonCanonical, len(data), len(canonical))
	}
	return e, nil
}

// CheckStructure validates the range and its canonical write records, including
// that every record version lies inside the range.
//
// It is exported, and it is called by BOTH Encode and Decode, so there is no way to produce or
// accept a range that reads backwards. Everything else is deliberately absent, and each absent
// rule has a home:
//
//   - CONTIGUITY (FirstBlock continues the previous registered range; no range registered twice)
//     is REGISTRY policy, on chain, against the registry's own record. A codec has no "previous".
//   - TERMINAL-HASH TRUTH (PrivateTerminalBlockHash really is that block's hash) is off-chain
//     VERIFIER and TOOLING policy, because the EVM's 256-block blockhash lookback cannot reach a
//     cadence range boundary — the registry physically cannot see the block it would check.
//   - CHAIN AND L1 BINDING (L1Head is on this node's L1; the config hashes are the ones this node
//     was configured for) is the consumer's, which is the only party that knows what "this node"
//     means.
//
// A codec that guessed at any of these would be inventing an acceptance policy nobody configured.
func (e *RangeClaim) CheckStructure() error {
	if _, err := writes.Encode(e.Writes); err != nil {
		return err
	}
	for _, r := range e.Writes {
		if r.BlockNumber < e.FirstBlock || r.BlockNumber > e.LastBlock {
			return fmt.Errorf("%w: write outside range", writes.ErrInvalid)
		}
	}
	if e.LastBlock < e.FirstBlock {
		return fmt.Errorf("%w: firstBlock %d, lastBlock %d", ErrInvertedRange, e.FirstBlock, e.LastBlock)
	}
	return nil
}

// Blocks reports how many public blocks the claim covers, inclusive.
func (e *RangeClaim) Blocks() uint64 {
	if e.LastBlock < e.FirstBlock {
		return 0
	}
	return e.LastBlock - e.FirstBlock + 1
}
