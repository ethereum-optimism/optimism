package silhouette

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestClampCursorsToSurvivingHead(t *testing.T) {
	store := &FactStore{}
	first := Fact{Number: 1, Hash: common.HexToHash("0x01"), ParentHash: common.HexToHash("0xaa"), Timestamp: 2}
	second := Fact{Number: 2, Hash: common.HexToHash("0x02"), ParentHash: first.Hash, Timestamp: 4}
	store.Record(first)
	store.Record(second)
	firstRef := eth.L2BlockRef{Hash: first.Hash, Number: first.Number, ParentHash: first.ParentHash, Time: first.Timestamp}
	secondRef := eth.L2BlockRef{Hash: second.Hash, Number: second.Number, ParentHash: second.ParentHash, Time: second.Timestamp}
	store.SetCursors(Cursors{Unsafe: secondRef, Safe: secondRef, Finalized: firstRef})

	store.truncateBlocksLockedForTest(t, first.Number)
	store.ClampCursorsTo(first)

	got := store.Cursors()
	require.Equal(t, firstRef, got.Unsafe)
	require.Equal(t, firstRef, got.Safe)
	require.Equal(t, firstRef, got.Finalized, "a surviving label must remain untouched")
}

func TestClampCursorsToAnchor(t *testing.T) {
	store := &FactStore{}
	orphan := eth.L2BlockRef{Hash: common.HexToHash("0x02"), Number: 2}
	store.SetCursors(Cursors{Unsafe: orphan, Safe: orphan, Finalized: orphan})
	anchor := Fact{Hash: common.HexToHash("0x01"), L1Origin: eth.BlockID{Hash: common.HexToHash("0xa1")}}

	store.ClampCursorsTo(anchor)

	want := eth.L2BlockRef{Hash: anchor.Hash, L1Origin: anchor.L1Origin}
	require.Equal(t, Cursors{Unsafe: want, Safe: want, Finalized: want}, store.Cursors())
}

// truncateBlocksLockedForTest exercises the same atomic fact/rendering truncation as an L1 rewind
// without requiring a synthetic proof carrier merely to establish its cutoff.
func (f *FactStore) truncateBlocksLockedForTest(t *testing.T, through uint64) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.truncateBlocksLocked(through, true)
}
