package render

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	msgr  = predeploys.L2toL2CrossDomainMessengerAddr
	inbox = predeploys.CrossL2InboxAddr
	// otherAddr stands for any ordinary private contract: an ERC-20, a DEX, the private chain's own
	// business. Its logs are exactly what the rendering must NOT reveal.
	otherAddr = common.HexToAddress("0x00000000000000000000000000000000000f00d1")
	extraAddr = common.HexToAddress("0x00000000000000000000000000000000000f00d2")
)

// exportLog builds a SentMessage log as the private messenger would emit it.
func exportLog(nonce uint64) *types.Log {
	return &types.Log{
		Address: msgr,
		Topics: []common.Hash{
			SentMessageEventTopic,
			common.BigToHash(big.NewInt(900)), // destination chain
			common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa"), // target
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

// importLog builds a CrossL2Inbox ExecutingMessage log as the private inbox would emit it. Its
// payload has to be a real ABI encoding, because the import replay decodes it back out.
func importLog(t *testing.T, id messages.Identifier, payloadHash common.Hash) *types.Log {
	t.Helper()
	data := make([]byte, 0, 32*5)
	data = append(data, common.LeftPadBytes(id.Origin.Bytes(), 32)...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.BlockNumber)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(uint64(id.LogIndex))).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.Timestamp)).Bytes()...)
	chainID := id.ChainID.Bytes32()
	data = append(data, chainID[:]...)
	return &types.Log{
		Address: inbox,
		Topics:  []common.Hash{messages.ExecutingMessageEventTopic, payloadHash},
		Data:    data,
	}
}

// relayedLog is the messenger's RelayedMessage, which the private chain emits on every import.
func relayedLog(nonce uint64) *types.Log {
	return &types.Log{
		Address: msgr,
		Topics: []common.Hash{
			RelayedMessageEventTopic,
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

func sampleIdentifier(logIdx uint32) messages.Identifier {
	return messages.Identifier{
		Origin:      msgr,
		BlockNumber: 77,
		LogIndex:    logIdx,
		Timestamp:   1700,
		ChainID:     eth.ChainIDFromUInt64(901),
	}
}

// block assembles a PrivateBlock out of per-transaction log groups, filling in the positional
// metadata a real EL would fill in, so the tests exercise the same shape production does.
func block(number, time uint64, txLogs ...[]*types.Log) PrivateBlock {
	hdr := &types.Header{Number: new(big.Int).SetUint64(number), Time: time}
	var txs types.Transactions
	var receipts types.Receipts
	logIndex := uint(0)
	for i, group := range txLogs {
		tx := types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 21000, Value: big.NewInt(0)})
		txs = append(txs, tx)
		r := &types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: tx.Hash(), TransactionIndex: uint(i)}
		for _, l := range group {
			l.BlockNumber = number
			l.TxIndex = uint(i)
			l.TxHash = tx.Hash()
			l.Index = logIndex
			logIndex++
			r.Logs = append(r.Logs, l)
		}
		receipts = append(receipts, r)
	}
	return PrivateBlock{Header: hdr, Txs: txs, Receipts: receipts}
}

