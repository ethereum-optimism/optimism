package proofbatch

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The import list's ARITHMETIC, pinned against the canonical vectors rather than against this
// package's own opinion.
//
// TestFixtures already checks every envelope byte for byte. What this file adds is the part of the
// contract that is computation rather than layout: from an executing message's five identifier fields
// and its message hash, both implementations must derive the same logHash, the same checksum and the
// same canonical ordering key. A field-order agreement that disagreed about the checksum would accept
// batches the judge then failed to resolve — the worst available failure, because it looks like a
// missing dependency rather than a codec bug.

type execMsgVectorFile struct {
	// Algorithm is prose for a human — nested and mixed-shape — not something to assert. Held as raw
	// JSON so a change in how the generator words its documentation can never redden this gate.
	Algorithm      json.RawMessage `json:"algorithm"`
	EventTopic     common.Hash     `json:"eventTopic"`
	EventSignature string          `json:"eventSignature"`
	CrossL2Inbox   common.Address  `json:"crossL2Inbox"`
	Vectors        []execMsgVector `json:"vectors"`
	Ordering       struct {
		Note      string             `json:"note"`
		Raw       []execMsgVectorMsg `json:"raw"`
		Canonical []execMsgVectorMsg `json:"canonical"`
	} `json:"ordering"`
}

type execMsgVector struct {
	Name    string           `json:"name"`
	Source  string           `json:"source"`
	ExecMsg execMsgVectorMsg `json:"execMsg"`
}

type execMsgVectorMsg struct {
	Origin      common.Address `json:"origin"`
	BlockNumber uint64         `json:"blockNumber"`
	LogIndex    uint32         `json:"logIndex"`
	Timestamp   uint64         `json:"timestamp"`
	ChainID     string         `json:"chainId"`
	MsgHash     common.Hash    `json:"msgHash"`
	LogHash     common.Hash    `json:"logHash"`
	Checksum    common.Hash    `json:"checksum"`
	Key         hexutil.Bytes  `json:"key"`
}

// build turns a vector into this side's ExecMsg, so everything asserted afterwards is derived by
// production code from the six wire fields alone.
func (m *execMsgVectorMsg) build(t *testing.T) ExecMsg {
	t.Helper()
	n, ok := new(big.Int).SetString(m.ChainID, 10)
	require.Truef(t, ok, "vector chainId %q is not a decimal integer", m.ChainID)
	return ExecMsg{Message: messages.Message{
		Identifier: messages.Identifier{
			Origin:      m.Origin,
			BlockNumber: m.BlockNumber,
			LogIndex:    m.LogIndex,
			Timestamp:   m.Timestamp,
			ChainID:     eth.ChainIDFromBig(n),
		},
		PayloadHash: m.MsgHash,
	}}
}

// check asserts the three derived values a vector publishes.
func (m *execMsgVectorMsg) check(t *testing.T, where string) {
	t.Helper()
	msg := m.build(t)

	// THE SIMPLIFICATION THAT MAKES THE JUDGE FLIP CHEAP (SPEC-WIRE-V3 §6): the first step of
	// CrossL2Inbox.calculateChecksum is keccak256(origin ‖ msgHash), which is byte-identical to wire
	// v2's LogExport.logHash. So the identifier plus the message hash already determine the value the
	// initiating chain's log database is keyed by — one comparison discharges both stock per-message
	// checks, and no log PREIMAGE is needed anywhere on the verification path. That is why a curated
	// export policy and a proven chain's imports can coexist.
	if m.LogHash != (common.Hash{}) {
		require.Equal(t, m.LogHash,
			messages.PayloadHashToLogHash(msg.PayloadHash, msg.Identifier.Origin),
			"%s: logHash must be keccak256(origin ‖ msgHash)", where)
	}
	if m.Checksum != (common.Hash{}) {
		require.Equal(t, messages.MessageChecksum(m.Checksum), msg.Executing().Checksum,
			"%s: checksum", where)
	}
	if len(m.Key) > 0 {
		key := msg.SortKey()
		require.Equal(t, []byte(m.Key), key[:], "%s: canonical 192-byte ordering key", where)
	}
}

