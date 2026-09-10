package silhouette

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestFactStoreRestartRestoresCompleteELState(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenFactStore(dir)
	require.NoError(t, err)

	fact := Fact{
		Number: 12, Timestamp: 1_234, ParentHash: common.HexToHash("0x11"),
		Hash: common.HexToHash("0x12"), StateRoot: common.HexToHash("0x13"),
		MessagePasserStorageRoot: common.HexToHash("0x14"), OutputRoot: common.HexToHash("0x15"),
		L1Origin: eth.BlockID{Number: 7, Hash: common.HexToHash("0x17")}, ExecMsgsKnown: true,
	}
	store.Record(fact)
	store.recordCarrier(carrier{
		L1: eth.BlockID{Number: 9, Hash: common.HexToHash("0x19")}, FirstBlock: 12,
		LastBlock: 12, LastHash: fact.Hash, NewOutputRoot: fact.OutputRoot,
	})
	require.NoError(t, store.MarkDenied(fact.Number, fact.Hash))
	header := &types.Header{
		Number: new(big.Int).SetUint64(fact.Number), Difficulty: new(big.Int), BaseFee: new(big.Int),
		ParentHash: fact.ParentHash, Root: fact.StateRoot, GasLimit: 30_000_000,
	}
	store.RecordRendering(Rendering{Header: header, Txs: [][]byte{{0x01, 0x02}}, Hash: fact.Hash})
	ref := eth.L2BlockRef{Number: fact.Number, Hash: fact.Hash, ParentHash: fact.ParentHash, Time: fact.Timestamp, L1Origin: fact.L1Origin}
	store.SetCursors(Cursors{Unsafe: ref, Safe: ref, Finalized: ref})
	replacement := fact
	replacement.Hash = common.HexToHash("0x22")
	replacement.Replacement = true
	store.RecordReplacement(replacement)
	rewind := fact
	rewind.Hash = common.HexToHash("0x32")
	store.RecordRewindFact(rewind)
	store.setTrackerState(trackerState{
		Initialized: true, Start: 3, Next: 10,
		Processed: map[uint64]common.Hash{9: common.HexToHash("0x99")},
	})
	require.NoError(t, store.Close())

	restored, err := OpenFactStore(dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, restored.Close()) })
	got, ok := restored.ByNumber(fact.Number)
	require.True(t, ok)
	require.Equal(t, fact, got)
	carrierID, ok := restored.CarrierOf(fact.Number)
	require.True(t, ok)
	require.Equal(t, uint64(9), carrierID.Number)
	require.True(t, restored.IsDenied(fact.Hash))
	rendering, ok := restored.Rendering(fact.Hash)
	require.True(t, ok)
	require.Equal(t, [][]byte{{0x01, 0x02}}, rendering.Txs)
	require.Equal(t, ref, restored.Cursors().Unsafe)
	gotReplacement, ok := restored.ReplacementByNumber(replacement.Number)
	require.True(t, ok)
	require.Equal(t, replacement, gotReplacement)
	gotRewind, ok := restored.RewindFact(rewind.Hash)
	require.True(t, ok)
	require.Equal(t, rewind, gotRewind)
	tracker := restored.trackerState(3)
	require.Equal(t, uint64(10), tracker.Next)
	require.Equal(t, common.HexToHash("0x99"), tracker.Processed[9])
}

func TestFactStoreRejectsConcurrentProcessOpen(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenFactStore(dir)
	require.NoError(t, err)
	_, err = OpenFactStore(dir)
	require.Error(t, err)
	require.NoError(t, store.Close())
}
