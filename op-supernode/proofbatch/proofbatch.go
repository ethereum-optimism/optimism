// Package proofbatch implements the Keccak Cove proof-batch wire format: the object a private
// chain's prover posts to L1 blobs so that a verifier which does NOT derive that chain can still
// cross-safe its messages.
//
// The authoritative spec is keccak-cove/SPEC-PROOF-BATCH.md. Two implementations build against it
// (this one and the Rust prover-side codec in rust/kona/sp1/crates/proof-batch), so everything here
// is byte-exact by construction: the public values are the ABI encoding the SP1 program commits
// to, and the envelope framing is fixed-width big-endian. The Rust side owns the golden fixtures;
// fixtures_test.go reads that one copy rather than mirroring it.
//
//	magic     "KCPB"          4 bytes
//	version   0x03            1 byte
//	pv_len    uint32 BE       4 bytes
//	public_values             pv_len bytes (ABI-encoded ProofBatch)
//	proof_len uint32 BE       4 bytes
//	proof                     proof_len bytes (the proving system's proof; EMPTY in attested mode)
//
// # What v3 changed, and why (G7)
//
// v3 adds ONE field: each block's ExecMsgs, the executing messages that block consumed. v2 said
// what a private chain EXPORTED and nothing about what it IMPORTED, which left its cross-chain
// reads proof-trusted — the guest asserted them and the public network took its word. With the
// import list on the wire the claim becomes conditional and checkable ("P's STF is correct AND the
// only cross-chain facts it assumed are exactly these"), so the public cross-safety machinery
// validates a proven chain's dependencies exactly like a driven chain's.
//
// Both versions are decodable here, and that is deliberate: a v2→v3 rotation is a vkey rotation, so
// a verifier must be configurable to accept exactly one of them (see DecodeAs) rather than to guess.
package proofbatch

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// Magic prefixes every envelope. It is not security-relevant: authenticity comes from the L1
	// sender/recipient pair. It exists so that a blob which is not a proof batch at all is
	// rejected before any ABI decoding is attempted.
	Magic = "KCPB"
	// Version is the current envelope version: the one this codec ENCODES, and the one a verifier
	// accepts unless it is configured otherwise. Any field change in ProofBatch bumps it.
	Version = 3
	// VersionV2 is the version v3 replaced: exports only, no import list, so a v2 verifier's view of
	// a proven chain's dependencies is empty rather than checked. It stays DECODABLE, because the
	// rotation from v2 to v3 is a real event with two live configurations in it, and it stays a
	// named constant so that accepting it is always a deliberate configuration rather than an
	// off-by-one in a comparison.
	VersionV2 = 2
	// VersionV1 carried no L2 block hash, no state roots and positional log indices; it is rejected
	// rather than supported, because a v1 batch says nothing about the facts a v2 verifier seals.
	VersionV1 = 1
	// VersionV3 is the first version that carries a per-block IMPORT LIST, and it is a named constant
	// distinct from Version for a reason that is easy to get wrong and expensive to get wrong.
	//
	// "Does this wire version carry execMsgs" is a FIXED fact about version 3, not a fact about
	// whatever version this codec currently encodes. Written as `version >= Version` it silently
	// changes meaning the day Version becomes 4: the v3 ABI type would lose its execMsgs field, and a
	// v3 batch's import list would be recorded as UNKNOWN — a node configured for v3, accepting v3
	// batches, validating no dependencies. That is precisely the fail-open Fact.ExecMsgsKnown exists
	// to prevent, reintroduced by a comparison against a moving constant.
	//
	// Use VersionHasExecMsgs, never a comparison against Version.
	VersionV3 = 3

	headerLen = len(Magic) + 1 + 4 // magic ‖ version ‖ pv_len

	// addressPrefixLen is the address prefix of a log preimage; see LogExport.Preimage.
	addressPrefixLen = common.AddressLength
)

// ExportPolicyAllHashes is the v2 default export policy: the prover exports every log, as a
// (block-level log index, log hash) pair with no preimage. Curated policies — a subset of the
// logs, or logs with their preimages attached — are a different policy hash and a config change
// on both sides, not a wire change: that is why the index and preimage fields ship in v2 even
// though the default policy leaves the preimage empty.
var ExportPolicyAllHashes = crypto.Keccak256Hash([]byte("cove-export-v2:all-hashes"))

var (
	ErrBadMagic     = errors.New("not a proof-batch envelope")
	ErrBadVersion   = errors.New("unsupported proof-batch version")
	ErrTruncated    = errors.New("truncated proof-batch envelope")
	ErrTrailingData = errors.New("trailing data after proof-batch envelope")
	ErrBadPreimage  = errors.New("log preimage does not hash to its log hash")
)

// LogExport is one exported initiating-message log: where it sits in its block, what it hashes
// to, and — only under a policy that includes them — the bytes it hashes from.
type LogExport struct {
	// Index is the log's index within its block, as the EVM assigns it. It is carried
	// explicitly rather than implied by position so that a curated export policy (a subset of a
	// block's logs) still lets an executing message reference this chain's REAL log indices.
	Index uint32
	// Hash is the messages.LogToLogHash value of the log.
	Hash common.Hash
	// Preimage is the log's content: its 20-byte emitting address followed by
	// messages.LogToMessagePayload (the topics concatenated, then the data). It is empty unless
	// the export policy includes preimages. When non-empty it MUST hash to Hash, which is
	// checked in-guest and again at decode (see checkLogs), so a preimage-bearing batch cannot
	// carry content that disagrees with the hash the proof committed to.
	Preimage []byte
}