// TestExecMsgVectors is the arithmetic gate, including the one vector that was verified against the
// DEPLOYED predeploy.
func TestExecMsgVectors(t *testing.T) {
	requireFixtures(t)
	var f execMsgVectorFile
	readFixtureJSON(t, "exec-msg-checksum-vectors.json", &f)
	require.NotEmpty(t, f.Vectors, "the vector file carries no vectors")

	// The event identity, so the two sides agree about WHICH log is an executing message before they
	// agree about what it hashes to.
	if f.EventTopic != (common.Hash{}) {
		require.Equal(t, messages.ExecutingMessageEventTopic, f.EventTopic,
			"the ExecutingMessage event topic must be the one this node's decoder matches on")
	}
	if f.CrossL2Inbox != (common.Address{}) {
		require.Equal(t, predeploys.CrossL2InboxAddr, f.CrossL2Inbox,
			"the CrossL2Inbox address must be the one this node's decoder matches on")
	}

	live := false
	for _, v := range f.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			v.ExecMsg.check(t, v.Name)
			t.Logf("%s: %s", v.Name, v.Source)
		})
		if v.ExecMsg.Checksum != (common.Hash{}) {
			live = true
		}
	}
	require.True(t, live, "no vector pinned a checksum, so this gate proved nothing about the arithmetic")
}

// TestExecMsgOrderingVector pins the canonical order against the generator's own worked example:
// `raw` is an authored extraction in APPEARANCE order containing an exact duplicate, `canonical` is
// what must reach the wire.
//
// This is the strongest available check on the ordering rule, because it is the rule applied to data
// this side did not choose — and appearance order is exactly what a naive implementation would post.
func TestExecMsgOrderingVector(t *testing.T) {
	requireFixtures(t)
	var f execMsgVectorFile
	readFixtureJSON(t, "exec-msg-checksum-vectors.json", &f)
	if len(f.Ordering.Raw) == 0 || len(f.Ordering.Canonical) == 0 {
		t.Skip("the vector file publishes no ordering example")
	}
	require.Less(t, len(f.Ordering.Canonical), len(f.Ordering.Raw),
		"the example must actually dedup something, or it does not exercise the rule")

	// Sort-and-dedup the raw extraction by this side's key, exactly as a submitter must.
	raw := make([]ExecMsg, 0, len(f.Ordering.Raw))
	for i := range f.Ordering.Raw {
		f.Ordering.Raw[i].check(t, "ordering.raw")
		raw = append(raw, f.Ordering.Raw[i].build(t))
	}
	slices.SortFunc(raw, func(a, b ExecMsg) int {
		ka, kb := a.SortKey(), b.SortKey()
		return bytes.Compare(ka[:], kb[:])
	})
	got := raw[:1]
	for _, m := range raw[1:] {
		prev, cur := got[len(got)-1].SortKey(), m.SortKey()
		if !bytes.Equal(prev[:], cur[:]) {
			got = append(got, m)
		}
	}

	require.Len(t, got, len(f.Ordering.Canonical),
		"sorting and deduplicating the raw extraction did not produce the canonical set's length")
	for i, want := range f.Ordering.Canonical {
		want.check(t, "ordering.canonical")
		wantMsg := want.build(t)
		require.Equal(t, wantMsg, got[i], "canonical execMsgs[%d]", i)
	}

	// The result is what the codec accepts, and the RAW order is not — which is the operational point
	// of the rule rather than a restatement of it.
	blk := BlockExport{Number: 1, Timestamp: 1 << 40, ExecMsgs: got}
	require.NoError(t, (&ProofBatch{Blocks: []BlockExport{blk}}).CheckStructure())
	if len(raw) > 1 {
		unsorted := BlockExport{Number: 1, Timestamp: 1 << 40, ExecMsgs: raw}
		require.Error(t, (&ProofBatch{Blocks: []BlockExport{unsorted}}).CheckStructure(),
			"the raw extraction still contains its duplicate, so the codec must refuse it")
	}
}

