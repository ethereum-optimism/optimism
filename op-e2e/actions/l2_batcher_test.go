package actions

import (
	"crypto/rand"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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
	// It will also eagerly derive the block from the batcher
	for i := uint64(0); i < sd.RollupCfg.SeqWindowSize; i++ {
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

// TestL2FinalizationWithSparseL1 tests that safe L2 blocks can be finalized even if we do not regularly get a L1 finalization signal
func TestL2FinalizationWithSparseL1(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, engine, sequencer := setupSequencerTest(t, sd, log)

	sequencer.ActL2PipelineFull(t)

	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	startStatus := sequencer.SyncStatus()
	require.Less(t, startStatus.SafeL2.Number, startStatus.UnsafeL2.Number, "sequencer has unsafe L2 block")

	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), engine.EthClient())
	batcher.ActSubmitAll(t)

	// include in L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// Make 2 L1 blocks without batches
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)

	// See the L1 head, and traverse the pipeline to it
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)

	updatedStatus := sequencer.SyncStatus()
	require.Equal(t, updatedStatus.SafeL2.Number, updatedStatus.UnsafeL2.Number, "unsafe L2 block is now safe")
	require.Less(t, updatedStatus.FinalizedL2.Number, updatedStatus.UnsafeL2.Number, "submitted block is not yet finalized")

	// Now skip straight to the head with L1 signals (sequencer has traversed the L1 blocks, but they did not have L2 contents)
	headL1Num := miner.UnsafeNum()
	miner.ActL1Safe(t, headL1Num)
	miner.ActL1Finalize(t, headL1Num)
	sequencer.ActL1SafeSignal(t)
	sequencer.ActL1FinalizedSignal(t)

	// Now see if the signals can be processed
	sequencer.ActL2PipelineFull(t)

	finalStatus := sequencer.SyncStatus()
	// Verify the signal was processed, even though we signalled a later L1 block than the one with the batch.
	require.Equal(t, finalStatus.FinalizedL2.Number, finalStatus.UnsafeL2.Number, "sequencer submitted its L2 block and it finalized")
}

// TestGarbageBatch tests the behavior of an invalid/malformed output channel frame containing
// valid batches being submitted to the batch inbox. These batches should always be rejected
// and the safe L2 head should remain unaltered.
func TestGarbageBatch(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := defaultRollupTestParams
	dp := e2eutils.MakeDeployParams(t, p)
	for _, garbageKind := range GarbageKinds {
		sd := e2eutils.Setup(t, dp, defaultAlloc)
		log := testlog.Logger(t, log.LvlError)
		miner, engine, sequencer := setupSequencerTest(t, sd, log)

		_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))

		batcherCfg := &BatcherCfg{
			MinL1TxSize: 0,
			MaxL1TxSize: 128_000,
			BatcherKey:  dp.Secrets.Batcher,
		}

		if garbageKind == MALFORM_RLP || garbageKind == INVALID_COMPRESSION {
			// If the garbage kind is `INVALID_COMPRESSION` or `MALFORM_RLP`, use the `actions` packages
			// modified `ChannelOut`.
			batcherCfg.GarbageCfg = &GarbageChannelCfg{
				useInvalidCompression: garbageKind == INVALID_COMPRESSION,
				malformRLP:            garbageKind == MALFORM_RLP,
			}
		}

		batcher := NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient())

		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		syncAndBuildL2 := func() {
			// Send a head signal to the sequencer and verifier
			sequencer.ActL1HeadSignal(t)
			verifier.ActL1HeadSignal(t)

			// Run the derivation pipeline on the sequencer and verifier
			sequencer.ActL2PipelineFull(t)
			verifier.ActL2PipelineFull(t)

			// Build the L2 chain to the L1 head
			sequencer.ActBuildToL1Head(t)
		}

		// Build an empty block on L1 and run the derivation pipeline + build L2
		// to the L1 head (block #1)
		miner.ActEmptyBlock(t)
		syncAndBuildL2()

		// Ensure that the L2 safe head has an L1 Origin at genesis before any
		// batches are submitted.
		require.Equal(t, uint64(0), sequencer.L2Safe().L1Origin.Number)
		require.Equal(t, uint64(1), sequencer.L2Unsafe().L1Origin.Number)

		// Submit a batch containing all blocks built on L2 while catching up
		// to the L1 head above. The output channel frame submitted to the batch
		// inbox will be invalid- it will be malformed depending on the passed
		// `garbageKind`.
		batcher.ActBufferAll(t)
		batcher.ActL2ChannelClose(t)
		batcher.ActL2BatchSubmitGarbage(t, garbageKind)

		// Include the batch on L1 in block #2
		miner.ActL1StartBlock(12)(t)
		miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
		miner.ActL1EndBlock(t)

		// Send a head signal + run the derivation pipeline on the sequencer
		// and verifier.
		syncAndBuildL2()

		// Verify that the L2 blocks that were batch submitted were *not* marked
		// as safe due to the malformed output channel frame. The safe head should
		// still have an L1 Origin at genesis.
		require.Equal(t, uint64(0), sequencer.L2Safe().L1Origin.Number)
		require.Equal(t, uint64(2), sequencer.L2Unsafe().L1Origin.Number)
	}
}