// LogHashFromPreimage recomputes a log hash from an exported preimage. It is the decode-side half
// of the rule that a preimage may never disagree with its hash.
//
// The preimage is deliberately the plainest encoding that determines the hash — address ‖ topics ‖
// data, no framing — because the two implementations of this format must agree on it byte for
// byte, and a concatenation cannot drift the way a serialization format can. The topic/data split
// is not recoverable from it, and does not need to be: nothing in the acceptance path reads a log
// as anything but its hash.
func LogHashFromPreimage(preimage []byte) (common.Hash, error) {
	if len(preimage) < addressPrefixLen {
		return common.Hash{}, fmt.Errorf("preimage is %d bytes, need at least %d for the emitting address",
			len(preimage), addressPrefixLen)
	}
	addr := common.BytesToAddress(preimage[:addressPrefixLen])
	payloadHash := crypto.Keccak256Hash(preimage[addressPrefixLen:])
	return messages.PayloadHashToLogHash(payloadHash, addr), nil
}

// ExecMsg is one executing message a block consumed: the interop Identifier it named and the
// message hash it claimed for that identifier. It is exactly interop's own `messages.Message`
// (`{Identifier, PayloadHash}`), which is also the content of the CrossL2Inbox `ExecutingMessage`
// event — so the wire carries the message and nothing about the transaction that executed it.
//
// That omission is the design, not a simplification (PLAN G7, minimal-leak): no executing-tx hash,
// index, sender or count appears on the wire. The full disclosure is the EDGE — (this block, that
// message) — at block granularity, which is also all the cross-safety judge needs, because every
// invariant it checks is a function of the identifier and the message hash.
//
// Two consequences of carrying no position, both load-bearing:
//
//   - the set must carry its own canonical ORDER, or two provers with the same import list would
//     commit to different bytes. See SortKey and CheckStructure;
//   - a block's executing messages cannot be placed among its logs, so they never enter the
//     positional same-timestamp dependency graph. That is why an execMsg's timestamp must be
//     STRICTLY below its block's — see CheckStructure and G7G D2.
type ExecMsg struct {
	messages.Message
}

// execMsgKeyLen is the width of an ExecMsg's canonical sort key: its six fields, one 32-byte
// big-endian word each.
const execMsgKeyLen = 6 * 32

// SortKey is the ExecMsg's canonical byte key: the total order the wire's deduplicated set is
// sorted by, and the only thing that makes "a set" a byte-exact object.
//
// It is the 192-byte ABI ENCODING of the six fields in DECLARATION order — each field in one 32-byte
// big-endian word, origin left-padded (SPEC-WIRE-V3 D17):
//
//	origin(32, left-padded) ‖ blockNumber(32) ‖ logIndex(32) ‖ timestamp(32) ‖ chainId(32) ‖ msgHash(32)
//
// Two properties make this the right key rather than a tighter packing. Lexicographic order over the
// ABI words coincides with field-by-field big-endian unsigned comparison in declaration order, so
// "sort the bytes" and "sort the fields" are the same instruction — there is no second list of sort
// columns for two implementations to keep in sync. And the key is a serialization both languages
// already have, rather than a bespoke one written twice.
//
// (This side originally packed the fields at their wire widths with chainId leading, which sorted
// differently and would have produced different BYTES for the same set. The canonical rule is the
// spec's; see G7G D6 for the reconciliation.)
//
// The key covers the WHOLE message, not the identifier alone. Two entries may legitimately share an
// identifier and differ in msgHash — one executing transaction referencing a log honestly and
// another referencing the same position with a wrong hash is a thing a block can contain, and it is
// precisely what the judge exists to catch. Deduplicating on the identifier would silently drop the
// bad one, which is the assumption a consolidator exists to reject.
func (e *ExecMsg) SortKey() [execMsgKeyLen]byte {
	var key [execMsgKeyLen]byte
	copy(key[32-common.AddressLength:32], e.Identifier.Origin.Bytes())
	binary.BigEndian.PutUint64(key[64-8:64], e.Identifier.BlockNumber)
	binary.BigEndian.PutUint32(key[96-4:96], e.Identifier.LogIndex)
	binary.BigEndian.PutUint64(key[128-8:128], e.Identifier.Timestamp)
	chainID := e.Identifier.ChainID.Bytes32()
	copy(key[128:160], chainID[:])
	copy(key[160:192], e.PayloadHash.Bytes())
	return key
}

// Executing is the message as the cross-safety judge consumes it: the identifier's coordinates plus
// the checksum derived from (identifier, msgHash).
//
// The derivation is the binding. A verifier never takes a checksum off the wire — it computes one
// from the identifier and the message hash the proof committed to, exactly as CrossL2Inbox does from
// what an executor supplies, so a wire that named a real log with a wrong hash produces a checksum
// that matches nothing in the initiating chain's database.
func (e *ExecMsg) Executing() *messages.ExecutingMessage {
	return &messages.ExecutingMessage{
		ChainID:   e.Identifier.ChainID,
		BlockNum:  e.Identifier.BlockNumber,
		LogIdx:    e.Identifier.LogIndex,
		Timestamp: e.Identifier.Timestamp,
		Checksum:  e.Checksum(),
	}
}

