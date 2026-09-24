package render

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The log-side leaves equal the leaves admission builds from the replay calldata the builder emits
// for the same block, including the rendered-index gap left by an extra emitter's event.
func TestMessageLeavesMatchReplayCalldata(t *testing.T) {
	extra := &types.Log{Address: extraAddr, Topics: []common.Hash{{0xe1}}, Data: []byte{1}}
	b := block(12, 1024,
		[]*types.Log{exportLog(0), otherLog(1)},
		[]*types.Log{importLog(t, sampleIdentifier(3), common.Hash{0x77}), relayedLog(5)},
		[]*types.Log{extra, exportLog(1)},
	)
	rb, err := RenderBlock(b, NewEmitterSet(extraAddr))
	require.NoError(t, err)
	require.Len(t, rb.Logs, 4)
	got, err := MessageLeaves(rb.Number, rb.Logs)
	require.NoError(t, err)

	txs := testBuilder(t)
	txs.SetEventReplayer(predeploys.EventReplayerAddr)
	var want []common.Hash
	for i, act := range rb.Actions {
		tx, err := txs.ReplayTx(act)
		require.NoError(t, err)
		data := tx.Data()
		switch *tx.To() {
		case predeploys.L2toL2CrossDomainMessengerAddr:
			m, err := wire.DecodeReplaySentMessage(data)
			require.NoError(t, err)
			want = append(want, projection.MessageLeaf(12, uint32(i), projection.MessageKindInit, projection.ExportMessageHash(m)))
		case predeploys.CrossL2InboxAddr:
			want = append(want, projection.MessageLeaf(12, uint32(i), projection.MessageKindExec, projection.ImportMessageHash([192]byte(data[4:196]))))
		default:
			require.Equal(t, predeploys.EventReplayerAddr, *tx.To())
		}
	}
	require.Len(t, want, 3)
	require.Equal(t, want, got)
	// Rendered indices 0, 1 and 3: the event at index 2 has no leaf.
	require.Equal(t, projection.MessageLeaf(12, 3, projection.MessageKindInit, crypto.Keccak256Hash(messages.LogToMessagePayload(rb.Logs[3].Log))), got[2])

	none, err := MessageLeaves(12, nil)
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestMessageLeavesRefuseUnrenderable(t *testing.T) {
	bridgeSender := exportLog(0)
	bridgeSender.Data = sentMessageData(predeploys.SuperchainETHBridgeAddr, []byte{1})
	bridgeTarget := exportLog(0)
	bridgeTarget.Topics[2] = common.BytesToHash(predeploys.SuperchainETHBridgeAddr[:])
	trailing := exportLog(0)
	trailing.Data = append(append([]byte(nil), trailing.Data...), make([]byte, 32)...)
	shortImport := importLog(t, sampleIdentifier(0), common.Hash{1})
	shortImport.Topics = append(shortImport.Topics, common.Hash{2})
	for name, l := range map[string]*types.Log{
		"bridge_sender": bridgeSender, "bridge_target": bridgeTarget, "noncanonical_data": trailing, "malformed_import": shortImport,
	} {
		_, err := MessageLeaves(1, []RenderedLog{{Log: l}})
		require.ErrorIs(t, err, ErrUnrenderableLog, name)
	}
}
