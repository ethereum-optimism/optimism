package proofs_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPectraBlobSchedule(gt *testing.T) {
	t := actionsHelpers.NewDefaultTesting(gt)
	testCfg := &helpers.TestCfg[any]{
		Hardfork: helpers.Holocene,
	}
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		// fix blob schedule after 2 L1 blocks
		dc.L2GenesisPectraBlobScheduleTimeOffset = ptr(hexutil.Uint64(24))
		dc.L1GenesisBlockExcessBlobGas = ptr(hexutil.Uint64(1e8))
	}

	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	// sanity check
	require.Equal(t, *env.Sd.RollupCfg.PectraBlobScheduleTime, env.Sd.L2Cfg.Timestamp+24)

	engine := env.Engine
	sequencer := env.Sequencer
	miner := env.Miner

	l1_0 := miner.L1Chain().CurrentHeader()
	t.Logf("Header0: Number: %v, Time: %v, ExcessBlobGas: %v", l1_0.Number, l1_0.Time, *l1_0.ExcessBlobGas)
	require.NotZero(t, *l1_0.ExcessBlobGas)
	require.Equal(t, env.Sd.L2Cfg.Timestamp, l1_0.Time, "we assume L1 and L2 genesis have the same time")

	ethCl := engine.EthClient()
	l1Block, err := legacybindings.NewL1Block(predeploys.L1BlockAddr, ethCl)
	require.NoError(t, err)

	miner.ActEmptyBlock(t)
	l1_1 := miner.L1Chain().CurrentHeader()
	t.Logf("Header1: Number: %v, Time: %v, ExcessBlobGas: %v", l1_1.Number, l1_1.Time, *l1_1.ExcessBlobGas)
	require.Less(t, l1_1.Time, *env.Sd.RollupCfg.PectraBlobScheduleTime)

	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadUnsafe(t)

	opts := &bind.CallOpts{}
	bbf1, err := l1Block.BlobBaseFee(opts)
	t.Logf("BlobBaseFee1: %v", bbf1)
	require.NoError(t, err)
	cancunBBF1 := eth.CalcBlobFeeCancun(*l1_1.ExcessBlobGas)
	pragueBBF1 := eth.CalcBlobFeeDefault(l1_1)
	require.Equal(t, cancunBBF1, bbf1)
	require.Less(t, pragueBBF1.Uint64(), cancunBBF1.Uint64())

	miner.ActEmptyBlock(t)
	l1_2 := miner.L1Chain().CurrentHeader()
	t.Logf("Header2: Number: %v, Time: %v, ExcessBlobGas: %v", l1_2.Number, l1_2.Time, *l1_2.ExcessBlobGas)
	require.Equal(t, l1_2.Time, *env.Sd.RollupCfg.PectraBlobScheduleTime)

	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadUnsafe(t)

	bbf2, err := l1Block.BlobBaseFee(opts)
	t.Logf("BlobBaseFee2: %v", bbf2)
	require.NoError(t, err)
	cancunBBF2 := eth.CalcBlobFeeCancun(*l1_2.ExcessBlobGas)
	pragueBBF2 := eth.CalcBlobFeeDefault(l1_2)
	require.Equal(t, pragueBBF2, bbf2)
	require.Less(t, pragueBBF2.Uint64(), cancunBBF2.Uint64())
	l2UnsafeHead := env.Engine.L2Chain().CurrentHeader()

	env.BatchAndMine(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, eth.HeaderBlockInfo(l2SafeHead), eth.HeaderBlockInfo(l2UnsafeHead))
}

func ptr[T any](v T) *T {
	return &v
}