func TestExtendedTimeWithoutL1Batches(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20, // larger than L1 block time we simulate in this test (12)
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, engine, sequencer := setupSequencerTest(t, sd, log)

	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))

	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), engine.EthClient())

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// make a long L1 chain, up to just one block left for L2 blocks to be included.
	for i := uint64(0); i < p.SequencerWindowSize-1; i++ {
		miner.ActEmptyBlock(t)
	}

	// Now build a L2 chain that references all of these L1 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// Now submit all the L2 blocks in the very last L1 block within sequencer window range
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// Now sync the verifier, and see if the L2 chain of the sequencer is safe
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Safe(), "all L2 blocks should have been included just in time")
	sequencer.ActL2PipelineFull(t)
	require.Equal(t, sequencer.L2Unsafe(), sequencer.L2Safe(), "same for sequencer")
}

// TestBigL2Txs tests a high-throughput case with constrained batcher:
//   - Fill 100 L2 blocks to near max-capacity, with txs of 120 KB each
//   - Buffer the L2 blocks into channels together as much as possible, submit data-txs only when necessary
//     (just before crossing the max RLP channel size)
//   - Limit the data-tx size to 40 KB, to force data to be split across multiple datat-txs
//   - Defer all data-tx inclusion till the end
//   - Fill L1 blocks with data-txs until we have processed them all
//   - Run the verifier, and check if it derives the same L2 chain as was created by the sequencer.
//
// The goal of this test is to quickly run through an otherwise very slow process of submitting and including lots of data.
// This does not test the batcher code, but is really focused at testing the batcher utils
// and channel-decoding verifier code in the derive package.
func TestBigL2Txs(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   100,
		SequencerWindowSize: 1000,
		ChannelTimeout:      200, // give enough space to buffer large amounts of data before submitting it
	}
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlInfo)
	miner, engine, sequencer := setupSequencerTest(t, sd, log)

	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))

	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 40_000, // try a small batch size, to force the data to be split between more frames
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), engine.EthClient())

	sequencer.ActL2PipelineFull(t)

	verifier.ActL2PipelineFull(t)
	cl := engine.EthClient()

	batcherNonce := uint64(0) // manually track batcher nonce. the "pending nonce" value in tx-pool is incorrect after we fill the pending-block gas limit and keep adding txs to the pool.
	batcherTxOpts := func(tx *types.DynamicFeeTx) {
		tx.Nonce = batcherNonce
		batcherNonce++
		tx.GasFeeCap = e2eutils.Ether(1) // be very generous with basefee, since we're spamming L1
	}

	// build many L2 blocks filled to the brim with large txs of random data
	for i := 0; i < 100; i++ {
		aliceNonce, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
		status := sequencer.SyncStatus()
		// build empty L1 blocks as necessary, so the L2 sequencer can continue to include txs while not drifting too far out
		if status.UnsafeL2.Time >= status.HeadL1.Time+12 {
			miner.ActEmptyBlock(t)
		}
		sequencer.ActL1HeadSignal(t)
		sequencer.ActL2StartBlock(t)
		baseFee := engine.l2Chain.CurrentBlock().BaseFee() // this will go quite high, since so many consecutive blocks are filled at capacity.
		// fill the block with large L2 txs from alice
		for n := aliceNonce; ; n++ {
			require.NoError(t, err)
			signer := types.LatestSigner(sd.L2Cfg.Config)
			data := make([]byte, 120_000) // very large L2 txs, as large as the tx-pool will accept
			_, err := rand.Read(data[:])  // fill with random bytes, to make compression ineffective
			require.NoError(t, err)
			gas, err := core.IntrinsicGas(data, nil, false, true, true, false)
			require.NoError(t, err)
			if gas > engine.l2GasPool.Gas() {
				break
			}
			tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
				ChainID:   sd.L2Cfg.Config.ChainID,
				Nonce:     n,
				GasTipCap: big.NewInt(2 * params.GWei),
				GasFeeCap: new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), big.NewInt(2*params.GWei)),
				Gas:       gas,
				To:        &dp.Addresses.Bob,
				Value:     big.NewInt(0),
				Data:      data,
			})
			require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))
			engine.ActL2IncludeTx(dp.Addresses.Alice)(t)
		}
		sequencer.ActL2EndBlock(t)
		for batcher.l2BufferedBlock.Number < sequencer.SyncStatus().UnsafeL2.Number {
			// if we run out of space, close the channel and submit all the txs
			if err := batcher.Buffer(t); errors.Is(err, derive.ErrTooManyRLPBytes) {
				log.Info("flushing filled channel to batch txs", "id", batcher.l2ChannelOut.ID())
				batcher.ActL2ChannelClose(t)
				for batcher.l2ChannelOut != nil {
					batcher.ActL2BatchSubmit(t, batcherTxOpts)
				}
			}
		}
	}

	// if anything is left in the channel, submit it
	if batcher.l2ChannelOut != nil {
		log.Info("flushing trailing channel to batch txs", "id", batcher.l2ChannelOut.ID())
		batcher.ActL2ChannelClose(t)
		for batcher.l2ChannelOut != nil {
			batcher.ActL2BatchSubmit(t, batcherTxOpts)
		}
	}

	// build L1 blocks until we're out of txs
	txs, _ := miner.eth.TxPool().ContentFrom(dp.Addresses.Batcher)
	for {
		if len(txs) == 0 {
			break
		}
		miner.ActL1StartBlock(12)(t)
		for range txs {
			if len(txs) == 0 {
				break
			}
			tx := txs[0]
			if miner.l1GasPool.Gas() < tx.Gas() { // fill the L1 block with batcher txs until we run out of gas
				break
			}
			log.Info("including batcher tx", "nonce", tx)
			miner.IncludeTx(t, tx)
			txs = txs[1:]
		}
		miner.ActL1EndBlock(t)
	}
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().SafeL2, "verifier synced sequencer data even though of huge tx in block")
}