// TestChecksumPackingTransposition is the trap the vector file names explicitly, kept as its own test
// because it is the one mistake that produces a plausible-looking wrong answer.
//
// CrossL2Inbox packs the checksum preimage as (blockNumber, TIMESTAMP, logIndex) — timestamp before
// log index — while the Identifier struct, and therefore the canonical ordering key, declares them
// (blockNumber, logIndex, TIMESTAMP). An implementation that used one order for the other purpose
// would still produce 32 well-formed bytes. So: swapping the two values must change BOTH the checksum
// and the key, and must not collide.
func TestChecksumPackingTransposition(t *testing.T) {
	base := testExecMsg()
	swapped := base
	swapped.Identifier.LogIndex = uint32(base.Identifier.Timestamp)
	swapped.Identifier.Timestamp = uint64(base.Identifier.LogIndex)

	require.NotEqual(t, base.Executing().Checksum, swapped.Executing().Checksum,
		"transposing logIndex and timestamp must not collide in the checksum")
	bk, sk := base.SortKey(), swapped.SortKey()
	require.NotEqual(t, bk, sk, "transposing logIndex and timestamp must not collide in the ordering key")

	// And the two orders really are different: the key ranks by logIndex before timestamp, so a pair
	// that disagrees on both ranks the way the KEY says, not the way the checksum's packing would.
	lowIdxHighTS, highIdxLowTS := base, base
	lowIdxHighTS.Identifier.LogIndex, lowIdxHighTS.Identifier.Timestamp = 1, 9
	highIdxLowTS.Identifier.LogIndex, highIdxLowTS.Identifier.Timestamp = 9, 1
	kLo, kHi := lowIdxHighTS.SortKey(), highIdxLowTS.SortKey()
	require.Negative(t, bytes.Compare(kLo[:], kHi[:]),
		"the ordering key must rank by logIndex before timestamp (declaration order), not by the "+
			"checksum's packing order")
}

