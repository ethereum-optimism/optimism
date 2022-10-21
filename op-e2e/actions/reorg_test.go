package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
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
	sd, miner, sequencer, _, verifier, verifierEng, batcher := setupReorgTest(t)
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
	ref, err := verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, verifier.L2Safe(), ref, "verifier engine matches rollup client")

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
	ref, err = verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Safe)
	require.NoError(t, err)
	require.Equal(t, verifier.L2Safe(), ref, "verifier engine matches rollup client")

	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Safe(), "verifier and sequencer see same safe L2 block, while only verifier dealt with the orphan and replay")
}

func TestReorgFlipFlop(gt *testing.T) {
	t := NewDefaultTesting(gt)
	sd, miner, sequencer, _, verifier, verifierEng, batcher := setupReorgTest(t)
	minerCl := miner.L1Client(t, sd.RollupCfg)
	verifEngClient := verifierEng.EngineClient(t, sd.RollupCfg)
	checkVerifEngine := func() {
		// TODO: geth preserves L2 chain with origin A1 after flip-flopping to B?
		//ref, err := verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Unsafe)
		//require.NoError(t, err)
		//t.Logf("l2 unsafe head %s with origin %s", ref, ref.L1Origin)
		//require.NotEqual(t, verifier.L2Unsafe().Hash, ref.ParentHash, "TODO off by one, engine syncs A0 after reorging back from B, while rollup node only inserts up to A0 (excl.)")
		//require.Equal(t, verifier.L2Unsafe(), ref, "verifier safe head of engine matches rollup client")

		ref, err := verifEngClient.L2BlockRefByLabel(t.Ctx(), eth.Safe)
		require.NoError(t, err)
		require.Equal(t, verifier.L2Safe(), ref, "verifier safe head of engine matches rollup client")
	}

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Start building chain A
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	blockA0, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)

	// Create L2 blocks, and reference the L1 head A0 as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks
	batcher.ActSubmitAll(t)

	// new L1 block A1 with L2 batch
	miner.ActL1SetFeeRecipient(common.Address{'A', 1})
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	batchTxA := miner.l1Transactions[0]
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe().L1Origin, blockA0.ID(), "verifier syncs L2 chain with L1 A0 origin")
	checkVerifEngine()

	// Flip to chain B!
	miner.ActL1RewindToParent(t) // undo A1
	miner.ActL1RewindToParent(t) // undo A0
	// build B0
	miner.ActL1SetFeeRecipient(common.Address{'B', 0})
	miner.ActEmptyBlock(t)
	blockB0, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(t, blockA0.Number, blockB0.Number, "same height")
	require.NotEqual(t, blockA0.Hash, blockB0.Hash, "different content")

	// re-include the batch tx that submitted L2 chain data that pointed to A0, in the new block B1
	miner.ActL1SetFeeRecipient(common.Address{'B', 1})
	miner.ActL1StartBlock(12)(t)
	require.NoError(t, miner.eth.TxPool().AddLocal(batchTxA))
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	miner.ActL1EndBlock(t)

	// make B2, the reorg is picked up when we have a new longer chain
	miner.ActL1SetFeeRecipient(common.Address{'B', 2})
	miner.ActEmptyBlock(t)
	blockB2, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)

	// now sync the verifier: some of the batches should be ignored:
	//	The safe head should have a genesis L1 origin, but past genesis, as some L2 blocks were built to get to A0 time
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sd.RollupCfg.Genesis.L1, verifier.L2Safe().L1Origin, "expected to be back at genesis origin after losing A0 and A1")

	require.NotZero(t, verifier.L2Safe().Number, "still preserving old L2 blocks that did not reference reorged L1 chain (assuming more than one L2 block per L1 block)")
	require.Equal(t, verifier.L2Safe(), verifier.L2Unsafe(), "head is at safe block after L1 reorg")
	checkVerifEngine()

	// and sync the sequencer, then build some new L2 blocks, up to and including with L1 origin B2
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	sequencer.ActBuildToL1Head(t)
	require.Equal(t, sequencer.L2Unsafe().L1Origin, blockB2.ID(), "B2 is the unsafe L1 origin of sequencer now")

	// submit all new L2 blocks for chain B, and include in new block B3
	batcher.ActSubmitAll(t)
	miner.ActL1SetFeeRecipient(common.Address{'B', 3})
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	miner.ActL1EndBlock(t)

	// sync the verifier to the L2 chain with origin B2
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe().L1Origin, blockB2.ID(), "B2 is the L1 origin of verifier now")
	checkVerifEngine()

	// Flop back to chain A!
	miner.ActL1RewindDepth(4)(t) // B3, B2, B1, B0
	pivotBlock, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(t, sd.RollupCfg.Genesis.L1, pivotBlock.ID(), "back at L1 genesis")
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	blockA0Again, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	require.Equal(t, blockA0, blockA0Again, "block A0 is back again after flip-flop")

	// Continue to build on A, and replay the old A-chain batch transaction (we can't replay B, since the nonce values would conflict)
	miner.ActL1SetFeeRecipient(common.Address{'A', 1})
	miner.ActEmptyBlock(t)

	miner.ActL1SetFeeRecipient(common.Address{'A', 2})
	miner.ActL1StartBlock(12)(t)
	require.NoError(t, miner.eth.TxPool().AddLocal(batchTxA)) // replay chain A batches, but now in A2 instead of A1
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	miner.ActL1EndBlock(t)

	// build more L1 blocks, so the chain is long enough for reorg to be picked up
	miner.ActL1SetFeeRecipient(common.Address{'A', 3})
	miner.ActEmptyBlock(t)
	miner.ActL1SetFeeRecipient(common.Address{'A', 4})
	miner.ActEmptyBlock(t)
	blockA4, err := minerCl.L1BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)

	// sync verifier, and see if A0 is safe, using the old replayed batch for chain A
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe().L1Origin, blockA0.ID(), "B2 is the L1 origin of verifier now")
	checkVerifEngine()

	// sync sequencer to the replayed L1 chain A
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Safe(), "sequencer reorgs to match verifier again")

	// and adopt the rest of L1 chain A into L2
	sequencer.ActBuildToL1Head(t)

	// submit the new unsafe A blocks
	batcher.ActSubmitAll(t)
	miner.ActL1SetFeeRecipient(common.Address{'A', 5})
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(sd.RollupCfg.BatchSenderAddress)(t)
	miner.ActL1EndBlock(t)

	// sync verifier to what ths sequencer submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer")
	require.Equal(t, verifier.L2Safe().L1Origin, blockA4.ID(), "L2 chain origin is A4")
	checkVerifEngine()
}
