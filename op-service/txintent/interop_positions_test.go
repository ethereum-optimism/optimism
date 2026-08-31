package txintent

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type fakeResolver struct {
	owns      common.Hash
	positions []PublicPosition
	calls     int
}

func (f *fakeResolver) Owns(_ context.Context, block eth.BlockRef) bool {
	return block.Hash == f.owns
}

func (f *fakeResolver) ResolvePositions(_ context.Context, _ *types.Receipt, _ eth.BlockRef) ([]PublicPosition, error) {
	f.calls++
	return f.positions, nil
}

func receiptWithLogs(n int) *types.Receipt {
	rec := &types.Receipt{}
	for i := range n {
		rec.Logs = append(rec.Logs, &types.Log{
			Address:     common.Address{byte(i + 1)},
			BlockNumber: 7,
			Index:       uint(i),
		})
	}
	return rec
}

// A chain with no resolver registered gets exactly the identifiers it always got: the receipt's own
// positions. This is the property that lets the seam exist in a shared helper at all.
func TestFromReceiptUnchangedWithoutResolver(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(901)
	rec := receiptWithLogs(3)
	block := eth.BlockRef{Number: 7, Hash: common.Hash{0xaa}, Time: 1234}

	out := (&InteropOutput{}).Init().(*InteropOutput)
	require.NoError(t, out.FromReceipt(context.Background(), rec, block, chainID))

	require.Len(t, out.Entries, 3)
	for i, entry := range out.Entries {
		require.Equal(t, common.Address{byte(i + 1)}, entry.Identifier.Origin)
		require.Equal(t, uint32(i), entry.Identifier.LogIndex)
		require.Equal(t, uint64(7), entry.Identifier.BlockNumber)
		require.Equal(t, uint64(1234), entry.Identifier.Timestamp)
	}
}

// A registered resolver moves the two coordinates that move on a rendering, and nothing else. A log
// with no public position keeps the receipt position it was built with, so the index a caller uses
// to reach an entry never shifts.
func TestFromReceiptAppliesPublicPositions(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(902)
	replayer := common.Address{0xee}
	resolver := &fakeResolver{
		owns: common.Hash{0xbb},
		positions: []PublicPosition{
			{Origin: replayer, LogIndex: 0, Public: true},
			{}, // not public: the export policy leaves this one on the private chain
			{Origin: common.Address{0x3}, LogIndex: 1, Public: true},
		},
	}
	defer RegisterPositionResolver(chainID, resolver)()

	rec := receiptWithLogs(3)
	block := eth.BlockRef{Number: 7, Hash: common.Hash{0xbb}, Time: 1234}
	out := (&InteropOutput{}).Init().(*InteropOutput)
	require.NoError(t, out.FromReceipt(context.Background(), rec, block, chainID))

	require.Equal(t, 1, resolver.calls)
	require.Len(t, out.Entries, 3)

	require.Equal(t, replayer, out.Entries[0].Identifier.Origin)
	require.Equal(t, uint32(0), out.Entries[0].Identifier.LogIndex)

	require.Equal(t, common.Address{0x2}, out.Entries[1].Identifier.Origin, "an unpublished log keeps its private position")
	require.Equal(t, uint32(1), out.Entries[1].Identifier.LogIndex)

	require.Equal(t, common.Address{0x3}, out.Entries[2].Identifier.Origin)
	require.Equal(t, uint32(1), out.Entries[2].Identifier.LogIndex, "the third log is the rendering's second")

	// Block number and timestamp are block-for-block and never move.
	for _, entry := range out.Entries {
		require.Equal(t, uint64(7), entry.Identifier.BlockNumber)
		require.Equal(t, uint64(1234), entry.Identifier.Timestamp)
	}
}

// A chain ID does not name a chain in a test process: a private chain and its rendering share one,
// and parallel worlds hold several. The block decides which resolver answers.
func TestPositionResolversDisambiguateByBlock(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(903)
	first := &fakeResolver{owns: common.Hash{0x11}, positions: []PublicPosition{{Origin: common.Address{0xf1}, Public: true}}}
	second := &fakeResolver{owns: common.Hash{0x22}, positions: []PublicPosition{{Origin: common.Address{0xf2}, Public: true}}}
	defer RegisterPositionResolver(chainID, first)()
	defer RegisterPositionResolver(chainID, second)()

	out := (&InteropOutput{}).Init().(*InteropOutput)
	block := eth.BlockRef{Number: 7, Hash: common.Hash{0x22}}
	require.NoError(t, out.FromReceipt(context.Background(), receiptWithLogs(1), block, chainID))
	require.Equal(t, common.Address{0xf2}, out.Entries[0].Identifier.Origin)
	require.Zero(t, first.calls)
	require.Equal(t, 1, second.calls)

	// A block neither of them produced is an error, not a guess.
	out = (&InteropOutput{}).Init().(*InteropOutput)
	err := out.FromReceipt(context.Background(), receiptWithLogs(1), eth.BlockRef{Number: 7, Hash: common.Hash{0x33}}, chainID)
	require.ErrorContains(t, err, "none of them owns block")
}

// Unregistering restores the stock path, which is what a test's cleanup has to guarantee for the
// next test in the process.
func TestUnregisterPositionResolver(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(904)
	resolver := &fakeResolver{owns: common.Hash{0xcc}, positions: []PublicPosition{{Origin: common.Address{0xf1}, Public: true}}}
	unregister := RegisterPositionResolver(chainID, resolver)
	unregister()
	unregister() // idempotent: a cleanup that ran twice must not corrupt the count

	out := (&InteropOutput{}).Init().(*InteropOutput)
	require.NoError(t, out.FromReceipt(context.Background(), receiptWithLogs(1), eth.BlockRef{Number: 7, Hash: common.Hash{0xcc}}, chainID))
	require.Equal(t, common.Address{0x1}, out.Entries[0].Identifier.Origin)
	require.Zero(t, resolver.calls)
}