func TestRenderedLogs(t *testing.T) {
	e0, e1 := exportLog(0), exportLog(1)
	i0 := importLog(t, sampleIdentifier(3), common.Hash{0x11})
	x0 := &types.Log{Address: extraAddr, Topics: []common.Hash{{0x77}}}
	// The messenger's RelayedMessage: right ADDRESS, wrong TOPIC0. Excluded by the pair rule.
	r0 := relayedLog(9)
	// A CrossL2Inbox log at some other topic, for symmetry.
	c0 := &types.Log{Address: inbox, Topics: []common.Hash{{0x42}}}

	tests := []struct {
		name string
		logs []*types.Log
		set  EmitterSet
		// want is, for each surviving log, its private index. The rendered index is always the
		// position in this slice, which is the whole content of the primitive.
		want []uint32
	}{
		{name: "empty block", logs: nil, want: nil},
		{name: "no emitter logs", logs: []*types.Log{otherLog(1), otherLog(2)}, want: nil},
		{name: "exports only", logs: []*types.Log{e0, e1}, want: []uint32{0, 1}},
		{
			// The case the ordering rule exists for: sorting exports before imports would move the
			// second export from rendered index 2 to rendered index 1.
			name: "export import export interleaved",
			logs: []*types.Log{e0, i0, e1},
			want: []uint32{0, 1, 2},
		},
		{
			name: "non-emitter logs interspersed",
			logs: []*types.Log{otherLog(1), e0, otherLog(2), otherLog(3), i0, e1, otherLog(4)},
			want: []uint32{1, 4, 5},
		},
		{
			name: "leading and trailing non-emitter logs",
			logs: []*types.Log{otherLog(1), otherLog(2), e0},
			want: []uint32{2},
		},
		{
			// The ruling's headline case: a private import block emits BOTH the inbox's
			// ExecutingMessage and the messenger's RelayedMessage, and only the first renders.
			name: "RelayedMessage is excluded, and shifts the logs after it",
			logs: []*types.Log{e0, i0, r0, e1},
			want: []uint32{0, 1, 3},
		},
		{
			name: "a CrossL2Inbox log at another topic is excluded",
			logs: []*types.Log{c0, e0},
			want: []uint32{1},
		},
		{
			name: "an import block renders exactly two logs",
			logs: []*types.Log{otherLog(1), i0, r0},
			want: []uint32{1},
		},
		{
			name: "extra emitter excluded by default",
			logs: []*types.Log{e0, x0, e1},
			want: []uint32{0, 2},
		},
		{
			name: "extra emitter included when configured, at any topic",
			logs: []*types.Log{e0, x0, e1},
			set:  NewEmitterSet(extraAddr),
			want: []uint32{0, 1, 2},
		},
		{
			name: "nil logs are skipped without consuming an index",
			logs: []*types.Log{nil, e0, nil, e1},
			want: []uint32{1, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderedLogs(tt.logs, tt.set)
			require.Len(t, got, len(tt.want))
			for i, rl := range got {
				require.Equal(t, tt.want[i], rl.PrivateLogIndex, "private index of rendered log %d", i)
				require.Equal(t, uint32(i), rl.RenderedLogIndex, "rendered indexes are dense from zero")
				require.Same(t, tt.logs[tt.want[i]], rl.Log, "the caller's log is carried, not a copy")
			}
		})
	}
}

func TestRenderedLogsDoesNotMutateInput(t *testing.T) {
	e0 := exportLog(0)
	e0.Index = 42
	got := RenderedLogs([]*types.Log{otherLog(1), e0}, EmitterSet{})
	require.Len(t, got, 1)
	require.Equal(t, uint(42), e0.Index, "the private log is untouched")
	require.Equal(t, uint32(0), got[0].RenderedLogIndex)
	require.Same(t, e0, got[0].Log, "the caller's log is carried, not a copy")
}