// TestExecMsgsFromLogs pins the extraction rule: the filter, the canonicalisation, and — the one that
// matters most — that a malformed CrossL2Inbox event is an ABORT, never a skip.
func TestExecMsgsFromLogs(t *testing.T) {
	// execLog builds a real ExecutingMessage event the way the chain emits one: the message hash in
	// topic 1, the identifier ABI-encoded in the data.
	execLog := func(id messages.Identifier, msgHash common.Hash) *types.Log {
		data := make([]byte, 0, 5*32)
		data = append(data, make([]byte, 12)...)
		data = append(data, id.Origin.Bytes()...)
		data = append(data, make([]byte, 24)...)
		data = binary.BigEndian.AppendUint64(data, id.BlockNumber)
		data = append(data, make([]byte, 28)...)
		data = binary.BigEndian.AppendUint32(data, id.LogIndex)
		data = append(data, make([]byte, 24)...)
		data = binary.BigEndian.AppendUint64(data, id.Timestamp)
		chainID := id.ChainID.Bytes32()
		data = append(data, chainID[:]...)
		return &types.Log{
			Address: predeploys.CrossL2InboxAddr,
			Topics:  []common.Hash{messages.ExecutingMessageEventTopic, msgHash},
			Data:    data,
		}
	}
	id := func(origin byte, blockNum uint64, logIdx uint32, ts uint64, chain uint64) messages.Identifier {
		return messages.Identifier{
			Origin:      common.Address{origin},
			BlockNumber: blockNum,
			LogIndex:    logIdx,
			Timestamp:   ts,
			ChainID:     eth.ChainIDFromUInt64(chain),
		}
	}
	ordinary := &types.Log{
		Address: common.Address{0xab},
		Topics:  []common.Hash{repeatHash(0x01)},
		Data:    []byte("an initiating message, not an executing one"),
	}

	t.Run("filters, sorts and dedups in one pass", func(t *testing.T) {
		// Authored in APPEARANCE order, with an ordinary log between them and one exact duplicate.
		got, err := ExecMsgsFromLogs([]*types.Log{
			execLog(id(0x02, 5, 0, 900, 2), repeatHash(0xbb)),
			ordinary,
			execLog(id(0x01, 7, 3, 950, 1), repeatHash(0xaa)),
			execLog(id(0x02, 5, 0, 900, 2), repeatHash(0xbb)), // exact repeat: one edge, not two
		})
		require.NoError(t, err)
		require.Len(t, got, 2, "the ordinary log is not an import and the repeat is not a second one")
		// Origin leads, so 0x01 sorts before 0x02 even though it appeared second.
		require.Equal(t, common.Address{0x01}, got[0].Identifier.Origin)
		require.Equal(t, common.Address{0x02}, got[1].Identifier.Origin)
		require.NoError(t, (&ProofBatch{Blocks: []BlockExport{
			{Number: 1, Timestamp: 1 << 40, ExecMsgs: got},
		}}).CheckStructure(), "the extraction's output must be something the codec accepts")
	})

	t.Run("a block that imports nothing yields nil, not an empty slice", func(t *testing.T) {
		got, err := ExecMsgsFromLogs([]*types.Log{ordinary})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("no logs at all", func(t *testing.T) {
		got, err := ExecMsgsFromLogs(nil)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	// THE ABORT RULE. Right address, right topic count, right event topic — and a body that does not
	// decode. Skipping it would commit to an import list that omits a message the block really
	// consumed, which is precisely what the conditional-validity claim forbids.
	t.Run("a malformed executing message aborts", func(t *testing.T) {
		bad := execLog(id(0x01, 5, 0, 900, 1), repeatHash(0xaa))
		bad.Data = bad.Data[:len(bad.Data)-1] // one byte short of a decodable identifier
		_, err := ExecMsgsFromLogs([]*types.Log{bad})
		require.Error(t, err, "a malformed CrossL2Inbox event must abort, never be skipped")
		require.ErrorContains(t, err, "import list cannot be determined")
	})

	// The OTHER malformed shape, and the one the shared decoder reports as "not an executing message"
	// rather than as an error: right emitter, right event topic, wrong TOPIC COUNT. The decoder checks
	// the count before the identity, so this returns nil — and skipping it would commit to an import
	// list that omits a message the block may really have consumed.
	t.Run("right topic, wrong topic count, still aborts", func(t *testing.T) {
		for _, topics := range [][]common.Hash{
			{messages.ExecutingMessageEventTopic},
			{messages.ExecutingMessageEventTopic, repeatHash(0xaa), repeatHash(0xbb)},
		} {
			bad := execLog(id(0x01, 5, 0, 900, 1), repeatHash(0xaa))
			bad.Topics = topics
			_, err := ExecMsgsFromLogs([]*types.Log{bad})
			require.Errorf(t, err, "a CrossL2Inbox ExecutingMessage with %d topics must abort", len(topics))
			require.ErrorContains(t, err, "import list cannot be determined")
		}
	})

	// ...and the abort is not reachable by an ordinary log that merely resembles one, or every block
	// with an unrelated two-topic event would fail to extract.
	t.Run("a look-alike from another address is just skipped", func(t *testing.T) {
		lookalike := execLog(id(0x01, 5, 0, 900, 1), repeatHash(0xaa))
		lookalike.Address = common.Address{0xff}
		lookalike.Data = lookalike.Data[:4]
		got, err := ExecMsgsFromLogs([]*types.Log{lookalike})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	// A CrossL2Inbox log for some OTHER event is also just skipped — the predeploy is allowed to emit
	// things that are not executing messages, and treating those as broken would fail every block.
	t.Run("another CrossL2Inbox event is just skipped", func(t *testing.T) {
		other := execLog(id(0x01, 5, 0, 900, 1), repeatHash(0xaa))
		other.Topics[0] = repeatHash(0x77)
		got, err := ExecMsgsFromLogs([]*types.Log{other})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	// A log with no topics at all must not panic on the topic[0] probe.
	t.Run("no topics", func(t *testing.T) {
		got, err := ExecMsgsFromLogs([]*types.Log{{Address: predeploys.CrossL2InboxAddr}})
		require.NoError(t, err)
		require.Nil(t, got)
	})
}
