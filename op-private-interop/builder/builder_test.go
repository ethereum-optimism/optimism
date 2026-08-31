package builder

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const (
	l1Genesis  = uint64(1_000_000)
	l1BlockGap = uint64(12)
	l2Genesis  = uint64(1_000_006)
	l2BlockGap = uint64(2)
	cadence    = 300
)

var (
	msgr       = predeploys.L2toL2CrossDomainMessengerAddr
	inbox      = predeploys.CrossL2InboxAddr
	otherAddr  = common.HexToAddress("0x00000000000000000000000000000000000f00d1")
	testKey, _ = crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	// testEventReplayer stands in for the genesis-assigned EventReplayer address, which the
	// genesis lane has not chosen yet.
	testEventReplayer = common.HexToAddress("0x00000000000000000000000000000000000e0e0e")
	// testRegistry stands in for the genesis-assigned ClaimRegistry address.
	testRegistry = common.HexToAddress("0x00000000000000000000000000000000000e9e9e")
	// testExtraEmitter is a genesis-configured extra emitter, which renders at any topic.
	testExtraEmitter = common.HexToAddress("0x00000000000000000000000000000000000eeeee")
	testEmitters     = render.NewEmitterSet(testExtraEmitter)
)

func testRollupCfg() *rollup.Config {
	cfg := &rollup.Config{
		Genesis:           rollup.Genesis{L2Time: l2Genesis},
		BlockTime:         l2BlockGap,
		MaxSequencerDrift: 1800,
		SeqWindowSize:     3600,
		L2ChainID:         big.NewInt(901),
	}
	cfg.ActivateAtGenesis(forks.Delta)
	return cfg
}

// l1Chain builds n contiguous L1 blocks at 12 s spacing, with hashes that are a function of the
// number so two constructions agree.
// It is FIXTURE data now, not builder input: the builder has no L1 view since origin-copy. It
// exists so sequencerOrigin below can play the part of the private sequencer, and so
// derive.CheckBatch has an L1 chain to validate the resulting span against.
func l1Chain(n int) []eth.L1BlockRef {
	out := make([]eth.L1BlockRef, 0, n)
	var parent common.Hash
	for i := range n {
		h := crypto.Keccak256Hash([]byte{byte(i), byte(i >> 8), 'l', '1'})
		out = append(out, eth.L1BlockRef{
			Hash: h, Number: uint64(i), ParentHash: parent, Time: l1Genesis + uint64(i)*l1BlockGap,
		})
		parent = h
	}
	return out
}

func exportLog(nonce uint64) *types.Log {
	return &types.Log{
		Address: msgr,
		Topics: []common.Hash{
			render.SentMessageEventTopic,
			common.BigToHash(big.NewInt(902)),
			common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa"),
			common.BigToHash(new(big.Int).SetUint64(nonce)),
		},
		Data: sentMessageData(otherAddr, []byte{0xde, 0xad, byte(nonce)}),
	}
}

// sentMessageData is abi.encode(address sender, bytes message), the SentMessage data section.
func sentMessageData(sender common.Address, message []byte) []byte {
	out := make([]byte, 0, 32*3+len(message))
	out = append(out, common.LeftPadBytes(sender.Bytes(), 32)...)
	out = append(out, common.BigToHash(big.NewInt(64)).Bytes()...)
	out = append(out, common.BigToHash(big.NewInt(int64(len(message)))).Bytes()...)
	out = append(out, message...)
	if pad := (32 - len(message)%32) % 32; pad > 0 {
		out = append(out, make([]byte, pad)...)
	}
	return out
}

func importLog(logIdx uint32) *types.Log {
	id := messages.Identifier{
		Origin: msgr, BlockNumber: 12, LogIndex: logIdx, Timestamp: l2Genesis, ChainID: eth.ChainIDFromUInt64(902),
	}
	data := make([]byte, 0, 32*5)
	data = append(data, common.LeftPadBytes(id.Origin.Bytes(), 32)...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.BlockNumber)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(uint64(id.LogIndex))).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.Timestamp)).Bytes()...)
	chainID := id.ChainID.Bytes32()
	data = append(data, chainID[:]...)
	return &types.Log{Address: inbox, Topics: []common.Hash{messages.ExecutingMessageEventTopic, {0x77}}, Data: data}
}

