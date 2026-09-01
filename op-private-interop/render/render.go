// Package render implements THE RENDERING TRANSFORMATION: the pure function that turns a private
// block into the content of the public block that stands for it.
//
// The architecture is op-private-interop/docs/DESIGN.md, sections "Canonical message positions"
// and "Replay transactions". A private L2's public presence is an
// ordinary derived OP chain — "the rendering" — with one public block per private block at the same
// number and timestamp, containing the stock L1-attributes deposit plus one batcher-signed REPLAY
// transaction per exported or imported message. This package is the single definition of which
// messages those are and what order they go in.
//
// # Why this is one function and not two
//
// The rendering's log positions are THE identity of the private chain's messages, for the
// counterparty judge, the interop filter, relayers, tooling, AND the operator's own supernode. If
// the builder that WRITES the rendering and the components that READ private blocks disagree by one
// index, the operator invalidates counterparties' perfectly valid executing messages. So there is
// exactly one implementation, and both sides call it in-process: the builder's write path
// (op-batcher), which renders the public blocks, and the operator's interop filter
// (op-interop-filter), which reads the RAW private node and applies RenderedLogs itself between
// fetching a block and inserting its entries into the LogsDB — rendered indices under the block's
// real private hash. No service wraps the private EL over RPC; there is nothing to keep in sync
// but this function.
//
// Everything here is PURE. No I/O, no clock, no map iteration order, no randomness: identical
// inputs must produce byte-identical output forever, because the batch payload posted to L1 is a
// function of this output and two operators (or the same operator twice) must produce the same
// bytes. The determinism test in this package is not a nicety; it is the consensus-critical gate.
//
// # The transformation
//
//	RenderedLogs(block, emitterSet) = the block's logs restricted to emitterSet,
//	                                  in their ORIGINAL order, re-indexed 0..k-1
//
// The emitter set is a set of (ADDRESS, TOPIC0) PAIRS (integrator ruling, superseding the
// address-only phrasing in the proposal):
//
//	{ (L2ToL2CrossDomainMessenger, SentMessage), (CrossL2Inbox, ExecutingMessage) }
//	  ∪ the genesis-configured extra emitters, at any topic
//
// The reason is that the set was always about which logs are CLAIMS. Under stock interop ANY log is
// a potential initiating message, so a log rendered at the wrong emitter address is not a harmless
// extra — it is a publicly consumable message with a broken identity. The case that forced the
// ruling is the messenger's RELAYEDMESSAGE, which the private chain emits on every import and which
// the replay messenger cannot emit (its relayMessage reverts Unsupported, because imports on the
// rendering execute directly against CrossL2Inbox). Rendering it through EventReplayer would put a
// messenger claim at the replayer's address; so it is EXCLUDED, which also saves a replay
// transaction and its gas on every import.
//
// Excluding it shifts the rendered index of every later log in an import block relative to an
// address-only rule. That is correct: the rule is the rule, and the rendered index is defined by
// this predicate and nothing else.
//
// Within a pair the policy is still TOTAL — every messenger message is exported, every import is
// public — so there is no per-message selection anywhere in this package. That is what makes the
// transformation checkable by anyone holding the private block. There is exactly ONE implementation
// of the predicate, EmitterSet.Renders, and every consumer calls it.
//
// # ORDER: the private block's own, interleaved
//
// Replay transactions go in RenderedLogs order — the private block's original log order, exports
// and imports interleaved exactly as they occurred — because the standing invariant is that
// RenderedLogs(private block) EQUALS the derived rendering block's log sequence. Any grouping
// rule breaks it: in a block that emits export, import, export, sorting by kind would move the
// second export from rendered index 2 to rendered index 1, and every consumer resolving that
// message by position would resolve a different one.
//
// # What a rendering block does NOT contain
//
// Nothing derived from execution. A pure function cannot produce the rendering's stateRoot, hence
// not its blockHash, hence not its receipts root. Those exist only once a node has executed the
// rendering, and the builder gets them from a rendering follower, never from here. This package
// also has no notion of a RANGE: the CLAIM transaction that leads each cadence range is a builder
// concern. It cannot shift any message's rendered index, because the ClaimRegistry is log-less —
// which is exactly why the claim is allowed to lead. A rendering block is otherwise the stock
// L1-attributes deposit plus one replay transaction per action.
package render

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// MaxRenderableMessageSize is the protocol ceiling for a private SentMessage payload. It is
	// mirrored by L2ToL2CrossDomainMessenger on the private half: checking it again here keeps the
	// rendering safe when reading historical or otherwise malformed private-chain data.
	MaxRenderableMessageSize = 64 * 1024
)