// ExecMsgsFromLogs is the canonical extraction: one block's logs in, its wire import list out.
//
// It is the Go side of SPEC-WIRE-V3 §4, and it exists as one function because there must be exactly
// one answer to "what does this block import" for every Go producer — a harness that extracted its
// own way would be testing agreement between two of this repo's opinions rather than agreement with
// the format.
//
// THREE RULES, and the third is the one worth stating:
//
//  1. FILTER. Only CrossL2Inbox `ExecutingMessage` events count, decided by
//     messages.MessageFromLog — the same decoder the supervisor's own ingestion uses. A log that is
//     not one decodes to nil and is skipped, which is how a block's exports and imports come off one
//     pass over the same receipts.
//  2. CANONICALISE. Sort by SortKey and drop exact repeats. A real step, not a formality: receipt
//     order is transaction order, the wire's order is key order, and two executing transactions may
//     legitimately consume the identical message — one edge, not two.
//  3. A MALFORMED CrossL2Inbox EVENT IS AN ABORT, NEVER A SKIP. If a log carries the right address
//     and the right topic but its body does not decode, that is not "no import here" — it is a block
//     whose import list this code cannot determine, and continuing would silently commit to a set
//     that omits a message the block really consumed. That omission is exactly what the conditional
//     validity claim forbids. Pinned identically on the Rust side (G7R D4, which deliberately adopted
//     the stricter of the two available decoders).
func ExecMsgsFromLogs(logs []*types.Log) ([]ExecMsg, error) {
	var out []ExecMsg
	for i, l := range logs {
		msg, err := messages.MessageFromLog(l)
		if err != nil {
			return nil, fmt.Errorf("logs[%d] is a CrossL2Inbox executing message that does not decode, so "+
				"this block's import list cannot be determined: %w", i, err)
		}
		if msg == nil {
			// nil means "not an executing message" — but the shared decoder reaches that verdict for
			// two very different reasons, and one of them must not be a silent skip. It checks the
			// TOPIC COUNT before the topic identity, so a log at the CrossL2Inbox carrying the
			// ExecutingMessage topic with the wrong number of topics returns nil rather than an error.
			// That is a malformed event, not an absent one, and skipping it would commit to an import
			// list omitting a message the block may really have consumed — the exact omission rule 3
			// exists to forbid. Unreachable from a stock predeploy (the event has exactly two topics),
			// which is why it has to be checked here rather than relied upon.
			if looksLikeExecutingMessage(l) {
				return nil, fmt.Errorf("logs[%d] is emitted by the CrossL2Inbox with the ExecutingMessage "+
					"topic but %d topics, which is not a well-formed event, so this block's import list "+
					"cannot be determined", i, len(l.Topics))
			}
			continue
		}
		out = append(out, ExecMsg{Message: *msg})
	}
	return CanonicaliseExecMsgs(out), nil
}

// looksLikeExecutingMessage reports whether a log CLAIMS to be an executing message — right emitter,
// right event topic — regardless of whether it is well formed enough to decode.
//
// It exists so that "the decoder said no" can be split into "this is not an executing message" and
// "this is a broken executing message". Only the first is a skip.
func looksLikeExecutingMessage(l *types.Log) bool {
	return l != nil &&
		l.Address == predeploys.CrossL2InboxAddr &&
		len(l.Topics) > 0 &&
		l.Topics[0] == messages.ExecutingMessageEventTopic
}

// CanonicaliseExecMsgs sorts an import list into canonical order and drops exact repeats, returning
// what must reach the wire. It is separate from ExecMsgsFromLogs so a producer that obtained its
// messages some other way still has one place to get the ordering right.
func CanonicaliseExecMsgs(msgs []ExecMsg) []ExecMsg {
	if len(msgs) == 0 {
		return nil
	}
	sorted := slices.Clone(msgs)
	slices.SortFunc(sorted, func(a, b ExecMsg) int {
		ka, kb := a.SortKey(), b.SortKey()
		return bytes.Compare(ka[:], kb[:])
	})
	out := sorted[:1]
	for _, m := range sorted[1:] {
		prev, cur := out[len(out)-1].SortKey(), m.SortKey()
		if !bytes.Equal(prev[:], cur[:]) {
			out = append(out, m)
		}
	}
	return out
}

// BlockExport is one L2 block's contribution to a proof batch: its identity (number, timestamp,
// real L2 block hash), the two roots that make its output root derivable, the initiating messages
// the prover exported for it, and (from v3) the executing messages it consumed.
type BlockExport struct {
	Number    uint64
	Timestamp uint64
	// Hash is the REAL L2 block hash. It is what a verifier seals the block under, so a
	// proof-carried chain's LogsDB is keyed exactly like a driven chain's.
	Hash common.Hash
	// StateRoot is the block's post-state root.
	StateRoot common.Hash
	// MessagePasserStorageRoot is the storage root of the L2ToL1MessagePasser predeploy at this
	// block.
	MessagePasserStorageRoot common.Hash
	// Logs are the exported logs in ascending Index order. May be empty (a block with no
	// exported messages).
	Logs []LogExport
	// ExecMsgs are the executing messages this block consumed, as a DEDUPLICATED SET in canonical
	// SortKey order (v3 and above; nil under v2, which said nothing about imports at all).
	//
	// This is the import list, and it is what turns the proof's claim conditional: "this block's
	// STF is correct AND the only cross-chain facts it assumed are exactly these". Nil versus empty
	// is therefore a distinction that must not be lost downstream — an empty set is the claim "this
	// block consumed nothing", which is checkable, while nil is the absence of any claim. The
	// version that produced the batch is what tells them apart, and it travels on Envelope.Version.
	ExecMsgs []ExecMsg
}