func TestRenderBlock(t *testing.T) {
	i0 := importLog(t, sampleIdentifier(3), common.Hash{0x11})

	t.Run("empty block renders to nothing", func(t *testing.T) {
		got, err := RenderBlock(block(10, 1000, []*types.Log{}), EmitterSet{})
		require.NoError(t, err)
		require.Equal(t, uint64(10), got.Number)
		require.Equal(t, uint64(1000), got.Timestamp)
		require.Empty(t, got.Logs)
		require.Empty(t, got.Actions)
	})

	t.Run("block with no transactions at all", func(t *testing.T) {
		got, err := RenderBlock(block(10, 1000), EmitterSet{})
		require.NoError(t, err)
		require.Empty(t, got.Actions)
	})

	t.Run("multiple logs per transaction", func(t *testing.T) {
		b := block(11, 1002,
			[]*types.Log{otherLog(1)},
			[]*types.Log{exportLog(0), otherLog(2), exportLog(1)},
			[]*types.Log{i0},
		)
		got, err := RenderBlock(b, EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, 3)
		require.Equal(t, []uint32{1, 3, 4}, []uint32{
			got.Actions[0].PrivateLogIndex, got.Actions[1].PrivateLogIndex, got.Actions[2].PrivateLogIndex,
		})
		require.Equal(t, []uint32{1, 1, 2}, []uint32{
			got.Actions[0].PrivateTxIndex, got.Actions[1].PrivateTxIndex, got.Actions[2].PrivateTxIndex,
		})
		require.Equal(t, []ReplayKind{ReplayExport, ReplayExport, ReplayImport}, []ReplayKind{
			got.Actions[0].Kind, got.Actions[1].Kind, got.Actions[2].Kind,
		})
	})

	t.Run("interleaved kinds keep their original order", func(t *testing.T) {
		b := block(12, 1004, []*types.Log{exportLog(0), i0, exportLog(1)})
		got, err := RenderBlock(b, EmitterSet{})
		require.NoError(t, err)
		require.Equal(t, []ReplayKind{ReplayExport, ReplayImport, ReplayExport}, []ReplayKind{
			got.Actions[0].Kind, got.Actions[1].Kind, got.Actions[2].Kind,
		})
		require.Equal(t, uint32(2), got.Actions[2].RenderedLogIndex,
			"the trailing export keeps rendered index 2; sorting by kind would give it 1")
	})

	t.Run("import carries its decoded message", func(t *testing.T) {
		id := sampleIdentifier(5)
		payloadHash := common.Hash{0xab}
		b := block(13, 1006, []*types.Log{importLog(t, id, payloadHash)})
		got, err := RenderBlock(b, EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, 1)
		require.NotNil(t, got.Actions[0].Import)
		require.Equal(t, id, got.Actions[0].Import.Identifier)
		require.Equal(t, payloadHash, got.Actions[0].Import.PayloadHash)
	})

	t.Run("an import block renders the inbox log and drops RelayedMessage", func(t *testing.T) {
		// The gas saving the ruling names: one replay transaction per import, not two.
		b := block(17, 1014, []*types.Log{importLog(t, sampleIdentifier(2), common.Hash{0x9}), relayedLog(2)})
		got, err := RenderBlock(b, EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, 1)
		require.Equal(t, ReplayImport, got.Actions[0].Kind)
	})

	t.Run("extra emitters render through the generic replayer", func(t *testing.T) {
		b := block(14, 1008, []*types.Log{{Address: extraAddr, Topics: []common.Hash{{0x5}}, Data: []byte{9}}})
		got, err := RenderBlock(b, NewEmitterSet(extraAddr))
		require.NoError(t, err)
		require.Len(t, got.Actions, 1)
		require.Equal(t, ReplayEvent, got.Actions[0].Kind)
		require.Nil(t, got.Actions[0].Import)
		require.Nil(t, got.Actions[0].Export)
	})

	t.Run("a messenger log that is not SentMessage is dropped", func(t *testing.T) {
		// A claim may only be rendered at its own emitter address, and the replay messenger can
		// emit nothing but SentMessage — so RelayedMessage is excluded rather than misplaced.
		got, err := RenderBlock(block(16, 1012, []*types.Log{exportLog(0), relayedLog(3), exportLog(1)}), EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, 2)
		require.Equal(t, []ReplayKind{ReplayExport, ReplayExport}, []ReplayKind{got.Actions[0].Kind, got.Actions[1].Kind})
		require.Equal(t, []uint32{0, 2}, []uint32{got.Actions[0].PrivateLogIndex, got.Actions[1].PrivateLogIndex})
		require.Equal(t, uint32(1), got.Actions[1].RenderedLogIndex,
			"the trailing export takes rendered index 1: excluding a log shifts what follows, by design")
	})

	t.Run("actions and logs stay one to one", func(t *testing.T) {
		b := block(15, 1010, []*types.Log{exportLog(0), otherLog(1), i0, exportLog(1)})
		got, err := RenderBlock(b, EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, len(got.Logs))
		for i := range got.Actions {
			require.Equal(t, got.Logs[i], got.Actions[i].RenderedLog)
		}
	})
}