var (
	// ErrInconsistentBlock is returned when the private block's own parts disagree — a receipt per
	// transaction is the weakest consistency a caller can be asked for, and a caller that cannot
	// supply it is holding data from two different blocks.
	ErrInconsistentBlock = errors.New("inconsistent private block")
	// ErrUnrenderableLog is returned for a log that is IN the emitter set but that no replay
	// transaction can reproduce. It is deliberately fatal rather than skipped: skipping it would
	// silently renumber every later message in the block, which is the one failure this package
	// exists to make impossible.
	ErrUnrenderableLog = errors.New("emitter-set log cannot be rendered")
)

// EmitterSet decides which logs appear on the rendering.
//
// Its ZERO VALUE is the ratified default: the two interop predeploys at their two event topics, and
// nothing else. Extra emitters are a genesis-time configuration for chains that install additional
// replay emitters; they are not a per-message policy and must never become one.
type EmitterSet struct {
	extra map[common.Address]struct{}
}

// NewEmitterSet returns the standard pairs plus the given extra emitters. Duplicates and
// re-listings of the standard predeploys are harmless.
func NewEmitterSet(extra ...common.Address) EmitterSet {
	if len(extra) == 0 {
		return EmitterSet{}
	}
	m := make(map[common.Address]struct{}, len(extra))
	for _, a := range extra {
		m[a] = struct{}{}
	}
	return EmitterSet{extra: m}
}

// Renders is THE predicate. Every component that needs to know whether a private log is public
// calls this one function; a second implementation of it anywhere is a divergence waiting for a
// block that interleaves.
func (s EmitterSet) Renders(l *types.Log) bool {
	if l == nil {
		return false
	}
	switch l.Address {
	case predeploys.L2toL2CrossDomainMessengerAddr:
		return topic0(l) == SentMessageEventTopic
	case predeploys.CrossL2InboxAddr:
		return topic0(l) == messages.ExecutingMessageEventTopic
	}
	// A configured extra emitter renders at ANY topic: it exists precisely because its shapes are
	// not known here, and there is no protocol claim to get wrong.
	_, ok := s.extra[l.Address]
	return ok
}

// RenderedLog is one private log that survives the filter, carrying both positions: where it was on
// the private chain, and where it will be on the rendering.
//
// Log points at the CALLER's log and is never modified: consumers read RenderedLogIndex, they do not
// get a rewritten log.
type RenderedLog struct {
	Log *types.Log
	// PrivateLogIndex is the log's block-level index on the PRIVATE chain: its position in the
	// block's full log sequence, counting the logs this filter removed.
	PrivateLogIndex uint32
	// PrivateTxIndex is the index of the private transaction that emitted it.
	PrivateTxIndex uint32
	// RenderedLogIndex is the log's block-level index on the RENDERING: its rank among the logs
	// that survived, 0..k-1. This is the message's canonical log index — the number that goes in
	// every Identifier, on every chain, in every component.
	RenderedLogIndex uint32
}

// RenderedLogs is the normative primitive: the logs restricted to the emitter set, in their
// original interleaved order, re-indexed from zero.
//
// logs MUST be the block's COMPLETE log sequence in block order. The private index is taken from
// the slice position and not from Log.Index, because the position IS the definition of a
// block-level log index and trusting a caller-populated field would give this function two sources
// of truth for the one number it exists to compute.
//
// Returns nil (not an empty slice) for a block with no emitter-set logs, so that a block with no
// messages renders to nothing at all.
func RenderedLogs(logs []*types.Log, set EmitterSet) []RenderedLog {
	var out []RenderedLog
	for i, l := range logs {
		if !set.Renders(l) {
			continue
		}
		out = append(out, RenderedLog{
			Log:              l,
			PrivateLogIndex:  uint32(i),
			PrivateTxIndex:   uint32(l.TxIndex),
			RenderedLogIndex: uint32(len(out)),
		})
	}
	return out
}

// ReplayKind distinguishes the two replay transactions.
type ReplayKind uint8

const (
	// ReplayExport re-emits a message the private chain SENT, through
	// L2ToL2CrossDomainMessengerReplay.replaySentMessage at the messenger predeploy address. The
	// emitted event is byte-identical to the private one, so every stock consumer sees what it
	// expects at the address it expects.
	ReplayExport ReplayKind = iota
	// ReplayImport executes a message the private chain RECEIVED, by calling the stock
	// CrossL2Inbox.validateMessage with the standard checksum access list. A lying operator's
	// imports are still caught by the counterparty's own message database — this is the half of
	// the v1 trust model that does not rest on attestation.
	ReplayImport
	// ReplayEvent re-emits a configured EXTRA EMITTER's log through EventReplayer.replayEvent.
	//
	// EventReplayer emits at its OWN address, so a ReplayEvent log's emitter on the rendering is the
	// replayer's address rather than the private emitter's. That is exactly why no protocol claim
	// may be routed here — see the emitter-set ruling in the package doc — and why extra emitters,
	// which carry no claim, can be.
	ReplayEvent
)