// OutputRoot is the block's output root, derived from the three committed roots exactly as
// op-node derives it: keccak256(0x0 ‖ stateRoot ‖ messagePasserStorageRoot ‖ blockHash).
//
// This is what makes v2 worth a version bump beyond honest hashes: a verifier that never executes
// the chain can answer "what is this chain's output root at block N / timestamp T" for every block
// in proven history, which is what a shared superroot needs.
func (b *BlockExport) OutputRoot() common.Hash {
	return common.Hash(eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(b.StateRoot),
		MessagePasserStorageRoot: eth.Bytes32(b.MessagePasserStorageRoot),
		BlockHash:                b.Hash,
	}))
}

// ProofBatch is the public values of a range proof: what the prover claims, and everything a
// verifier needs to bind that claim to its own view of L1.
//
// Batch and deposit binding come from the in-circuit derivation itself — the proof derives the
// range from L1 data under L1Head — so the only bindings left for a verifier are that L1Head is
// really on its L1, and that the chain the prover derived (RollupConfigHash, DepSetHash) and the
// filter it exported under (ExportPolicyHash) are the ones this node was configured for.
type ProofBatch struct {
	PrevOutputRoot   common.Hash
	NewOutputRoot    common.Hash
	L1Head           common.Hash
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	ExportPolicyHash common.Hash
	Blocks           []BlockExport
}

// CheckStructure enforces the invariants a proof batch must satisfy on its own, independent of
// any verifier's state: it exports at least one block, its blocks are contiguous, their timestamps
// advance, and each block's exported logs are in ascending index order without repeats. The
// head-relative rules (which block it must start at, which output root it must extend) belong to
// the verifier.
//
// Log indices are checked for strict increase rather than contiguity on purpose: a curated export
// policy legitimately skips indices, and the whole point of carrying the index explicitly is that
// a gap is data rather than a defect. A repeat, by contrast, is always a defect — two hashes
// claiming the same position.
func (b *ProofBatch) CheckStructure() error {
	if len(b.Blocks) == 0 {
		return errors.New("blocks[] is empty")
	}
	for i, blk := range b.Blocks {
		if i > 0 {
			if want := b.Blocks[i-1].Number + 1; blk.Number != want {
				return fmt.Errorf("blocks[%d] number %d, expected %d", i, blk.Number, want)
			}
			if blk.Timestamp <= b.Blocks[i-1].Timestamp {
				return fmt.Errorf("blocks[%d] timestamp %d does not advance past %d", i, blk.Timestamp, b.Blocks[i-1].Timestamp)
			}
		}
		for j, l := range blk.Logs {
			if j > 0 && l.Index <= blk.Logs[j-1].Index {
				return fmt.Errorf("block %d logs[%d] index %d does not advance past %d",
					blk.Number, j, l.Index, blk.Logs[j-1].Index)
			}
		}
		if err := checkExecMsgs(&b.Blocks[i]); err != nil {
			return err
		}
	}
	return nil
}

// ErrSameTimestampImport is returned by CheckNoSameTimestampImports. It is its own error because the
// rule is a restriction on what a proven chain may DO, not a statement about whether the bytes are
// well formed, and an operator meeting it deserves to find the reason (G7G D2).
var ErrSameTimestampImport = errors.New("a proven chain may not consume a same-timestamp message")

// checkExecMsgs enforces the one rule that makes an import list a well-formed wire object: canonical
// order, strictly increasing by SortKey.
//
// The wire object is a SET, and a set has to be serialised in some order for two implementations to
// commit to the same bytes; "sorted, no repeats" is that order and the deduplication in one
// predicate. A repeat is a defect either way — consuming the same message twice is one edge, not two.
//
// Deliberately NOT here: the same-timestamp rule. See CheckNoSameTimestampImports for where it lives
// and why it is not a structural property.
func checkExecMsgs(blk *BlockExport) error {
	var prev [execMsgKeyLen]byte
	for i := range blk.ExecMsgs {
		msg := &blk.ExecMsgs[i]
		key := msg.SortKey()
		if i > 0 && bytes.Compare(key[:], prev[:]) <= 0 {
			return fmt.Errorf("block %d execMsgs[%d] is not strictly after its predecessor in canonical order "+
				"(duplicate or unsorted import list)", blk.Number, i)
		}
		prev = key
	}
	return nil
}

