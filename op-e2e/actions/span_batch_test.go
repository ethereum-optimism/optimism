package actions

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestDropSpanBatchBeforeHardfork tests behavior of op-node before SpanBatch hardfork.
// op-node must drop SpanBatch before SpanBatch hardfork.
func TestDropSpanBatchBeforeHardfork(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20, // larger than L1 block time we simulate in this test (12)
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
		L1BlockTime:         12,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	// do not activate SpanBatch hardfork for verifier
	dp.DeployConfig.L2GenesisSpanBatchTimeOffset = nil
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), &sync.Config{})

	rollupSeqCl := sequencer.RollupClient()
	dp2 := e2eutils.MakeDeployParams(t, p)
	minTs := hexutil.Uint64(0)
	// activate SpanBatch hardfork for batcher. so batcher will submit SpanBatches to L1.
	dp2.DeployConfig.L2GenesisSpanBatchTimeOffset = &minTs
	sd2 := e2eutils.Setup(t, dp2, defaultAlloc)
	batcher := NewL2Batcher(log, sd2.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	// Alice makes a L2 tx
	cl := seqEngine.EthClient()
	n, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
	require.NoError(t, err)
	signer := types.LatestSigner(sd.L2Cfg.Config)
	tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   sd.L2Cfg.Config.ChainID,
		Nonce:     n,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: new(big.Int).Add(miner.l1Chain.CurrentBlock().BaseFee, big.NewInt(2*params.GWei)),
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

	// batch submit to L1. batcher should submit span batches.
	batcher.ActL2BatchBuffer(t)
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)

	// confirm batch on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)
	bl := miner.l1Chain.CurrentBlock()
	log.Info("bl", "txs", len(miner.l1Chain.GetBlockByHash(bl.Hash()).Transactions()))

	// Now make enough L1 blocks that the verifier will have to derive a L2 block
	// It will also eagerly derive the block from the batcher
	for i := uint64(0); i < sd.RollupCfg.SeqWindowSize; i++ {
		miner.ActL1StartBlock(12)(t)
		miner.ActL1EndBlock(t)
	}

	// try to sync verifier from L1 batch. but verifier should drop every span batch.
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, uint64(1), verifier.SyncStatus().SafeL2.L1Origin.Number)

	verifCl := verifEngine.EthClient()
	for i := int64(1); i < int64(verifier.L2Safe().Number); i++ {
		block, _ := verifCl.BlockByNumber(t.Ctx(), big.NewInt(i))
		require.NoError(t, err)
		// because verifier drops every span batch, it should generate empty blocks.
		// so every block has only L1 attribute deposit transaction.
		require.Equal(t, block.Transactions().Len(), 1)
	}
	// check that the tx from alice is not included in verifier's chain
	_, _, err = verifCl.TransactionByHash(t.Ctx(), tx.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)
}

// TestAcceptSingularBatchAfterHardfork tests behavior of op-node after SpanBatch hardfork.
// op-node must accept SingularBatch after SpanBatch hardfork.
func TestAcceptSingularBatchAfterHardfork(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20, // larger than L1 block time we simulate in this test (12)
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
		L1BlockTime:         12,
	}
	minTs := hexutil.Uint64(0)
	dp := e2eutils.MakeDeployParams(t, p)

	// activate SpanBatch hardfork for verifier.
	dp.DeployConfig.L2GenesisSpanBatchTimeOffset = &minTs
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), &sync.Config{})

	rollupSeqCl := sequencer.RollupClient()
	dp2 := e2eutils.MakeDeployParams(t, p)

	// not activate SpanBatch hardfork for batcher
	dp2.DeployConfig.L2GenesisSpanBatchTimeOffset = nil
	sd2 := e2eutils.Setup(t, dp2, defaultAlloc)
	batcher := NewL2Batcher(log, sd2.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	// Alice makes a L2 tx
	cl := seqEngine.EthClient()
	n, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
	require.NoError(t, err)
	signer := types.LatestSigner(sd.L2Cfg.Config)
	tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   sd.L2Cfg.Config.ChainID,
		Nonce:     n,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: new(big.Int).Add(miner.l1Chain.CurrentBlock().BaseFee, big.NewInt(2*params.GWei)),
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

	// batch submit to L1. batcher should submit singular batches.
	batcher.ActL2BatchBuffer(t)
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)

	// confirm batch on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)
	bl := miner.l1Chain.CurrentBlock()
	log.Info("bl", "txs", len(miner.l1Chain.GetBlockByHash(bl.Hash()).Transactions()))

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