func (k ReplayKind) String() string {
	switch k {
	case ReplayExport:
		return "export"
	case ReplayImport:
		return "import"
	case ReplayEvent:
		return "event"
	default:
		return fmt.Sprintf("replayKind(%d)", uint8(k))
	}
}

// ReplayAction is one rendering transaction's worth of intent: everything a ReplayTxBuilder needs
// to construct the signed transaction, and nothing about how it is signed, priced or nonced.
//
// One action produces exactly one transaction which emits exactly one log, which is why an action's
// RenderedLogIndex is also its position among the block's replay transactions.
type ReplayAction struct {
	RenderedLog
	Kind ReplayKind
	// Topics and Data are the private log's payload. They are set for every kind — a ReplayEvent
	// re-emits them verbatim, and for the other two they are what the resulting rendering log must
	// match, which makes an action checkable against the derived rendering without a second lookup.
	Topics []common.Hash
	Data   []byte
	// Export is the SentMessage decoded from the private messenger log. Non-nil exactly when Kind
	// is ReplayExport.
	Export *SentMessage
	// Import is the executing message decoded from the private CrossL2Inbox log. Non-nil exactly
	// when Kind is ReplayImport.
	Import *messages.Message
}

// PrivateBlock is the input to RenderBlock: one private block's header, transactions and receipts.
//
// Transactions are taken but NOT consumed. The rendering's content is a function of the receipts
// alone — which is worth stating, because it is why the rendering can exist without the private
// chain's transactions ever leaving the operator. They are here only so the block's own consistency
// can be checked, and the check is skipped when Txs is nil: a caller reporting on an EL response
// may hold transaction hashes rather than bodies, and inventing placeholder transactions to satisfy
// a consistency check would defeat the check. The builder always supplies them.
type PrivateBlock struct {
	Header   *types.Header
	Txs      types.Transactions
	Receipts types.Receipts
	// Ref is the private block's own L2BlockRef. Supply it whenever the rendering is being BUILT:
	// the rendering block copies this block's L1 origin and sequence number verbatim, and the range
	// claim publishes its hash and parent hash, so all six fields are load-bearing.
	//
	// It is a whole ref rather than a hash because a caller who holds an eth.ExecutionPayload rather
	// than a full header (the batcher does) cannot produce any of it from the header alone — the
	// origin and sequence number live in the L1-info deposit, and a hash re-derived from a partial
	// header is no block's hash. derive.PayloadToBlockRef produces exactly this value.
	//
	// A caller that only wants a block's RENDERED LOGS — the filter's ingestion path — leaves it
	// zero: it is reporting on a private block, not building a rendering, and needs none of these
	// fields.
	Ref eth.L2BlockRef
}

// RenderedBlock is the content of the rendering block that stands for one private block.
//
// It is content only. Number and Timestamp are the private block's, because the chains are
// block-for-block; PrivateRef identifies the SOURCE and is never the rendering's own ref, which
// no pure function can know.
type RenderedBlock struct {
	Number    uint64
	Timestamp uint64
	// PrivateRef is the private block this renders, as supplied. Its L1Origin is the epoch the
	// rendering block MUST carry — origins are copied, never chosen — and its Hash and ParentHash
	// are what a range claim publishes for its terminal block.
	//
	// When the caller supplied no Ref (the filter's ingestion path, which is reporting on a private
	// block rather than building a rendering), only Hash, ParentHash, Number and Time are filled,
	// from the header; L1Origin and SequenceNumber stay zero, because a header alone cannot say
	// what its origin was. Builders must supply a Ref, and the builder refuses a zero origin.
	PrivateRef eth.L2BlockRef
	// Logs is the rendering block's log sequence: RenderedLogs of the private block.
	Logs []RenderedLog
	// Actions is the ordered list of replay transactions. Actions[i] renders Logs[i]; the two
	// slices always have the same length, and TestActionsMatchLogs pins it.
	Actions []ReplayAction
}

