package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var _ interop.InteropBackend = (*testutils.MockInteropBackend)(nil)

func TestInteropVerifier(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, helpers.DefaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	// Temporary work-around: interop needs to be active, for cross-safety to not be instant.
	// The state genesis in this test is pre-interop however.
	sd.RollupCfg.InteropTime = new(uint64)
	logger := testlog.Logger(t, log.LevelDebug)
	seqMockBackend := &testutils.MockInteropBackend{}
	l1Miner, seqEng, seq := helpers.SetupSequencerTest(t, sd, logger,
		helpers.WithVerifierOpts(helpers.WithInteropBackend(seqMockBackend)))

	batcher := helpers.NewL2Batcher(logger, sd.RollupCfg, helpers.DefaultBatcherCfg(dp),
		seq.RollupClient(), l1Miner.EthClient(), seqEng.EthClient(), seqEng.EngineClient(t, sd.RollupCfg))

	verMockBackend := &testutils.MockInteropBackend{}
	_, ver := helpers.SetupVerifier(t, sd, logger,
		l1Miner.L1Client(t, sd.RollupCfg), l1Miner.BlobStore(), &sync.Config{},
		helpers.WithInteropBackend(verMockBackend))

	seq.ActL2PipelineFull(t)
	ver.ActL2PipelineFull(t)

	l2ChainID := types.ChainIDFromBig(sd.RollupCfg.L2ChainID)
	seqMockBackend.ExpectCheckBlock(l2ChainID, 1, types.Unsafe, nil)
	// create an unsafe L2 block
	seq.ActL2StartBlock(t)
	seq.ActL2EndBlock(t)
	seq.ActL2PipelineFull(t)
	seqMockBackend.AssertExpectations(t)
	status := seq.SyncStatus()
	require.Equal(t, uint64(1), status.UnsafeL2.Number)
	require.Equal(t, uint64(0), status.CrossUnsafeL2.Number)
	require.Equal(t, uint64(0), status.LocalSafeL2.Number)
	require.Equal(t, uint64(0), status.SafeL2.Number)

	// promote it to cross-unsafe in the backend
	// and see if the node picks up on it
	seqMockBackend.ExpectCheckBlock(l2ChainID, 1, types.CrossUnsafe, nil)
	seq.ActInteropBackendCheck(t)
	seq.ActL2PipelineFull(t)
	seqMockBackend.AssertExpectations(t)
	status = seq.SyncStatus()
	require.Equal(t, uint64(1), status.UnsafeL2.Number)
	require.Equal(t, uint64(1), status.CrossUnsafeL2.Number, "cross unsafe now")
	require.Equal(t, uint64(0), status.LocalSafeL2.Number)
	require.Equal(t, uint64(0), status.SafeL2.Number)

	// submit all new L2 blocks
	batcher.ActSubmitAll(t)
	// new L1 block with L2 batch
	l1Miner.ActL1StartBlock(12)(t)
	l1Miner.ActL1IncludeTx(sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(t)
	l1Miner.ActL1EndBlock(t)

	// Sync the L1 block, to verify the L2 block as local-safe.
	seqMockBackend.ExpectCheckBlock(l2ChainID, 1, types.CrossUnsafe, nil) // not cross-safe yet
	seq.ActL1HeadSignal(t)
	seq.ActL2PipelineFull(t)
	seqMockBackend.AssertExpectations(t)

	status = seq.SyncStatus()
	require.Equal(t, uint64(1), status.UnsafeL2.Number)
	require.Equal(t, uint64(1), status.CrossUnsafeL2.Number)
	require.Equal(t, uint64(1), status.LocalSafeL2.Number, "local safe changed")
	require.Equal(t, uint64(0), status.SafeL2.Number)

	// Now mark it as cross-safe
	seqMockBackend.ExpectCheckBlock(l2ChainID, 1, types.CrossSafe, nil)
	seq.ActInteropBackendCheck(t)
	seq.ActL2PipelineFull(t)
	seqMockBackend.AssertExpectations(t)

	status = seq.SyncStatus()
	require.Equal(t, uint64(1), status.UnsafeL2.Number)
	require.Equal(t, uint64(1), status.CrossUnsafeL2.Number)
	require.Equal(t, uint64(1), status.LocalSafeL2.Number)
	require.Equal(t, uint64(1), status.SafeL2.Number, "cross-safe reached")
	require.Equal(t, uint64(0), status.FinalizedL2.Number)

	// The verifier might not see the L2 block that was just derived from L1 as cross-verified yet.
	verMockBackend.ExpectCheckBlock(l2ChainID, 1, types.Unsafe, nil) // for the local unsafe check
	verMockBackend.ExpectCheckBlock(l2ChainID, 1, types.Unsafe, nil) // for the local safe check
	ver.ActL1HeadSignal(t)
	ver.ActL2PipelineFull(t)
	verMockBackend.AssertExpectations(t)
	status = ver.SyncStatus()
	require.Equal(t, uint64(1), status.UnsafeL2.Number, "synced the block")
	require.Equal(t, uint64(0), status.CrossUnsafeL2.Number, "not cross-verified yet")
	require.Equal(t, uint64(1), status.LocalSafeL2.Number, "derived from L1, thus local-safe")
	require.Equal(t, uint64(0), status.SafeL2.Number, "not yet cross-safe")
	require.Equal(t, uint64(0), status.FinalizedL2.Number)

	// signal that L1 finalized; the cross-safe block we have should get finalized too
	l1Miner.ActL1SafeNext(t)
	l1Miner.ActL1FinalizeNext(t)
	seq.ActL1SafeSignal(t)
	seq.ActL1FinalizedSignal(t)
	seq.ActL2PipelineFull(t)
	seqMockBackend.AssertExpectations(t)

	status = seq.SyncStatus()
	require.Equal(t, uint64(1), status.FinalizedL2.Number, "finalized the block")
}