// extraLog is a configured extra emitter's log: no protocol claim, any topic, rendered through
// EventReplayer at the replayer's own address.
func extraLog(tag uint64) *types.Log {
	return &types.Log{
		Address: testExtraEmitter,
		Topics:  []common.Hash{{0x21}, common.BigToHash(new(big.Int).SetUint64(tag))},
		Data:    []byte{0x22},
	}
}

// relayedLog is the messenger's RelayedMessage, emitted by the private chain on every import.
func relayedLog(nonce uint64) *types.Log {
	return &types.Log{
		Address: msgr,
		Topics: []common.Hash{
			render.RelayedMessageEventTopic,
			common.BigToHash(big.NewInt(902)),
			common.BigToHash(new(big.Int).SetUint64(nonce)),
			{0x5a},
		},
		Data: common.Hash{0x5b}.Bytes(),
	}
}

func otherLog(tag byte) *types.Log {
	return &types.Log{Address: otherAddr, Topics: []common.Hash{{tag}}, Data: []byte{tag}}
}

// sequencerOrigin plays the PRIVATE SEQUENCER: it picks the origin a stock sequencer would have
// picked for a block at this timestamp — the newest L1 block at or before it, never advancing more
// than one epoch at a time — and the sequence number within that epoch.
//
// This rule used to live in the builder. It lives here now because it is the private chain's
// behaviour, and the whole point of origin-copy is that the builder does not repeat it.
func sequencerOrigin(l1 []eth.L1BlockRef, prev eth.BlockID, prevSeq uint64, blockTime uint64) (eth.BlockID, uint64) {
	newest := l1[0]
	for _, b := range l1 {
		if b.Time > blockTime {
			break
		}
		newest = b
	}
	chosen := newest.ID()
	if newest.Number > prev.Number+1 {
		chosen = l1[prev.Number+1].ID()
	}
	if chosen == prev {
		return chosen, prevSeq + 1
	}
	return chosen, 0
}

// renderedBlock runs the real transformation over a synthetic private block, so the builder's input
// is what the builder's input will be in production.
func renderedBlock(t *testing.T, number, timestamp uint64, ref eth.L2BlockRef, txLogs ...[]*types.Log) *render.RenderedBlock {
	t.Helper()
	hdr := &types.Header{Number: new(big.Int).SetUint64(number), Time: timestamp}
	var txs types.Transactions
	var receipts types.Receipts
	idx := uint(0)
	for i, group := range txLogs {
		tx := types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 21000, Value: big.NewInt(0)})
		txs = append(txs, tx)
		r := &types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: tx.Hash(), TransactionIndex: uint(i)}
		for _, l := range group {
			l.BlockNumber = number
			l.TxIndex = uint(i)
			l.Index = idx
			idx++
			r.Logs = append(r.Logs, l)
		}
		receipts = append(receipts, r)
	}
	out, err := render.RenderBlock(render.PrivateBlock{Header: hdr, Txs: txs, Receipts: receipts, Ref: ref}, testEmitters)
	require.NoError(t, err)
	return out
}