// RenderBlock is the block-level transformation: a private block in, the rendering block's replay
// actions and log sequence out.
//
// It is a pure function of its inputs and must stay one. It reads no config beyond the emitter set,
// consults no chain, and returns the same value for the same input every time.
func RenderBlock(b PrivateBlock, set EmitterSet) (*RenderedBlock, error) {
	if b.Header == nil {
		return nil, fmt.Errorf("%w: no header", ErrInconsistentBlock)
	}
	if b.Txs != nil && len(b.Receipts) != len(b.Txs) {
		return nil, fmt.Errorf("%w: %d transactions but %d receipts", ErrInconsistentBlock, len(b.Txs), len(b.Receipts))
	}
	num := b.Header.Number.Uint64()

	// The block's full log sequence, in block order: receipts in transaction order, each receipt's
	// logs in emission order. This is the sequence whose positions are private log indexes.
	var all []*types.Log
	for _, r := range b.Receipts {
		if r == nil {
			return nil, fmt.Errorf("%w: nil receipt", ErrInconsistentBlock)
		}
		for _, l := range r.Logs {
			if l == nil {
				return nil, fmt.Errorf("%w: nil log", ErrInconsistentBlock)
			}
			// Cheap and worth having: catches receipts fetched for the wrong block, which is a
			// real operational hazard and otherwise produces a plausible-looking wrong rendering.
			if l.BlockNumber != 0 && l.BlockNumber != num {
				return nil, fmt.Errorf("%w: log from block %d in block %d", ErrInconsistentBlock, l.BlockNumber, num)
			}
			all = append(all, l)
		}
	}

	logs := RenderedLogs(all, set)
	out := &RenderedBlock{
		Number:     num,
		Timestamp:  b.Header.Time,
		PrivateRef: b.privateRef(),
		Logs:       logs,
	}
	if len(logs) == 0 {
		return out, nil
	}
	out.Actions = make([]ReplayAction, 0, len(logs))
	for _, rl := range logs {
		act, err := actionFor(rl)
		if err != nil {
			return nil, err
		}
		out.Actions = append(out.Actions, act)
	}
	return out, nil
}

// privateRef is the block's own ref: the caller's, when it supplied one, and otherwise as much of
// one as a header can give. The distinction is not a convenience — a caller holding an execution
// payload rather than a full header would otherwise get the hash of a PARTIAL header, which is no
// block's hash at all, and both the copied origin and the range claim depend on this value.
func (b PrivateBlock) privateRef() eth.L2BlockRef {
	if b.Ref != (eth.L2BlockRef{}) {
		return b.Ref
	}
	return eth.L2BlockRef{
		Hash:       b.Header.Hash(),
		ParentHash: b.Header.ParentHash,
		Number:     b.Header.Number.Uint64(),
		Time:       b.Header.Time,
	}
}

// actionFor classifies one log that Renders already admitted, so the switch is total by
// construction: the inbox pair is an import, the messenger pair is an export, and anything left is
// a configured extra emitter.
//
// A log admitted by its pair whose PAYLOAD does not decode is fatal rather than skipped. Skipping it
// would renumber every later message in the block, which is the one failure this package exists to
// make impossible; a build that halts is a stall the operator can see.
func actionFor(rl RenderedLog) (ReplayAction, error) {
	act := ReplayAction{
		RenderedLog: rl,
		Topics:      append([]common.Hash(nil), rl.Log.Topics...),
		Data:        append([]byte(nil), rl.Log.Data...),
		Kind:        ReplayEvent,
	}
	unrenderable := func(err error) (ReplayAction, error) {
		return ReplayAction{}, fmt.Errorf("%w: private log index %d: %w", ErrUnrenderableLog, rl.PrivateLogIndex, err)
	}
	switch rl.Log.Address {
	case predeploys.CrossL2InboxAddr:
		msg, err := messages.MessageFromLog(rl.Log)
		if err != nil {
			return unrenderable(err)
		}
		if msg == nil {
			// Topic0 matched but the log is malformed for it — a wrong topic count, say.
			return unrenderable(errors.New("an ExecutingMessage log that does not decode"))
		}
		act.Kind = ReplayImport
		act.Import = msg
	case predeploys.L2toL2CrossDomainMessengerAddr:
		sent, err := DecodeSentMessage(rl.Log.Topics, rl.Log.Data)
		if err != nil {
			return unrenderable(err)
		}
		if sent.Sender == predeploys.SuperchainETHBridgeAddr {
			return unrenderable(errors.New("SentMessage sender is the SuperchainETHBridge"))
		}
		if sent.Target == predeploys.SuperchainETHBridgeAddr {
			return unrenderable(errors.New("SentMessage target is the SuperchainETHBridge"))
		}
		if len(sent.Message) > MaxRenderableMessageSize {
			return unrenderable(fmt.Errorf(
				"SentMessage payload is %d bytes, exceeding the %d-byte rendering limit",
				len(sent.Message), MaxRenderableMessageSize,
			))
		}
		act.Kind = ReplayExport
		act.Export = sent
	}
	return act, nil
}

func topic0(l *types.Log) common.Hash {
	if len(l.Topics) == 0 {
		return common.Hash{}
	}
	return l.Topics[0]
}
