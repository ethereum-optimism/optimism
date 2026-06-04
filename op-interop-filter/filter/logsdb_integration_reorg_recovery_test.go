package filter

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// putIntoReorg forces the ingester into ErrorReorg by ingesting a block whose
// parent hash doesn't match the stored latest. Returns the wrong-parent block
// number that was rejected.
func putIntoReorg(t *testing.T, si *seededIngester, num, ts uint64) uint64 {
	t.Helper()
	si.addBlock(num, ts, common.Hash{0xde, 0xad}, []seedLog{{}})
	require.Error(t, si.ingestBlock(num))
	require.NotNil(t, si.Error())
	require.Equal(t, ErrorReorg, si.Error().Reason)
	return num
}

// reorgRecoveryBackend builds a one-chain backend with reorg recovery
// disabled (so tests can drive it deterministically). The returned ingester is
// already seeded with three blocks.
func reorgRecoveryBackend(t *testing.T) (*seededBackend, *seededIngester) {
	t.Helper()
	bk := newSeededBackend(t, backendOpts{
		Specs: []seedSpec{
			{ChainID: 901, AnchorNumber: 99, AnchorTime: 1198,
				Blocks: []seedBlock{
					{Num: 100, Ts: 1200, Logs: []seedLog{{}}},
					{Num: 101, Ts: 1202, Logs: []seedLog{{}}},
					{Num: 102, Ts: 1204, Logs: []seedLog{{}}},
				}},
		},
	})
	si := bk.ingesters[eth.ChainIDFromUInt64(901)]
	return bk, si
}
func TestIntegration_RecoverReorg_HappyPath(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[101])
	putIntoReorg(t, si, 103, 1206)
	blockID, timestamp, err := bk.recoverChainReorg(context.Background(), si.chainID, si.LogsDBChainIngester)
	require.NoError(t, err)
	require.Equal(t, uint64(101), blockID.Number)
	require.Equal(t, si.blockInfo[101].hash, blockID.Hash)
	require.Equal(t, uint64(1202), timestamp)
	require.Equal(t, uint64(102), si.applyPendingRewind(200),
		"ingestion loop must resume from finalized+1 after recovery")
}
func TestIntegration_RecoverReorg_BeforeInit_Uninitialized(t *testing.T) {
	t.Parallel()
	si := newSeededIngester(t, seedSpec{
		NoIngest: true,
	})
	require.NoError(t, si.logsDB.Close())
	si.logsDB = nil
	_, _, err := si.RewindToFinalized(context.Background())
	require.ErrorIs(t, err, interop.ErrUninitialized)
}
func TestIntegration_RecoverReorg_FinalizedBlockNotInDB_StaysInFailsafe(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	// finalized points at a block far past latest sealed — FindSealedBlock returns ErrFuture.
	future := makeBlockInfo(200, 1500, common.Hash{0x99})
	si.eth.SetLabelBlock(eth.Finalized, future)
	putIntoReorg(t, si, 103, 1206)
	_, _, err := bk.recoverChainReorg(context.Background(), si.chainID, si.LogsDBChainIngester)
	require.Error(t, err)
	require.True(t, errors.Is(err, interop.ErrFuture), "expected ErrFuture, got %v", err)
	require.NotNil(t, si.Error())
	require.Equal(t, ErrorReorg, si.Error().Reason)
}
func TestIntegration_RecoverReorg_FinalizedHashMismatch_StaysInFailsafe(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	// finalized block has the correct number but a different hash than stored.
	fake := makeBlockInfo(101, 1202, common.Hash{0x77})
	si.eth.SetLabelBlock(eth.Finalized, fake)
	putIntoReorg(t, si, 103, 1206)
	_, _, err := bk.recoverChainReorg(context.Background(), si.chainID, si.LogsDBChainIngester)
	require.Error(t, err)
	require.True(t, errors.Is(err, interop.ErrConflict), "expected ErrConflict, got %v", err)
	require.NotNil(t, si.Error())
	require.Equal(t, ErrorReorg, si.Error().Reason)
}
func TestIntegration_RecoverReorg_FinalizedBelowEarliest_StaysInFailsafe(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	stale := makeBlockInfo(10, 1020, common.Hash{0x55})
	si.eth.SetLabelBlock(eth.Finalized, stale)
	putIntoReorg(t, si, 103, 1206)
	_, _, err := bk.recoverChainReorg(context.Background(), si.chainID, si.LogsDBChainIngester)
	require.Error(t, err)
	require.NotNil(t, si.Error())
	require.Equal(t, ErrorReorg, si.Error().Reason)
}
func TestIntegration_RecoverReorg_NotInReorgState_SkippedNoOp(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[101])
	si.SetError(ErrorConflict, "forced")
	_, _, err := bk.recoverChainReorg(context.Background(), si.chainID, si.LogsDBChainIngester)
	require.Error(t, err)
	require.NotNil(t, si.Error())
	require.Equal(t, ErrorConflict, si.Error().Reason)
}
func TestIntegration_RecoverReorg_FullSequence_ResolvesAndResumes(t *testing.T) {
	t.Parallel()
	bk, si := reorgRecoveryBackend(t)
	si.eth.SetLabelBlock(eth.Finalized, si.blockInfo[101])
	putIntoReorg(t, si, 103, 1206)
	bk.tryResolveReorgs(context.Background())
	require.Nil(t, si.Error(), "ingester error should be cleared after resolution")
	require.False(t, bk.FailsafeEnabled(), "failsafe should clear once last ingester error clears")
}
