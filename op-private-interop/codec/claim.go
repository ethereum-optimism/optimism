// Package codec implements the one wire format Private Interop needs: the RANGE CLAIM, which the
// operator publishes as an ordinary L2 transaction on the public chain.
//
// The architecture is op-private-interop/docs/DESIGN.md. This package owns nothing but the bytes.
// It holds no chain state, reads no config, and has no opinion
// about whether a well-formed claim is a TRUE one — which is what makes it safe for the batching
// service, the claim-follower, an auditor's tool and a future proof program to share.
//
// # Where the claim lives
//
// A range's claim is the LEADING transaction of that range, and it describes ITS OWN range. Range
// N opens by announcing what range N is going to be; there is no lag, no off-by-one, and no
// special case at the start of the chain — range 0 opens with range 0's claim, exactly like every
// range after it.
//
// The thing that makes announcing your own range possible rather than circular is
// privateTerminalBlockHash: it names the PRIVATE chain's block at lastBlock, and the private chain
// is always ahead of its public rendering. When the operator builds range N's leading block, the
// private blocks that range N will render have already been produced, so their terminal hash is a
// past fact being published — not a prediction, and not a value that depends on the transaction
// carrying it. The public terminal hash could not do this job: it is a function of a range that
// includes the claim transaction itself.
//
// There is deliberately no parent-claim hash. The public chain's own block linkage plus the
// registry's contiguity check provide the ordering, and a hash-linked list over the same ranges the
// chain already links would be a second, weaker copy of something the chain establishes for free.
//
// # The wire
//
// Pure ABI. There is no magic, no length prefix and no hand-rolled framing: the claim is the
// argument of a contract call, so the ABI is already the framing and a second one on top would only
// be a second thing to disagree about.
//
//	struct RangeClaim {
//	    uint8   version;                  // exactly 1
//	    uint64  firstBlock;               // the range this claim describes, inclusive
//	    uint64  lastBlock;
//	    bytes32 privateTerminalBlockHash;  // the PRIVATE chain's block hash at lastBlock
//	    bytes32 privateTerminalParentHash; // and that block's parent hash
//	    bytes32 l1Head;
//	    bytes32 rollupConfigHash;
//	    bytes32 depSetHash;
//	    bytes32 privateDataHash;          // content address of the range's full private input
//	    bytes   proof;                    // EMPTY in attested mode
//	}
//
// The struct has a dynamic member, so abi.encode(claim) is the leading offset word 0x20, then the
// tuple: ten head words (nine statics and the offset to `proof`), then the proof's length word
// and its padded bytes. An empty-proof claim is therefore 384 bytes, whatever the range's size —
// v1 carries no per-block data at all.
//
// # What this package does NOT own
//
// The registry binding. The claim reaches the chain as a single tuple argument —
//
//	postClaim(RangeClaim calldata claim)
//
// — where the call's calldata is a 4-byte selector followed by exactly what Encode produces. This
// package owns the STRUCT VALUE; the selector, the function name, and the registry's storage layout
// belong with the binding, which is where a change to any of them is a change to a contract ABI
// rather than to a wire format.
//
// THE REGISTRY EMITS NO EVENT, by ratified design. A claim leads its range, so a rendering-only log
// at the front of a range-opening block would shift every message index in that block — the public
// chain's whole point is that its receipts are ordinary, and a log that exists solely to announce
// the rendering would make range-opening blocks differ in shape from every other block. The durable
// record is therefore the CALLDATA itself, plus a storage hash-chain the registry exposes through a
// getter. A reader scans transactions addressed to the registry and reads the chain via that
// getter; it does not filter logs.
//
// That makes the strict decoder load-bearing rather than a courtesy. Calldata is whatever the
// caller put there — nothing re-encodes it on the way in — so "these bytes are the canonical
// encoding of exactly one claim" is a property a reader must CHECK, not one it may assume. It is
// the reason DecodeMode insists on canonical form: without it, two readers of the same transaction
// could decode the same value from different bytes, and the hash chain would agree with only one of
// them. TestSolidityProducesTheSameBytes pins this package's idea of that canonical form against
// solc's real output, so the encoder that fills the calldata and the decoder that reads it back
// cannot drift apart.
//
// Two acceptance rules are also NOT here, and it is worth naming where they live:
//
//   - CONTIGUITY — that a range starts where the previous one ended, and that no range is
//     registered twice — is REGISTRY policy, enforced on chain against the registry's own record of
//     the last range. A codec has no notion of "previous".
//   - PRIVATE-TERMINAL-HASH TRUTH — that privateTerminalBlockHash really is the private chain's
//     block hash at lastBlock — is off-chain VERIFIER and TOOLING policy, and it could not be
//     anything else: the public chain's EVM cannot see the private chain at all, and even for a
//     public hash the 256-block blockhash lookback would not reach a cadence boundary.
//
// # The attested-mode rule
//
// The proof slot is unconditional on the wire and empty in v1, where a claim's authority is the
// operator's signature on the L2 transaction carrying it. A verifier configured for attested mode
// MUST REFUSE a non-empty proof slot rather than ignore it — a verifier that accepts what it cannot
// check has a hole exactly the size of the thing it skipped, and "there is a proof here" is
// precisely the assertion an attested verifier is not equipped to evaluate. This is the standing v1
// rule inherited from the proof-batch wire, and ModeAttested is the ZERO VALUE of Mode so that a
// caller who thinks about none of this gets the strict decoder.
package codec

