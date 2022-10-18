package actions

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func setupReorgTest(t Testing) (*e2eutils.SetupData, *L1Miner, *L2Sequencer, *L2Engine, *L2Verifier, *L2Engine, *L2Batcher) {
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))
	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient())
	return sd, miner, sequencer, seqEngine, verifier, verifEngine, batcher
}

func TestReorgOrphanBlock(gt *testing.T) {
	t := NewDefaultTesting(gt)
	sd, miner, sequencer, _, verifier, _, batcher := setupReorgTest(t)

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
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	batchTx := miner.l1Transactions[0]
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")

	// orphan the L1 block that included the batch tx, and build a new different L1 block
	miner.ActL1RewindToParent(t)
	miner.ActL1SetFeeRecipient(common.Address{'B'})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t) // needs to be a longer chain for reorg to be applied. TODO: maybe more aggressively react to reorgs to shorter chains?

	// sync verifier again. The L1 reorg excluded the batch, so now the previous L2 chain should be unsafe again.
	// However, the L2 chain can still be canonical later, since it did not reference the reorged L1 block
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Safe(), "verifier rewinds safe when L1 reorgs out batch")
	// TODO check that the same holds for verifier engine

	// Now replay the batch tx in a new L1 block
	miner.ActL1StartBlock(12)(t)
	miner.ActL1SetFeeRecipient(common.Address{'C'})
	// note: the geth tx pool reorgLoop is too slow (responds to chain head events, but async),
	// and there's no way to manually trigger runReorg, so we re-insert it ourselves.
	require.NoError(t, miner.eth.TxPool().AddLocal(batchTx))
	// need to re-insert previously included tx into the block
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	miner.ActL1EndBlock(t)

	// sync the verifier again: now it should be safe again
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via replayed batch on L1")
	// TODO check that the same holds for verifier engine

	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Safe(), "verifier and sequencer see same safe L2 block, while only verifier dealt with the orphan and replay")
}