// TestPrivateRef covers what the builder copies and what the claim commits to. A caller holding an
// execution payload rather than a full header supplies the ref explicitly and MUST get it back
// whole: hashing a partial header would produce no block's hash, and a header cannot say what the
// block's L1 origin was at all.
func TestPrivateRef(t *testing.T) {
	b := block(30, 3000, []*types.Log{exportLog(0)})

	got, err := RenderBlock(b, EmitterSet{})
	require.NoError(t, err)
	require.Equal(t, b.Header.Hash(), got.PrivateRef.Hash, "with no ref, as much as the header gives")
	require.Equal(t, b.Header.ParentHash, got.PrivateRef.ParentHash)
	require.Equal(t, uint64(30), got.PrivateRef.Number)
	require.Equal(t, uint64(3000), got.PrivateRef.Time)
	require.Zero(t, got.PrivateRef.L1Origin, "a header cannot say what its L1 origin was")
	require.Zero(t, got.PrivateRef.SequenceNumber)

	ref := eth.L2BlockRef{
		Hash:           common.Hash{0xab, 0xcd},
		ParentHash:     common.Hash{0xef},
		Number:         30,
		Time:           3000,
		L1Origin:       eth.BlockID{Hash: common.Hash{0x11}, Number: 7},
		SequenceNumber: 4,
	}
	b.Ref = ref
	got, err = RenderBlock(b, EmitterSet{})
	require.NoError(t, err)
	require.Equal(t, ref, got.PrivateRef, "the caller's ref is carried whole")
	require.NotEqual(t, b.Header.Hash(), got.PrivateRef.Hash)
}