// TestSpanBatchEmptyChain tests derivation of empty chain using SpanBatch.
func TestSpanBatchEmptyChain(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20,
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
		L1BlockTime:         12,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	minTs := hexutil.Uint64(0)
	// Activate SpanBatch hardfork
	dp.DeployConfig.L2GenesisSpanBatchTimeOffset = &minTs
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), &sync.Config{})

	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	miner.ActEmptyBlock(t)
	// Make 1200 empty L2 blocks (L1BlockTime / L2BlockTime * 100)
	for i := 0; i < 100; i++ {
		sequencer.ActL1HeadSignal(t)
		sequencer.ActBuildToL1Head(t)

		if i%10 == 9 {
			// batch submit to L1
			batcher.ActSubmitAll(t)

			// confirm batch on L1
			miner.ActL1StartBlock(12)(t)
			miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
			miner.ActL1EndBlock(t)
		} else {
			miner.ActEmptyBlock(t)
		}
	}
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)

	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)

	require.Equal(t, sequencer.L2Unsafe(), sequencer.L2Safe())
	require.Equal(t, verifier.L2Unsafe(), verifier.L2Safe())
	require.Equal(t, sequencer.L2Safe(), verifier.L2Safe())
}

// TestSpanBatchLowThroughputChain tests derivation of low-throughput chain using SpanBatch.
func TestSpanBatchLowThroughputChain(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20,
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
		L1BlockTime:         12,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	minTs := hexutil.Uint64(0)
	// Activate SpanBatch hardfork
	dp.DeployConfig.L2GenesisSpanBatchTimeOffset = &minTs
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlError)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), &sync.Config{})

	rollupSeqCl := sequencer.RollupClient()
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))
	cl := seqEngine.EthClient()

	const numTestUsers = 5
	var privKeys [numTestUsers]*ecdsa.PrivateKey
	var addrs [numTestUsers]common.Address
	for i := 0; i < numTestUsers; i++ {
		// Create a new test account
		privateKey, err := dp.Secrets.Wallet.PrivateKey(accounts.Account{
			URL: accounts.URL{
				Path: fmt.Sprintf("m/44'/60'/0'/0/%d", 10+i),
			},
		})
		privKeys[i] = privateKey
		addr := crypto.PubkeyToAddress(privateKey.PublicKey)
		require.NoError(t, err)
		addrs[i] = addr
	}

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	miner.ActEmptyBlock(t)
	totalTxCount := 0
	// Make 600 L2 blocks (L1BlockTime / L2BlockTime * 50) including 1~3 txs
	for i := 0; i < 50; i++ {
		sequencer.ActL1HeadSignal(t)
		for sequencer.derivation.UnsafeL2Head().L1Origin.Number < sequencer.l1State.L1Head().Number {
			sequencer.ActL2PipelineFull(t)
			sequencer.ActL2StartBlock(t)
			// fill the block with random number of L2 txs
			for j := 0; j < rand.Intn(3); j++ {
				userIdx := totalTxCount % numTestUsers
				signer := types.LatestSigner(sd.L2Cfg.Config)
				data := make([]byte, rand.Intn(100))
				_, err := crand.Read(data[:]) // fill with random bytes
				require.NoError(t, err)
				gas, err := core.IntrinsicGas(data, nil, false, true, true, false)
				require.NoError(t, err)
				baseFee := seqEngine.l2Chain.CurrentBlock().BaseFee
				nonce, err := cl.PendingNonceAt(t.Ctx(), addrs[userIdx])
				require.NoError(t, err)
				tx := types.MustSignNewTx(privKeys[userIdx], signer, &types.DynamicFeeTx{
					ChainID:   sd.L2Cfg.Config.ChainID,
					Nonce:     nonce,
					GasTipCap: big.NewInt(2 * params.GWei),
					GasFeeCap: new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), big.NewInt(2*params.GWei)),
					Gas:       gas,
					To:        &dp.Addresses.Bob,
					Value:     big.NewInt(0),
					Data:      data,
				})
				require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))
				seqEngine.ActL2IncludeTx(addrs[userIdx])(t)
				totalTxCount += 1
			}
			sequencer.ActL2EndBlock(t)
		}

		if i%10 == 9 {
			// batch submit to L1
			batcher.ActSubmitAll(t)

			// confirm batch on L1
			miner.ActL1StartBlock(12)(t)
			miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
			miner.ActL1EndBlock(t)
		} else {
			miner.ActEmptyBlock(t)
		}
	}
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)

	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)

	require.Equal(t, sequencer.L2Unsafe(), sequencer.L2Safe())
	require.Equal(t, verifier.L2Unsafe(), verifier.L2Safe())
	require.Equal(t, sequencer.L2Safe(), verifier.L2Safe())
}