// CheckNoSameTimestampImports requires every imported message to be stamped STRICTLY BELOW its
// consuming block. It is a VERIFIER ACCEPTANCE RULE, not part of CheckStructure.
//
// THE RULE. Stock interop allows equality — the same-timestamp fixpoint is a real feature — and this
// refuses it, for a structural reason rather than a conservative one. Resolving same-timestamp
// dependencies needs each executing message's POSITION in its block, so the cycle graph can order
// them (`buildCycleGraph`, `executingMessageBefore`), and wire v3 deliberately carries no position.
// Admitting the class with invented positions could hide a real cycle or invent a false one, so the
// class is refused instead — which makes the honest claim "a proven chain contributes no
// same-timestamp executing messages" true by construction rather than by convention. The Rust lane
// reached the same conclusion independently and from the other side (G7R D10, `SameTimestampEdgeRefused`
// in the consolidation kernel), so the guest never PRODUCES one and a verifier never ACCEPTS one.
//
// The strictly-greater case, which stock interop treats as invalid (ErrTimestampViolation), is
// refused by the same comparison.
//
// WHY IT IS NOT IN CheckStructure, which is where this side first put it. CheckStructure answers "is
// this a well-formed instance of the wire format", and the canonical fixture corpus is the authority
// on that — it contains `exec-msgs-max-widths`, a VALID case whose import is stamped at `u64::MAX`
// specifically to pin the field widths. A codec that refused it would be a codec disagreeing with
// the format's own definition, and would have failed the byte-identity gate for a reason that has
// nothing to do with bytes. The rule belongs with the other things a NODE requires of a batch —
// block-time spacing, config binding, L1-head depth — all of which are equally not the codec's
// business. Same failure mode either way: the batch is rejected, the proven head does not move, and
// the prover can post a correct one. That recoverable failure is the whole point of refusing here
// rather than letting the judge find proven history invalid (G7G D3).
func (b *ProofBatch) CheckNoSameTimestampImports() error {
	for i := range b.Blocks {
		blk := &b.Blocks[i]
		for j := range blk.ExecMsgs {
			if ts := blk.ExecMsgs[j].Identifier.Timestamp; ts >= blk.Timestamp {
				return fmt.Errorf("%w: block %d (timestamp %d) execMsgs[%d] is stamped %d",
					ErrSameTimestampImport, blk.Number, blk.Timestamp, j, ts)
			}
		}
	}
	return nil
}

// Envelope is a decoded proof-batch blob payload: the claim plus the proof of it.
type Envelope struct {
	// Version is the envelope version byte this payload carried. It is retained because it is what
	// distinguishes "this block imported nothing" from "this wire version says nothing about
	// imports" — see BlockExport.ExecMsgs.
	Version uint8
	Batch   ProofBatch
	// Proof is the proving system's proof bytes. It is EMPTY under attested mode (v1), where the
	// batch's authority is the operator's signature on the L1 transaction that carried it, and a
	// verifier in that mode requires it to be empty. The slot itself is unconditional and is the
	// upgrade path: a proving system fills it, and nothing else about the wire changes.
	Proof []byte
	// PublicValues is the exact ABI encoding the proof commits to. It is retained verbatim
	// because proof verification hashes these bytes: re-encoding Batch could differ from what
	// the prover committed if the two codecs ever disagree, and a proof must be checked against
	// what was actually posted.
	PublicValues []byte
}

// abiLogExport / abiExecMsg / abiBlockExport / abiProofBatch mirror the Solidity structs in the
// spec. Field names are the CamelCase of the ABI names, which is how go-ethereum's ABI codec binds
// tuple components.
//
// There is ONE set of Go structs for both wire versions, with the v2 blocks carrying an ExecMsgs
// field that the v2 ABI type does not mention. go-ethereum binds tuple components positionally
// against the type, so a field the type omits is simply not visited — which is why the two versions
// share a decoder body and differ only in the abi.Type they are unpacked against.
type abiLogExport struct {
	LogIndex uint32
	LogHash  common.Hash
	Preimage []byte
}

// abiIdentifier is interop's Identifier, at the wire's integer widths rather than Solidity's.
//
// CrossL2Inbox declares every numeric field as uint256, and this narrows blockNumber/logIndex/
// timestamp to the widths v2 already uses for exactly those three quantities. Consistency inside the
// object beats consistency with a Solidity struct that never appears in it, and the narrowing is
// lossless for all three (they are uint64/uint32 everywhere in the node too). chainId stays uint256,
// because interop chain IDs really are 256-bit.
type abiIdentifier struct {
	Origin      common.Address
	BlockNumber uint64
	LogIndex    uint32
	Timestamp   uint64
	ChainId     *big.Int //nolint:revive,staticcheck // ABI component name is chainId
}

type abiExecMsg struct {
	Id      abiIdentifier //nolint:revive,staticcheck // ABI component name is id
	MsgHash common.Hash
}

type abiBlockExport struct {
	BlockNumber              uint64
	Timestamp                uint64
	BlockHash                common.Hash
	StateRoot                common.Hash
	MessagePasserStorageRoot common.Hash
	Logs                     []abiLogExport
	ExecMsgs                 []abiExecMsg
}

type abiProofBatch struct {
	PrevOutputRoot   common.Hash
	NewOutputRoot    common.Hash
	L1Head           common.Hash
	RollupConfigHash common.Hash
	DepSetHash       common.Hash
	ExportPolicyHash common.Hash
	Blocks           []abiBlockExport
}