// renderRange builds `count` rendered blocks starting at firstNumber. Every tenth block carries an
// interleaved export/import/export, so a full cadence exercises messages, empty blocks and the
// interleaving rule at once.
func renderRange(t *testing.T, l1 []eth.L1BlockRef, head eth.L2BlockRef, count int) []*render.RenderedBlock {
	t.Helper()
	firstNumber, firstTimestamp := head.Number+1, head.Time+l2BlockGap
	origin, seq := head.L1Origin, head.SequenceNumber
	parent := head.Hash
	out := make([]*render.RenderedBlock, 0, count)
	for i := range count {
		num := firstNumber + uint64(i)
		ts := firstTimestamp + uint64(i)*l2BlockGap
		origin, seq = sequencerOrigin(l1, origin, seq, ts)
		ref := eth.L2BlockRef{
			Hash:           crypto.Keccak256Hash([]byte("private-block"), common.BigToHash(new(big.Int).SetUint64(num)).Bytes()),
			ParentHash:     parent,
			Number:         num,
			Time:           ts,
			L1Origin:       origin,
			SequenceNumber: seq,
		}
		parent = ref.Hash
		if i%10 == 0 {
			out = append(out, renderedBlock(t, num, ts, ref,
				[]*types.Log{otherLog(1), exportLog(num)},
				// A real import block: the inbox's ExecutingMessage plus the messenger's
				// RelayedMessage, which is a messenger log at a non-claim topic and is therefore
				// EXCLUDED by the (address, topic0) emitter set — one replay transaction per
				// import, not two. The extra-emitter log next to it is what does render through
				// EventReplayer.
				[]*types.Log{importLog(uint32(i)), relayedLog(num), extraLog(num), otherLog(2), exportLog(num + 1)},
			))
			continue
		}
		out = append(out, renderedBlock(t, num, ts, ref, []*types.Log{otherLog(3)}))
	}
	return out
}

func testBuilder(t *testing.T) *Builder {
	t.Helper()
	cfg := testRollupCfg()
	txs := render.NewOperatorTxBuilder(cfg.L2ChainID, render.DefaultGasPolicy(), render.PrivateKeySigner(testKey, cfg.L2ChainID))
	txs.SetEventReplayer(testEventReplayer)
	txs.SetRegistry(testRegistry)
	b, err := New(Config{Rollup: cfg, Emitters: testEmitters}, txs)
	require.NoError(t, err)
	return b
}

// safeHead is the rendering's last block of the previous range, as a verifier sees it. Its hash is
// what the follower reports and what the parent check must match.
func safeHead(l1 []eth.L1BlockRef, number, timestamp uint64) eth.L2BlockRef {
	origin, _ := sequencerOrigin(l1, l1[0].ID(), 0, timestamp)
	return eth.L2BlockRef{
		Hash:           crypto.Keccak256Hash([]byte("rendering-terminal")),
		Number:         number,
		Time:           timestamp,
		L1Origin:       origin,
		SequenceNumber: (timestamp - l1[origin.Number].Time) / l2BlockGap,
	}
}

func testRange(t *testing.T, l1 []eth.L1BlockRef, head eth.L2BlockRef, count int, claim *ClaimInput) *Range {
	t.Helper()
	return &Range{
		Blocks:                    renderRange(t, l1, head, count),
		PrevTerminalRenderingHash: head.Hash,
		Claim:                     claim,
		StartNonce:                42,
	}
}

// testClaimInput is the operator-supplied part of a claim. The builder derives the rest from the
// range itself, which is the property TestClaimDescribesItsOwnRange checks.
func testClaimInput() *ClaimInput {
	return &ClaimInput{
		RollupConfigHash: common.Hash{0x1b},
		DepSetHash:       common.Hash{0x1c}, PrivateDataHash: common.Hash{0x1d},
	}
}