import (
	"bytes"
	"errors"
	"fmt"

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
	ClaimVersion uint8 = 1

	// EncodedSizeEmptyProof is the length of an attested (empty-proof) claim encoding: the
	// outer offset word, the head words, and the proof's length word.
	//
	// It is also the minimum length of ANY valid encoding, which is why it is the first thing
	// Decode checks.
	EncodedSizeEmptyProof = (headWords + 2) * 32

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

	// MaxEncodedSize is the largest a valid encoding can be: an empty-proof claim plus a
	// maximum proof, padded up to a whole number of words. It lets Decode reject an absurd input
	// on its length alone, before any decoding allocates anything.
	MaxEncodedSize = EncodedSizeEmptyProof + MaxProofSize

	// headWords is the tuple's head: nine static fields plus one offset word for `proof`.
	headWords = 10
	// proofOffset is the byte offset of the proof's length word, measured from the start of the
	// TUPLE (that is, from the word after the outer offset) — the value the ninth head word must
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
	// the right price for a chain of claims that means anything. Nothing else about the private
	// chain's blocks reaches the public record.
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
	// Proof fills the proof slot. It is EMPTY under attested mode (v1), where a non-empty slot is
	// refused outright rather than carried. The slot itself is unconditional and is the upgrade
	// path: a proving system fills it, and nothing else about the wire changes.
	Proof []byte
}

// Mode is the proof posture a decoder is configured for. Its zero value is the strict one.
type Mode uint8

const (
	// ModeAttested is the v1 posture: the claim's authority is the operator's signature on the
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
	})
	if err != nil {
		return nil, fmt.Errorf("pack range claim: %w", err)
	}
	return out, nil
}

// Decode parses a claim in attested mode: exactly version 1, a non-inverted range, canonical
// ABI form, and an empty proof slot. It is the decoder a v1 verifier wants, and it is what the
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
	}
	if err := e.CheckStructure(); err != nil {
		return nil, err
	}
	// The length guard above rejects any encoding long enough to CARRY an over-cap proof, so this
	// is unreachable through a canonical encoding. It stays because it is the rule — the guard is
	// an optimisation over it, not a replacement for it — and because it is what a caller
	// constructing a value by hand and re-encoding it will hit.
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

// CheckStructure enforces the one invariant a claim must satisfy on its own, with no reference
// to any consumer's state: the covered range must be non-empty and non-inverted.
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