// proofBatchArgsV2 / proofBatchArgsV3 encode ProofBatch as a single ABI value, which is what
// `abi.encode(proofBatch)` in Solidity and `ProofBatch::abi_encode()` in alloy produce: a head
// offset word followed by the tuple body.
var (
	proofBatchArgsV2 = abi.Arguments{{Type: mustProofBatchType(VersionV2)}}
	proofBatchArgsV3 = abi.Arguments{{Type: mustProofBatchType(Version)}}
)

// argsFor returns the ABI arguments for a wire version, refusing any version this codec does not
// implement. It is the single place a version becomes a layout, so adding v4 is one case here.
func argsFor(version uint8) (abi.Arguments, error) {
	switch version {
	case VersionV2:
		return proofBatchArgsV2, nil
	case Version:
		return proofBatchArgsV3, nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrBadVersion, version)
	}
}

// VersionHasExecMsgs reports whether a wire version carries a per-block import list.
//
// This is the ONE predicate every consumer must ask, rather than comparing against Version. See
// VersionV3 for what goes wrong otherwise.
func VersionHasExecMsgs(version uint8) bool { return version >= VersionV3 }

// CheckVersion reports whether this codec implements a wire version. It is exported so a
// configuration can be refused at load rather than at the first blob.
func CheckVersion(version uint8) error {
	_, err := argsFor(version)
	return err
}

// DecodeAny parses envelope bytes at WHATEVER version they declare, provided this codec implements
// it. Envelope.Version says which.
//
// This is for DIAGNOSTICS and for validating a payload somebody else framed — never for acceptance.
// A verifier must use DecodeAs with its configured version, because for a verifier the version is
// part of the acceptance rule: it decides whether a proven chain's dependencies are checked or
// trusted (see DecodeAs). A tool that inspects blobs has the opposite need — being unable to read the
// version the live chain is actually running is exactly when an inspector is worthless.
func DecodeAny(data []byte) (*Envelope, error) {
	if len(data) < headerLen {
		return nil, fmt.Errorf("%w: %d bytes", ErrTruncated, len(data))
	}
	if string(data[:len(Magic)]) != Magic {
		return nil, fmt.Errorf("%w: magic %x", ErrBadMagic, data[:len(Magic)])
	}
	return DecodeAs(data, data[len(Magic)])
}

func mustProofBatchType(version uint8) abi.Type {
	blockComponents := []abi.ArgumentMarshaling{
		{Name: "blockNumber", Type: "uint64"},
		{Name: "timestamp", Type: "uint64"},
		{Name: "blockHash", Type: "bytes32"},
		{Name: "stateRoot", Type: "bytes32"},
		{Name: "messagePasserStorageRoot", Type: "bytes32"},
		{Name: "logs", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
			{Name: "logIndex", Type: "uint32"},
			{Name: "logHash", Type: "bytes32"},
			{Name: "preimage", Type: "bytes"},
		}},
	}
	if VersionHasExecMsgs(version) {
		blockComponents = append(blockComponents, abi.ArgumentMarshaling{
			Name: "execMsgs", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
				{Name: "id", Type: "tuple", Components: []abi.ArgumentMarshaling{
					{Name: "origin", Type: "address"},
					{Name: "blockNumber", Type: "uint64"},
					{Name: "logIndex", Type: "uint32"},
					{Name: "timestamp", Type: "uint64"},
					{Name: "chainId", Type: "uint256"},
				}},
				{Name: "msgHash", Type: "bytes32"},
			},
		})
	}
	t, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "prevOutputRoot", Type: "bytes32"},
		{Name: "newOutputRoot", Type: "bytes32"},
		{Name: "l1Head", Type: "bytes32"},
		{Name: "rollupConfigHash", Type: "bytes32"},
		{Name: "depSetHash", Type: "bytes32"},
		{Name: "exportPolicyHash", Type: "bytes32"},
		{Name: "blocks", Type: "tuple[]", Components: blockComponents},
	})
	if err != nil {
		panic(fmt.Errorf("proof-batch v%d ABI type: %w", version, err))
	}
	return t
}

// EncodePublicValues ABI-encodes a proof batch at the current wire version: the bytes the SP1
// program commits to.
func EncodePublicValues(b *ProofBatch) ([]byte, error) {
	return EncodePublicValuesAs(b, Version)
}