// Since origin-copy this gate is a stronger statement than it was: the range is now a pure function
// of PRIVATE-CHAIN DATA ALONE plus the two operator inputs (the previous terminal rendering hash and
// the starting nonce). There is no live L1 input anywhere — the builder has no L1 view to be stale,
// and l1Head is read off the range's own terminal block — so two operators cannot diverge by
// observing L1 at different moments, which was the one remaining way honest builders could.
//
// TestRangeIsByteDeterministic is THE consensus-critical gate. Two builders with no shared state
// render the same private range into byte-identical span batch bytes, frames and blobs. If this
// ever fails, two honest operators (or one operator restarted) post different bytes for the same
// chain, and the rendering is not a function of the private chain any more.
func TestRangeIsByteDeterministic(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)

	build := func() *BuiltRange {
		out, err := testBuilder(t).Build(testRange(t, l1, head, cadence, testClaimInput()))
		require.NoError(t, err)
		return out
	}
	first, second := build(), build()

	require.Equal(t, first.ChannelID, second.ChannelID, "the channel ID is derived, never random")
	require.Equal(t, first.ChannelData, second.ChannelData, "the compressed channel payload is identical")
	require.Equal(t, first.Frames, second.Frames, "every frame is byte-identical")
	require.Len(t, first.Blobs, len(second.Blobs))
	for i := range first.Blobs {
		require.Equal(t, first.Blobs[i][:], second.Blobs[i][:], "blob %d is byte-identical", i)
	}
	require.Equal(t, first.Blocks, second.Blocks, "every block's transaction list is identical")
	require.Equal(t, first.NextNonce, second.NextNonce)

	// A single blob comfortably holds a 300-block cadence; the proposal measures three orders of
	// magnitude of headroom and this is the assertion that keeps that true.
	require.Len(t, first.Blobs, 1, "one cadence, one blob")
	require.NotEmpty(t, first.ChannelData)
}

func TestChannelIDIsNormative(t *testing.T) {
	prev := common.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	got := ChannelID(prev, 0x0102030405060708)
	seed := append(prev.Bytes(), []byte{1, 2, 3, 4, 5, 6, 7, 8}...)
	require.Equal(t, crypto.Keccak256(seed)[:16], got[:])

	// Distinct inputs give distinct channels, which is what stops two ranges colliding on L1.
	require.NotEqual(t, got, ChannelID(prev, 0x0102030405060709))
	require.NotEqual(t, got, ChannelID(common.Hash{0x99}, 0x0102030405060708))
}

// TestSpanBatchPassesStockValidation drives the synthesized span batch through the real
// derive.CheckBatch against a mocked L2 safe head and a real L1 chain. Nothing about the builder is
// trustworthy until stock derivation accepts its output.
//
// It is also the ORIGIN-COPY validity argument made concrete: the range's origins are the private
// sequencer's own, chosen under the identical stock rules at the identical timestamps, so a range
// that copies them satisfies span validation by construction. (TestOriginsAreCopied pins that they
// are in fact copied; this pins that copying them is valid.)
func TestSpanBatchPassesStockValidation(t *testing.T) {
	l1 := l1Chain(120)
	cfg := testRollupCfg()
	head := safeHead(l1, 900, l2Genesis+1800)

	for _, tc := range []struct {
		name   string
		blocks int
	}{
		{name: "full cadence with messages", blocks: cadence},
		{name: "single block", blocks: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			built, err := testBuilder(t).Build(testRange(t, l1, head, tc.blocks, testClaimInput()))
			require.NoError(t, err)
			require.Equal(t, derive.BatchValidity(derive.BatchAccept), validity(t, cfg, l1, head, built))
		})
	}
}

// TestWrongParentCheckIsDropped is the loud-stall gate: a parent check the rendering does not match
// must be a plain BatchDrop — attributable to the operator, and never a divergence or a reset.
func TestWrongParentCheckIsDropped(t *testing.T) {
	l1 := l1Chain(120)
	cfg := testRollupCfg()
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, 20, testClaimInput())
	// The parent check is only the FIRST 20 BYTES of the hash, so a wrong hash must differ there:
	// flipping a trailing byte would sail through, which is worth pinning.
	wrong := head.Hash
	wrong[0] ^= 0xff
	r.PrevTerminalRenderingHash = wrong

	built, err := testBuilder(t).Build(r)
	require.NoError(t, err, "the builder does not know the hash is wrong; only a verifier does")
	require.Equal(t, derive.BatchValidity(derive.BatchDrop), validity(t, cfg, l1, head, built))

	// The same range with the right hash is accepted, so the drop is attributable to the parent
	// check and to nothing else about the range.
	r.PrevTerminalRenderingHash = head.Hash
	good, err := testBuilder(t).Build(r)
	require.NoError(t, err)
	require.Equal(t, derive.BatchValidity(derive.BatchAccept), validity(t, cfg, l1, head, good))
}