func TestRenderBlockRefusesBadInput(t *testing.T) {
	t.Run("no header", func(t *testing.T) {
		_, err := RenderBlock(PrivateBlock{}, EmitterSet{})
		require.ErrorIs(t, err, ErrInconsistentBlock)
	})

	t.Run("receipt count mismatch", func(t *testing.T) {
		b := block(10, 1000, []*types.Log{})
		b.Receipts = append(b.Receipts, &types.Receipt{})
		_, err := RenderBlock(b, EmitterSet{})
		require.ErrorIs(t, err, ErrInconsistentBlock)
	})

	t.Run("log from another block", func(t *testing.T) {
		b := block(10, 1000, []*types.Log{exportLog(0)})
		b.Receipts[0].Logs[0].BlockNumber = 11
		_, err := RenderBlock(b, EmitterSet{})
		require.ErrorIs(t, err, ErrInconsistentBlock)
	})

	t.Run("inbox log at another topic is filtered, not fatal", func(t *testing.T) {
		// It is not in the emitter set at all, so there is nothing to render and nothing to fail.
		bad := &types.Log{Address: inbox, Topics: []common.Hash{{0x01}}, Data: nil}
		got, err := RenderBlock(block(10, 1000, []*types.Log{bad, exportLog(0)}), EmitterSet{})
		require.NoError(t, err)
		require.Len(t, got.Actions, 1)
		require.Equal(t, ReplayExport, got.Actions[0].Kind)
	})

	t.Run("ExecutingMessage topic with a wrong topic count", func(t *testing.T) {
		// Admitted by its pair, then fails to decode: fatal, because skipping it would renumber
		// every later message in the block.
		bad := &types.Log{Address: inbox, Topics: []common.Hash{messages.ExecutingMessageEventTopic, {0x1}, {0x2}}}
		_, err := RenderBlock(block(10, 1000, []*types.Log{bad}), EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
	})

	t.Run("SentMessage topic with a malformed payload", func(t *testing.T) {
		bad := &types.Log{Address: msgr, Topics: []common.Hash{SentMessageEventTopic, {0x1}, {0x2}, {0x3}}, Data: []byte{0x1}}
		_, err := RenderBlock(block(10, 1000, []*types.Log{bad}), EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
	})

	t.Run("SentMessage from SuperchainETHBridge", func(t *testing.T) {
		bad := exportLog(0)
		bad.Data = sentMessageData(predeploys.SuperchainETHBridgeAddr, []byte{0x01})
		_, err := RenderBlock(block(10, 1000, []*types.Log{bad}), EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
		require.ErrorContains(t, err, "sender is the SuperchainETHBridge")
	})

	t.Run("SentMessage to SuperchainETHBridge", func(t *testing.T) {
		bad := exportLog(0)
		bad.Topics[2] = common.BytesToHash(predeploys.SuperchainETHBridgeAddr.Bytes())
		_, err := RenderBlock(block(10, 1000, []*types.Log{bad}), EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
		require.ErrorContains(t, err, "target is the SuperchainETHBridge")
	})

	t.Run("SentMessage payload over rendering limit", func(t *testing.T) {
		bad := exportLog(0)
		bad.Data = sentMessageData(otherAddr, make([]byte, MaxRenderableMessageSize+1))
		_, err := RenderBlock(block(10, 1000, []*types.Log{bad}), EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
		require.ErrorContains(t, err, "exceeding the 1048576-byte rendering limit")
	})

	t.Run("inbox log with the right topic but a malformed payload", func(t *testing.T) {
		bad := &types.Log{Address: inbox, Topics: []common.Hash{messages.ExecutingMessageEventTopic, {0x2}}, Data: []byte{0x1}}
		b := block(10, 1000, []*types.Log{bad})
		_, err := RenderBlock(b, EmitterSet{})
		require.ErrorIs(t, err, ErrUnrenderableLog)
	})
}

// TestRenderBlockIsDeterministic is the small end of the project's consensus-critical invariant:
// the whole batch payload is a pure function of private-chain data, and this is the innermost
// function in that chain. The full byte-determinism gate lives in op-private-interop/builder.
func TestRenderBlockIsDeterministic(t *testing.T) {
	mk := func() PrivateBlock {
		return block(21, 2000,
			[]*types.Log{otherLog(1), exportLog(0)},
			[]*types.Log{importLog(t, sampleIdentifier(9), common.Hash{0x33}), exportLog(1), otherLog(2)},
			[]*types.Log{},
			[]*types.Log{exportLog(2)},
		)
	}
	first, err := RenderBlock(mk(), NewEmitterSet(extraAddr))
	require.NoError(t, err)
	second, err := RenderBlock(mk(), NewEmitterSet(extraAddr))
	require.NoError(t, err)

	// Deep equality over independently built inputs: no pointer identity can accidentally pass it.
	require.Equal(t, first, second)
	require.Len(t, first.Actions, 4)
}

// TestEmitterSetRenders pins the predicate itself: the emitter set is (address, topic0) PAIRS for
// the two protocol emitters, because a claim rendered at the wrong address is a broken claim, and
// under stock interop any log is a potential initiating message.
func TestEmitterSetRenders(t *testing.T) {
	var zero EmitterSet
	at := func(addr common.Address, topic common.Hash) *types.Log {
		return &types.Log{Address: addr, Topics: []common.Hash{topic}}
	}

	require.True(t, zero.Renders(exportLog(0)), "the zero value is the ratified default")
	require.True(t, zero.Renders(importLog(t, sampleIdentifier(0), common.Hash{})))
	require.False(t, zero.Renders(relayedLog(1)), "RelayedMessage is a messenger log at the wrong topic")
	require.False(t, zero.Renders(at(msgr, common.Hash{0x1})), "any other messenger topic")
	require.False(t, zero.Renders(at(inbox, common.Hash{0x1})), "any other inbox topic")
	require.False(t, zero.Renders(&types.Log{Address: msgr}), "a messenger log with no topics")
	require.False(t, zero.Renders(otherLog(1)))
	require.False(t, zero.Renders(nil))
	require.False(t, zero.Renders(at(extraAddr, common.Hash{0x77})))

	set := NewEmitterSet(extraAddr, extraAddr)
	require.True(t, set.Renders(at(extraAddr, common.Hash{0x77})), "an extra emitter renders at any topic")
	require.True(t, set.Renders(at(extraAddr, common.Hash{0x88})))
	require.True(t, set.Renders(exportLog(0)), "extras never remove the standard pairs")
	require.False(t, set.Renders(relayedLog(1)), "and never widen them either")

	// Re-listing a standard predeploy does NOT widen it to all topics: the pair rule wins, because
	// the whole point is that a messenger address may only carry a messenger claim.
	require.False(t, NewEmitterSet(msgr).Renders(relayedLog(1)))
}

func TestSentMessageEventTopic(t *testing.T) {
	// Pinned against the contract's own SENT_MESSAGE_EVENT_SELECTOR, so a signature change in
	// L2ToL2CrossDomainMessenger.sol shows up here rather than as a silently empty rendering.
	require.Equal(t,
		crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)")),
		SentMessageEventTopic)
	require.Equal(t,
		common.HexToHash("0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320"),
		SentMessageEventTopic)
}