// EncodePublicValuesAs ABI-encodes a proof batch at an explicit wire version. Encoding a batch that
// carries an import list at a version with no field for it is refused rather than silently dropping
// it: a v2 encoding of a v3 batch would be a strictly weaker claim wearing the same roots.
func EncodePublicValuesAs(b *ProofBatch, version uint8) ([]byte, error) {
	args, err := argsFor(version)
	if err != nil {
		return nil, err
	}
	blocks := make([]abiBlockExport, len(b.Blocks))
	for i, blk := range b.Blocks {
		if !VersionHasExecMsgs(version) && len(blk.ExecMsgs) > 0 {
			return nil, fmt.Errorf("cannot encode block %d's %d executing messages at wire version %d, "+
				"which has no field for them", blk.Number, len(blk.ExecMsgs), version)
		}
		logs := make([]abiLogExport, len(blk.Logs))
		for j, l := range blk.Logs {
			preimage := l.Preimage
			if preimage == nil {
				preimage = []byte{}
			}
			logs[j] = abiLogExport{LogIndex: l.Index, LogHash: l.Hash, Preimage: preimage}
		}
		execMsgs := make([]abiExecMsg, len(blk.ExecMsgs))
		for j, m := range blk.ExecMsgs {
			execMsgs[j] = abiExecMsg{
				Id: abiIdentifier{
					Origin:      m.Identifier.Origin,
					BlockNumber: m.Identifier.BlockNumber,
					LogIndex:    m.Identifier.LogIndex,
					Timestamp:   m.Identifier.Timestamp,
					ChainId:     m.Identifier.ChainID.ToBig(),
				},
				MsgHash: m.PayloadHash,
			}
		}
		blocks[i] = abiBlockExport{
			BlockNumber:              blk.Number,
			Timestamp:                blk.Timestamp,
			BlockHash:                blk.Hash,
			StateRoot:                blk.StateRoot,
			MessagePasserStorageRoot: blk.MessagePasserStorageRoot,
			Logs:                     logs,
			ExecMsgs:                 execMsgs,
		}
	}
	return args.Pack(abiProofBatch{
		PrevOutputRoot:   b.PrevOutputRoot,
		NewOutputRoot:    b.NewOutputRoot,
		L1Head:           b.L1Head,
		RollupConfigHash: b.RollupConfigHash,
		DepSetHash:       b.DepSetHash,
		ExportPolicyHash: b.ExportPolicyHash,
		Blocks:           blocks,
	})
}

// DecodePublicValues decodes ABI-encoded public values at the current wire version, rejecting any
// exported preimage that does not hash to the log hash beside it.
func DecodePublicValues(data []byte) (*ProofBatch, error) {
	return DecodePublicValuesAs(data, Version)
}

// DecodePublicValuesAs decodes ABI-encoded public values at an explicit wire version.
func DecodePublicValuesAs(data []byte, version uint8) (*ProofBatch, error) {
	args, err := argsFor(version)
	if err != nil {
		return nil, err
	}
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpack public values: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("expected 1 ABI value, got %d", len(values))
	}
	decoded := abi.ConvertType(values[0], new(abiProofBatch)).(*abiProofBatch)
	blocks := make([]BlockExport, len(decoded.Blocks))
	for i, blk := range decoded.Blocks {
		logs := make([]LogExport, len(blk.Logs))
		for j, l := range blk.Logs {
			// An absent preimage decodes to nil rather than to an empty slice, so that a decoded
			// batch compares equal to the batch that was encoded and re-encodes to the same bytes.
			preimage := l.Preimage
			if len(preimage) == 0 {
				preimage = nil
			}
			logs[j] = LogExport{Index: l.LogIndex, Hash: l.LogHash, Preimage: preimage}
		}
		// Same reason as the preimage: an empty import list decodes to nil, so a decoded batch
		// re-encodes to the bytes it came from. The "did this version carry imports at all"
		// question is answered by the version, never by nil-versus-empty.
		var execMsgs []ExecMsg
		if len(blk.ExecMsgs) > 0 {
			execMsgs = make([]ExecMsg, len(blk.ExecMsgs))
			for j, m := range blk.ExecMsgs {
				if m.Id.ChainId == nil {
					return nil, fmt.Errorf("block %d execMsgs[%d] has no chain id", blk.BlockNumber, j)
				}
				execMsgs[j] = ExecMsg{Message: messages.Message{
					Identifier: messages.Identifier{
						Origin:      m.Id.Origin,
						BlockNumber: m.Id.BlockNumber,
						LogIndex:    m.Id.LogIndex,
						Timestamp:   m.Id.Timestamp,
						ChainID:     eth.ChainIDFromBig(m.Id.ChainId),
					},
					PayloadHash: m.MsgHash,
				}}
			}
		}
		blocks[i] = BlockExport{
			Number:                   blk.BlockNumber,
			Timestamp:                blk.Timestamp,
			Hash:                     blk.BlockHash,
			StateRoot:                blk.StateRoot,
			MessagePasserStorageRoot: blk.MessagePasserStorageRoot,
			Logs:                     logs,
			ExecMsgs:                 execMsgs,
		}
		if err := checkPreimages(&blocks[i]); err != nil {
			return nil, err
		}
	}
	return &ProofBatch{
		PrevOutputRoot:   decoded.PrevOutputRoot,
		NewOutputRoot:    decoded.NewOutputRoot,
		L1Head:           decoded.L1Head,
		RollupConfigHash: decoded.RollupConfigHash,
		DepSetHash:       decoded.DepSetHash,
		ExportPolicyHash: decoded.ExportPolicyHash,
		Blocks:           blocks,
	}, nil
}

// checkPreimages enforces the one rule a preimage-bearing export must satisfy. The proof commits
// to the hash, so a preimage that hashes to something else is content the proof never made a claim
// about; decoding it would put unproven bytes into a verifier that believes everything it decoded
// was proven.
func checkPreimages(blk *BlockExport) error {
	for _, l := range blk.Logs {
		if len(l.Preimage) == 0 {
			continue
		}
		got, err := LogHashFromPreimage(l.Preimage)
		if err != nil {
			return fmt.Errorf("%w: block %d log %d: %w", ErrBadPreimage, blk.Number, l.Index, err)
		}
		if got != l.Hash {
			return fmt.Errorf("%w: block %d log %d hashes to %s, not %s",
				ErrBadPreimage, blk.Number, l.Index, got, l.Hash)
		}
	}
	return nil
}

