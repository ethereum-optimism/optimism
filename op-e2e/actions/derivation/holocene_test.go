package derivation

import (
	"math/big"
	"math/rand"
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	upgradesHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/upgrades/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestOutOfOrderFrames uses batcher.ActL2BatchSubmitOutOfOrder
// To buffer an entire channel and submit it in reverse order. The
// surrounding test ensures that the safe head either does or does
// not progress, depending on whether Holocene is activated in the
// verifier node.
func TestOutOfOrderFrames(gt *testing.T) {
	t := actionsHelpers.NewDefaultTesting(gt)
	p := actionsHelpers.DefaultRollupTestParams()
	dp := e2eutils.MakeDeployParams(t, p)
	upgradesHelpers.ApplyDeltaTimeOffset(dp, nil) // disable span bacthes for now

	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	log := testlog.Logger(t, log.LevelInfo)
	miner, engine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)

	_, verifier := actionsHelpers.SetupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{})

	batcherCfg := actionsHelpers.DefaultBatcherCfg(dp)

	batcher := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engine.EngineClient(t, sd.RollupCfg))

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build an L2 block filled to the brim with large txs of random data
	rng := rand.New(rand.NewSource(555))
	cl := engine.EthClient()
	aliceNonce, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
	status := sequencer.SyncStatus()
	// build empty L1 blocks as necessary, so the L2 sequencer can continue to include txs while not drifting too far out
	if status.UnsafeL2.Time >= status.HeadL1.Time+12 {
		miner.ActEmptyBlock(t)
	}
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2StartBlock(t)
	baseFee := engine.L2Chain().CurrentBlock().BaseFee // this will go quite high, since so many consecutive blocks are filled at capacity.
	// fill the block with large L2 txs from alice
	for n := aliceNonce; ; n++ {
		require.NoError(t, err)
		signer := types.LatestSigner(sd.L2Cfg.Config)
		data := make([]byte, 120_000) // very large L2 txs, as large as the tx-pool will accept
		_, err := rng.Read(data[:])   // fill with random bytes, to make compression ineffective
		require.NoError(t, err)
		gas, err := core.IntrinsicGas(data, nil, false, true, true, false)
		require.NoError(t, err)
		if gas > engine.EngineApi.RemainingBlockGas() {
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
		require.NoError(t, cl.SendTransaction(t.Ctx(), tx))
		engine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	}
	sequencer.ActL2EndBlock(t)

	// Ensure that the L2 safe head has an L1 Origin at genesis before any
	// batches are submitted.
	require.Equal(t, uint64(0), verifier.L2Safe().Number)

	// Here's where we trigger the unusual behaviour on the batcher
	batcher.ActBufferAll(t)
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmitOutOfOrder(t)

	// build L1 blocks until we're out of txs
	txs, _ := miner.Eth.TxPool().ContentFrom(dp.Addresses.Batcher)
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
			if miner.L1GasPool.Gas() < tx.Gas() { // fill the L1 block with batcher txs until we run out of gas
				break
			}
			log.Info("including batcher tx", "nonce", tx.Nonce())
			miner.IncludeTx(t, tx)
			txs = txs[1:]
		}
		miner.ActL1EndBlock(t)
	}

	// Send a head signal + run the derivation pipeline on the sequencer
	// and verifier.
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)

	// Verify that the L2 blocks that were batch submitted were either
	holocene := false
	if holocene {
		// NOT marked as safe due to the out of order frame submission. The safe head should
		// still have an L1 Origin at genesis.
		require.Equal(t, uint64(0), verifier.L2Safe().Number)
	} else {
		// Marked as safe due to the derivation pipeline buffering frames
		// which arrive  out of order. The safe head should
		// advance.
		require.Equal(t, uint64(1), verifier.L2Safe().Number)
	}
}