// TestClaimPlacement is the placement gate. The claim is the range's permanent record and it LEADS
// the range it describes: the registry is log-less, so a leading transaction shifts nothing, and
// every message's rendering log index still equals its RenderedLogs rank.
func TestClaimPlacement(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, 12, testClaimInput())
	built, err := testBuilder(t).Build(r)
	require.NoError(t, err)

	// The range's opening block is the one that carries messages in this fixture (i%10 == 0).
	opening := built.Blocks[0]
	rendered := r.Blocks[0]
	require.Len(t, rendered.Actions, 4, "export, import, an extra emitter, export; RelayedMessage is excluded")
	require.Equal(t,
		[]render.ReplayKind{render.ReplayExport, render.ReplayImport, render.ReplayEvent, render.ReplayExport},
		[]render.ReplayKind{rendered.Actions[0].Kind, rendered.Actions[1].Kind, rendered.Actions[2].Kind, rendered.Actions[3].Kind},
		"the import contributes ONE replay transaction: its RelayedMessage is not a renderable claim")
	require.Len(t, opening.Txs, len(rendered.Actions)+1, "the claim, then the replay transactions")

	firstTx := decodeTx(t, opening.Txs[0])
	require.Equal(t, testRegistry, *firstTx.To(), "the FIRST transaction posts this range's claim")
	require.Equal(t, render.PostClaimSelector[:], firstTx.Data()[:4])
	body, err := codec.Encode(built.Claim)
	require.NoError(t, err)
	require.Equal(t, body, firstTx.Data()[4:], "the codec owns the claim's bytes on both sides")

	// THE POINT OF THE LOG-LESS REGISTRY. The claim transaction sits at index 0 and emits nothing,
	// so replay transaction k still emits rendering log k: message rendered index i is transaction
	// i+1, with no index anywhere shifted by the claim's presence. If the registry ever grew an
	// event, this offset would be the wrong one and this assertion would be the thing that caught it.
	// (That the call emits no log is the registry's own property, covered by the contracts suite;
	// here it is the transaction ORDERING that has to be right.)
	replayTxs := opening.Txs[1:]
	require.Len(t, replayTxs, len(rendered.Actions))
	for i, act := range rendered.Actions {
		require.Equal(t, uint32(i), act.RenderedLogIndex, "rendered index is the RenderedLogs rank")
		tx := decodeTx(t, replayTxs[i])
		switch act.Kind {
		case render.ReplayExport:
			require.Equal(t, msgr, *tx.To(), "replay transaction %d re-emits an export at the messenger", i)
			require.Equal(t, render.ReplaySentMessageSelector[:], tx.Data()[:4])
		case render.ReplayImport:
			require.Equal(t, inbox, *tx.To(), "replay transaction %d executes an import", i)
			require.NotEmpty(t, tx.AccessList(), "an import carries the checksum access list")
		case render.ReplayEvent:
			require.Equal(t, testEventReplayer, *tx.To(), "replay transaction %d re-emits through EventReplayer", i)
			require.Equal(t, render.ReplayEventSelector[:], tx.Data()[:4])
		}
		require.NotEqual(t, testRegistry, *tx.To(), "no replay transaction is a claim")
	}

	// One claim per range, in the opening block only.
	for i, blk := range built.Blocks[1:] {
		require.Len(t, blk.Txs, len(r.Blocks[i+1].Actions), "block %d has replay transactions only", i+1)
		for j, raw := range blk.Txs {
			require.NotEqual(t, testRegistry, *decodeTx(t, raw).To(),
				"block %d transaction %d posts a second claim", i+1, j)
		}
	}
}

