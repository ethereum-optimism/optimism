package actions

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestBatcher(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20, // larger than L1 block time we simulate in this test (12)
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))

	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient())

	// Alice makes a L2 tx
	cl := seqEngine.EthClient()
	n, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
	require.NoError(t, err)
	signer := types.LatestSigner(sd.L2Cfg.Config)
	tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   sd.L2Cfg.Config.ChainID,
		Nonce:     n,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: new(big.Int).Add(miner.l1Chain.CurrentBlock().BaseFee(), big.NewInt(2*params.GWei)),
		Gas:       params.TxGas,
		To:        &dp.Addresses.Bob,
		Value:     e2eutils.Ether(2),
	})
	require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Make L2 block
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	// batch submit to L1
	batcher.ActL2BatchBuffer(t)
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)

	// confirm batch on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)
	bl := miner.l1Chain.CurrentBlock()
	log.Info("bl", "txs", len(bl.Transactions()))

	// Now make enough L1 blocks that the verifier will have to derive a L2 block
	for i := uint64(1); i < sd.RollupCfg.SeqWindowSize; i++ {
		miner.ActL1StartBlock(12)(t)
		miner.ActL1EndBlock(t)
	}

	// sync verifier from L1 batch in otherwise empty sequence window
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, uint64(1), verifier.SyncStatus().SafeL2.L1Origin.Number)

	// check that the tx from alice made it into the L2 chain
	verifCl := verifEngine.EthClient()
	vTx, isPending, err := verifCl.TransactionByHash(t.Ctx(), tx.Hash())
	require.NoError(t, err)
	require.False(t, isPending)
	require.NotNil(t, vTx)
}

func TestL2Finalization(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, engine, sequencer := setupSequencerTest(t, sd, log)

	sequencer.ActL2PipelineFull(t)

	// build an empty L1 block (#1), mark it as justified
	miner.ActEmptyBlock(t)
	miner.ActL1SafeNext(t) // #0 -> #1

	// sequencer builds L2 chain, up to and including a block that has the new L1 block as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	sequencer.ActL2PipelineFull(t)
	sequencer.ActL1SafeSignal(t)
	require.Equal(t, uint64(1), sequencer.SyncStatus().SafeL1.Number)

	// build another L1 block (#2), mark it as justified. And mark previous justified as finalized.
	miner.ActEmptyBlock(t)
	miner.ActL1SafeNext(t)     // #1 -> #2
	miner.ActL1FinalizeNext(t) // #0 -> #1
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// continue to build L2 chain referencing the new L1 blocks
	sequencer.ActL2PipelineFull(t)
	sequencer.ActL1FinalizedSignal(t)
	sequencer.ActL1SafeSignal(t)
	require.Equal(t, uint64(2), sequencer.SyncStatus().SafeL1.Number)
	require.Equal(t, uint64(1), sequencer.SyncStatus().FinalizedL1.Number)
	require.Equal(t, uint64(0), sequencer.SyncStatus().FinalizedL2.Number, "L2 block has to be included on L1 before it can be finalized")

	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), engine.EthClient())

	heightToSubmit := sequencer.SyncStatus().UnsafeL2.Number

	batcher.ActSubmitAll(t)
	// confirm batch on L1, block #3
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// read the batch
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, uint64(0), sequencer.SyncStatus().FinalizedL2.Number, "Batch must be included in finalized part of L1 chain for L2 block to finalize")

	// build some more L2 blocks, so there is an unsafe part again that hasn't been submitted yet
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit those blocks too, block #4
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// add some more L1 blocks #5, #6
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)

	// and more unsafe L2 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// move safe/finalize markers: finalize the L1 chain block with the first batch, but not the second
	miner.ActL1SafeNext(t)     // #2 -> #3
	miner.ActL1SafeNext(t)     // #3 -> #4
	miner.ActL1FinalizeNext(t) // #1 -> #2
	miner.ActL1FinalizeNext(t) // #2 -> #3

	sequencer.ActL2PipelineFull(t)
	sequencer.ActL1FinalizedSignal(t)
	sequencer.ActL1SafeSignal(t)
	sequencer.ActL1HeadSignal(t)
	require.Equal(t, uint64(6), sequencer.SyncStatus().HeadL1.Number)
	require.Equal(t, uint64(4), sequencer.SyncStatus().SafeL1.Number)
	require.Equal(t, uint64(3), sequencer.SyncStatus().FinalizedL1.Number)
	require.Equal(t, heightToSubmit, sequencer.SyncStatus().FinalizedL2.Number, "finalized L2 blocks in first batch")

	// need to act with the engine on the signals still
	sequencer.ActL2PipelineFull(t)

	engCl := engine.EngineClient(t, sd.RollupCfg)
	engBlock, err := engCl.L2BlockRefByLabel(t.Ctx(), eth.Finalized)
	require.NoError(t, err)
	require.Equal(t, heightToSubmit, engBlock.Number, "engine finalizes what rollup node finalizes")

	// Now try to finalize block 4, but with a bad/malicious alternative hash.
	// If we get this false signal, we shouldn't finalize the L2 chain.
	altBlock4 := sequencer.SyncStatus().SafeL1
	altBlock4.Hash = common.HexToHash("0xdead")
	sequencer.derivation.Finalize(altBlock4)
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, uint64(3), sequencer.SyncStatus().FinalizedL1.Number)
	require.Equal(t, heightToSubmit, sequencer.SyncStatus().FinalizedL2.Number, "unknown/bad finalized L1 blocks are ignored")
}
