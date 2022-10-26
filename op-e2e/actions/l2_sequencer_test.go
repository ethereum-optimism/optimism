package actions

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func setupSequencerTest(t Testing, sd *e2eutils.SetupData, log log.Logger) (*L1Miner, *L2Engine, *L2Sequencer) {
	jwtPath := e2eutils.WriteDefaultJWT(t)

	miner := NewL1Miner(t, log, sd.L1Cfg)

	l1F, err := sources.NewL1Client(miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false))
	require.NoError(t, err)
	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
	l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	sequencer := NewL2Sequencer(t, log, l1F, l2Cl, sd.RollupCfg, 0)
	return miner, engine, sequencer
}

func TestL2Sequencer_SequencerDrift(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   20, // larger than L1 block time we simulate in this test (12)
		SequencerWindowSize: 24,
		ChannelTimeout:      20,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, engine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})

	sequencer.ActL2PipelineFull(t)

	signer := types.LatestSigner(sd.L2Cfg.Config)
	cl := engine.EthClient()
	aliceTx := func() {
		n, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
		require.NoError(t, err)
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
	}
	makeL2BlockWithAliceTx := func() {
		aliceTx()
		sequencer.ActL2StartBlock(t)
		engine.ActL2IncludeTx(dp.Addresses.Alice)(t) // include a test tx from alice
		sequencer.ActL2EndBlock(t)
	}

	// L1 makes a block
	miner.ActL1StartBlock(12)(t)
	miner.ActL1EndBlock(t)
	sequencer.ActL1HeadSignal(t)
	origin := miner.l1Chain.CurrentBlock()

	// L2 makes blocks to catch up
	for sequencer.SyncStatus().UnsafeL2.Time+sd.RollupCfg.BlockTime < origin.Time() {
		makeL2BlockWithAliceTx()
		require.Equal(t, uint64(0), sequencer.SyncStatus().UnsafeL2.L1Origin.Number, "no L1 origin change before time matches")
	}
	// Check that we adopted the origin as soon as we could (conf depth is 0)
	makeL2BlockWithAliceTx()
	require.Equal(t, uint64(1), sequencer.SyncStatus().UnsafeL2.L1Origin.Number, "L1 origin changes as soon as L2 time equals or exceeds L1 time")

	miner.ActL1StartBlock(12)(t)
	miner.ActL1EndBlock(t)
	sequencer.ActL1HeadSignal(t)

	// Make blocks up till the sequencer drift is about to surpass, but keep the old L1 origin
	for sequencer.SyncStatus().UnsafeL2.Time+sd.RollupCfg.BlockTime < origin.Time()+sd.RollupCfg.MaxSequencerDrift {
		sequencer.ActL2KeepL1Origin(t)
		makeL2BlockWithAliceTx()
		require.Equal(t, uint64(1), sequencer.SyncStatus().UnsafeL2.L1Origin.Number, "expected to keep old L1 origin")
	}

	// We passed the sequencer drift: we can still keep the old origin, but can't include any txs
	sequencer.ActL2KeepL1Origin(t)
	sequencer.ActL2StartBlock(t)
	require.True(t, engine.l2ForceEmpty, "engine should not be allowed to include anything after sequencer drift is surpassed")
}