// TestClaimDescribesItsOwnRange pins the shape that made leading placement possible: a claim names
// the range it opens, and commits to the PRIVATE chain's terminal hash — a fact that already
// existed when the claim was built, which is why there is no circularity and no one-range lag.
func TestClaimDescribesItsOwnRange(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, 12, testClaimInput())
	built, err := testBuilder(t).Build(r)
	require.NoError(t, err)

	require.Equal(t, built.FirstBlock, built.Claim.FirstBlock, "the claim opens the range it names")
	require.Equal(t, built.LastBlock, built.Claim.LastBlock)
	require.Equal(t, r.Blocks[len(r.Blocks)-1].PrivateRef.Hash, built.Claim.PrivateTerminalBlockHash,
		"the PRIVATE chain's hash at lastBlock, straight from the range's own blocks")
	require.NotEqual(t, head.Hash, built.Claim.PrivateTerminalBlockHash,
		"and NOT the rendering's terminal hash, which has left the claim entirely")
	require.Empty(t, built.Claim.Proof, "v1 posts an empty proof slot")

	// The chain's FIRST range is an ordinary range: it carries its own claim, so there is no
	// genesis edge to special-case anywhere.
	first := testRange(t, l1, head, 12, testClaimInput())
	first.Blocks = first.Blocks[:1]
	firstBuilt, err := testBuilder(t).Build(first)
	require.NoError(t, err)
	require.Equal(t, testRegistry, *decodeTx(t, firstBuilt.Blocks[0].Txs[0]).To())
	require.Equal(t, firstBuilt.FirstBlock, firstBuilt.Claim.FirstBlock)
	require.Equal(t, firstBuilt.LastBlock, firstBuilt.Claim.LastBlock)

	// A range with no claim is refused: the claim is the range's permanent record, not an option.
	noClaim := testRange(t, l1, head, 4, nil)
	_, err = testBuilder(t).Build(noClaim)
	require.ErrorIs(t, err, ErrRange)
}

// TestPrevTerminalHashSeedsParentCheckAndChannelID pins the one asynchronous input: the builder
// cannot start a range until somebody has DERIVED AND EXECUTED the previous one, and that single
// hash does two jobs — the span's 20-byte parent check, and the channel ID's seed.
func TestPrevTerminalHashSeedsParentCheckAndChannelID(t *testing.T) {
	l1 := l1Chain(120)
	cfg := testRollupCfg()
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, 20, testClaimInput())
	r.PrevTerminalRenderingHash = head.Hash

	built, err := testBuilder(t).Build(r)
	require.NoError(t, err)
	require.Equal(t, derive.BatchValidity(derive.BatchAccept), validity(t, cfg, l1, head, built))
	require.Equal(t, ChannelID(head.Hash, head.Number+1), built.ChannelID,
		"the same hash seeds the parent check and the channel ID")
}

// TestOriginsAreCopied is the origin-copy gate. The rendering block's epoch and sequence number are
// the private block's own, verbatim — not re-derived from a timestamp against a live L1 view.
func TestOriginsAreCopied(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, cadence, testClaimInput())
	built, err := testBuilder(t).Build(r)
	require.NoError(t, err)
	require.Len(t, built.Blocks, cadence)

	sawAdvance, sawRepeat := false, false
	for i, blk := range built.Blocks {
		priv := r.Blocks[i].PrivateRef
		require.Equal(t, priv.L1Origin, blk.Origin, "block %d copies its private origin", i)
		require.Equal(t, priv.SequenceNumber, blk.SeqNum, "block %d copies its private sequence number", i)
		if i > 0 {
			if blk.Origin != built.Blocks[i-1].Origin {
				sawAdvance = true
				require.Equal(t, built.Blocks[i-1].Origin.Number+1, blk.Origin.Number, "origins never skip")
				require.Zero(t, blk.SeqNum, "a new epoch restarts the sequence number")
			} else {
				sawRepeat = true
				require.Equal(t, built.Blocks[i-1].SeqNum+1, blk.SeqNum)
			}
		}
	}
	require.True(t, sawAdvance, "the fixture must cross epoch boundaries for this to mean anything")
	require.True(t, sawRepeat, "and must also hold an origin across blocks")
}

// TestBuildRefusesBlockWithoutOrigin: there is no fallback. A block that arrives without the origin
// to copy is refused, because the only alternative is inventing one — which is exactly the
// machinery origin-copy deleted, and which could only ever agree with the private chain or be wrong.
func TestBuildRefusesBlockWithoutOrigin(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)
	r := testRange(t, l1, head, 4, testClaimInput())
	r.Blocks[2].PrivateRef.L1Origin = eth.BlockID{}
	_, err := testBuilder(t).Build(r)
	require.ErrorIs(t, err, ErrNoOrigin)
}