// Encode frames a proof batch and its proof into envelope bytes at the current wire version.
func Encode(b *ProofBatch, proof []byte) ([]byte, error) {
	return EncodeAs(b, proof, Version)
}

// EncodeAs frames a proof batch at an explicit wire version. It exists for the rotation: a
// dual-running submitter posts the same history to two inboxes under two vkeys, and a fixture
// generator must be able to produce the version it is pinning.
func EncodeAs(b *ProofBatch, proof []byte, version uint8) ([]byte, error) {
	pv, err := EncodePublicValuesAs(b, version)
	if err != nil {
		return nil, err
	}
	return EncodeWithPublicValuesAs(pv, proof, version), nil
}

// EncodeWithPublicValues frames already-encoded public values at the current wire version, for a
// caller that received the exact bytes a proof was generated over.
func EncodeWithPublicValues(publicValues []byte, proof []byte) []byte {
	return EncodeWithPublicValuesAs(publicValues, proof, Version)
}

// EncodeWithPublicValuesAs frames already-encoded public values under an explicit version byte.
func EncodeWithPublicValuesAs(publicValues []byte, proof []byte, version uint8) []byte {
	out := make([]byte, 0, headerLen+len(publicValues)+4+len(proof))
	out = append(out, Magic...)
	out = append(out, version)
	out = binary.BigEndian.AppendUint32(out, uint32(len(publicValues)))
	out = append(out, publicValues...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(proof)))
	return append(out, proof...)
}

// Decode parses envelope bytes, accepting only the current wire version.
func Decode(data []byte) (*Envelope, error) {
	return DecodeAs(data, Version)
}

// DecodeAs parses envelope bytes, accepting EXACTLY the given wire version.
//
// One version, not a set, and that is the config story for the rotation. A verifier's vkey pins the
// guest, and the guest pins the version it commits to, so a node that accepted both would be
// claiming a flexibility its proof verification does not have — and would silently apply the weaker
// v2 dependency posture to a chain the operator believes is running v3. Two versions in flight means
// two configured verifiers, each strict, which is also what makes a dark launch comparable.
//
// It is strict about length too: every length must be exactly consumed, so a blob that decodes at
// all decodes to exactly one envelope.
func DecodeAs(data []byte, accept uint8) (*Envelope, error) {
	if _, err := argsFor(accept); err != nil {
		return nil, fmt.Errorf("cannot accept wire version %d: %w", accept, err)
	}
	if len(data) < headerLen {
		return nil, fmt.Errorf("%w: %d bytes", ErrTruncated, len(data))
	}
	if string(data[:len(Magic)]) != Magic {
		return nil, fmt.Errorf("%w: magic %x", ErrBadMagic, data[:len(Magic)])
	}
	version := data[len(Magic)]
	if version != accept {
		return nil, fmt.Errorf("%w: %d, this node accepts %d", ErrBadVersion, version, accept)
	}
	pvLen := binary.BigEndian.Uint32(data[len(Magic)+1:])
	rest := data[headerLen:]
	if uint64(len(rest)) < uint64(pvLen)+4 {
		return nil, fmt.Errorf("%w: public values length %d exceeds %d remaining bytes", ErrTruncated, pvLen, len(rest))
	}
	pv := rest[:pvLen]
	rest = rest[pvLen:]
	proofLen := binary.BigEndian.Uint32(rest)
	rest = rest[4:]
	if uint64(len(rest)) < uint64(proofLen) {
		return nil, fmt.Errorf("%w: proof length %d exceeds %d remaining bytes", ErrTruncated, proofLen, len(rest))
	}
	if uint64(len(rest)) > uint64(proofLen) {
		return nil, fmt.Errorf("%w: %d bytes", ErrTrailingData, uint64(len(rest))-uint64(proofLen))
	}
	batch, err := DecodePublicValuesAs(pv, version)
	if err != nil {
		return nil, err
	}
	return &Envelope{Version: version, Batch: *batch, Proof: rest[:proofLen], PublicValues: pv}, nil
}

// ToBlobs packs envelope bytes into blobs using the op-stack blob encoding. A payload larger than
// one blob is split across blobs, which concatenate in sidecar index order on the way back.
func ToBlobs(data []byte) ([]*eth.Blob, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot pack an empty payload")
	}
	var blobs []*eth.Blob
	for len(data) > 0 {
		chunk := data
		if len(chunk) > eth.MaxBlobDataSize {
			chunk = chunk[:eth.MaxBlobDataSize]
		}
		blob := new(eth.Blob)
		if err := blob.FromData(chunk); err != nil {
			return nil, fmt.Errorf("encode blob %d: %w", len(blobs), err)
		}
		blobs = append(blobs, blob)
		data = data[len(chunk):]
	}
	return blobs, nil
}

// FromBlobs concatenates the blobs of one transaction back into envelope bytes.
func FromBlobs(blobs []*eth.Blob) ([]byte, error) {
	var out []byte
	for i, blob := range blobs {
		if blob == nil {
			return nil, fmt.Errorf("blob %d is missing", i)
		}
		data, err := blob.ToData()
		if err != nil {
			return nil, fmt.Errorf("decode blob %d: %w", i, err)
		}
		out = append(out, data...)
	}
	return out, nil
}
