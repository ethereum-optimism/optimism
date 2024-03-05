package actions

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestRecordSafeHeadUpdates(gt *testing.T) {
	t := NewDefaultTesting(gt)
	sd, miner, sequencer, verifier, verifierEng, batcher := setupSafeDBTest(t, defaultRollupTestParams)
	verifEngClient := verifierEng.EngineClient(t, sd.RollupCfg)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 block
	miner.ActEmptyBlock(t)

	// Create L2 blocks, and reference the L1 head as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks
	batcher.ActSubmitAll(t)

	// new L1 block with L2 batch
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(t)
	batchTx := miner.l1Transactions[0]
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")

	// Verify the safe head is recorded
	l1Head := miner.l1Chain.CurrentBlock()
	firstSafeHeadUpdateL1Block := l1Head.Number.Uint64()
	response, err := verifier.RollupClient().SafeHeadAtL1Block(context.Background(), firstSafeHeadUpdateL1Block)
	require.NoError(t, err)
	require.Equal(t, eth.HeaderBlockID(l1Head), response.L1Block)
	require.Equal(t, verifier.L2Unsafe().ID(), response.SafeHead)

	// Should get the same result for anything after that L1 block too
	response, err = verifier.RollupClient().SafeHeadAtL1Block(context.Background(), firstSafeHeadUpdateL1Block+1)
	require.NoError(t, err)
	require.Equal(t, eth.HeaderBlockID(l1Head), response.L1Block)
	require.Equal(t, verifier.L2Unsafe().ID(), response.SafeHead)

	// Only genesis is safe at this point
	response, err = verifier.RollupClient().SafeHeadAtL1Block(context.Background(), firstSafeHeadUpdateL1Block-1)
	require.NoError(t, err)
	require.Equal(t, eth.HeaderBlockID(miner.l1Chain.Genesis().Header()), response.L1Block)
	require.Equal(t, sd.RollupCfg.Genesis.L2, response.SafeHead)

	// orphan the L1 block that included the batch tx, and build a new different L1 block
	miner.ActL1RewindToParent(t)
	miner.ActL1SetFeeRecipient(common.Address{'B'})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t) // needs to be a longer chain for reorg to be applied.

	// sync verifier again. The L1 reorg excluded the batch, so now the previous L2 chain should be unsafe again.
	// However, the L2 chain can still be canonical later, since it did not reference the reorged L1 block
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Safe(), "verifier rewinds safe when L1 reorgs out batch")
	ref, err := verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, verifier.L2Safe(), ref, "verifier engine matches rollup client")

	// The safe head has been reorged so the record should have been deleted, leaving us back with just genesis safe
	response, err = verifier.RollupClient().SafeHeadAtL1Block(context.Background(), firstSafeHeadUpdateL1Block)
	require.NoError(t, err)
	require.Equal(t, eth.HeaderBlockID(miner.l1Chain.Genesis().Header()), response.L1Block)
	require.Equal(t, sd.RollupCfg.Genesis.L2, response.SafeHead)

	// Now replay the batch tx in a new L1 block
	miner.ActL1StartBlock(12)(t)
	miner.ActL1SetFeeRecipient(common.Address{'C'})
	// note: the geth tx pool reorgLoop is too slow (responds to chain head events, but async),
	// and there's no way to manually trigger runReorg, so we re-insert it ourselves.
	require.NoError(t, miner.eth.TxPool().Add([]*types.Transaction{batchTx}, true, true)[0])
	// need to re-insert previously included tx into the block
	miner.ActL1IncludeTx(sd.RollupCfg.Genesis.SystemConfig.BatcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync the verifier again: now it should be safe again
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via replayed batch on L1")
	ref, err = verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, verifier.L2Safe(), ref, "verifier engine matches rollup client")

	// Verify the safe head is recorded again
	l1Head = miner.l1Chain.CurrentBlock()
	firstSafeHeadUpdateL1Block = l1Head.Number.Uint64()
	response, err = verifier.RollupClient().SafeHeadAtL1Block(context.Background(), firstSafeHeadUpdateL1Block)
	require.NoError(t, err)
	require.Equal(t, eth.HeaderBlockID(l1Head), response.L1Block)
	require.Equal(t, verifier.L2Unsafe().ID(), response.SafeHead)
}

func setupSafeDBTest(t Testing, config *e2eutils.TestParams) (*e2eutils.SetupData, *L1Miner, *L2Sequencer, *L2Verifier, *L2Engine, *L2Batcher) {
	dp := e2eutils.MakeDeployParams(t, config)

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	logger := testlog.Logger(t, log.LevelDebug)

	return setupSafeDBTestActors(t, dp, sd, logger)
}

func setupSafeDBTestActors(t Testing, dp *e2eutils.DeployParams, sd *e2eutils.SetupData, log log.Logger) (*e2eutils.SetupData, *L1Miner, *L2Sequencer, *L2Verifier, *L2Engine, *L2Batcher) {
	dir := t.TempDir()
	db, err := safedb.NewSafeDB(log, dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{}, WithSafeHeadListener(db))
	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))
	return sd, miner, sequencer, verifier, verifEngine, batcher
}