// TestL1HeadIsTheTerminalOrigin pins the derived definition of the claim's l1Head: the newest L1
// block the range actually depends on, which since origin-copy is its terminal block's own origin.
// It is derived rather than supplied precisely so a verifier can check it against the rendering.
func TestL1HeadIsTheTerminalOrigin(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)

	r := testRange(t, l1, head, cadence, testClaimInput())
	built, err := testBuilder(t).Build(r)
	require.NoError(t, err)

	terminal := r.Blocks[len(r.Blocks)-1].PrivateRef
	require.Equal(t, terminal.L1Origin.Hash, built.Claim.L1Head)
	require.Equal(t, built.Blocks[len(built.Blocks)-1].Origin.Hash, built.Claim.L1Head,
		"and it is readable off the rendering's own terminal block, so it is checkable")
	require.NotZero(t, built.Claim.L1Head)
}

func TestBuildRefusesBadRanges(t *testing.T) {
	l1 := l1Chain(60)
	head := safeHead(l1, 900, l2Genesis+1800)
	b := testBuilder(t)

	t.Run("empty range", func(t *testing.T) {
		_, err := b.Build(&Range{})
		require.ErrorIs(t, err, ErrRange)
	})

	t.Run("gap in block numbers", func(t *testing.T) {
		r := testRange(t, l1, head, 3, testClaimInput())
		r.Blocks[2].Number++
		_, err := b.Build(r)
		require.ErrorIs(t, err, ErrRange)
	})

	t.Run("gap in timestamps", func(t *testing.T) {
		r := testRange(t, l1, head, 3, testClaimInput())
		r.Blocks[2].Timestamp += 2
		_, err := b.Build(r)
		require.ErrorIs(t, err, ErrRange)
	})
}

func TestNonceCarriesAcrossRanges(t *testing.T) {
	l1 := l1Chain(120)
	head := safeHead(l1, 900, l2Genesis+1800)
	b := testBuilder(t)
	r := testRange(t, l1, head, 12, testClaimInput())
	built, err := b.Build(r)
	require.NoError(t, err)

	var txCount uint64
	for _, blk := range built.Blocks {
		txCount += uint64(len(blk.Txs))
	}
	require.Equal(t, r.StartNonce+txCount, built.NextNonce,
		"NextNonce is the next range's StartNonce, which is what keeps ranges reproducible in sequence")
}

// --- helpers ---

func validity(t *testing.T, cfg *rollup.Config, l1 []eth.L1BlockRef, head eth.L2BlockRef, built *BuiltRange) derive.BatchValidity {
	t.Helper()
	// l1Blocks must start at the safe head's origin, as the batch queue supplies them.
	var window []eth.L1BlockRef
	for _, b := range l1 {
		if b.Number >= head.L1Origin.Number {
			window = append(window, b)
		}
	}
	inclusion := window[len(window)-1]
	return derive.CheckBatch(context.Background(), cfg, testlog.Logger(t, log.LevelError), window, head,
		&derive.BatchWithL1InclusionBlock{Batch: built.SpanBatch, L1InclusionBlock: inclusion},
		&noSafeBlocks{})
}

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

// noSafeBlocks is the fetcher for a NON-OVERLAPPING span: it is never consulted, and saying so by
// failing loudly is better than a mock that quietly returns zero values.
type noSafeBlocks struct{}

func (noSafeBlocks) L2BlockRefByNumber(context.Context, uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, errNotFound("the span does not overlap the safe chain; no fetch should happen")
}

func (noSafeBlocks) PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, errNotFound("the span does not overlap the safe chain; no fetch should happen")
}

func decodeTx(t *testing.T, raw []byte) *types.Transaction {
	t.Helper()
	var tx types.Transaction
	require.NoError(t, tx.UnmarshalBinary(raw))
	return &tx
}
