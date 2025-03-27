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

type pectraBlobScheduleTestCfg struct {
	offset          *uint64
	expectCancunBBF bool
}

func Test_ProgramAction_PectraBlobSchedule(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddDefaultTestCases(
		// aligned with an L1 timestamp
		pectraBlobScheduleTestCfg{ptr(uint64(24)), true},
		helpers.NewForkMatrix(helpers.Holocene, helpers.Isthmus),
		testPectraBlobSchedule,
	).AddDefaultTestCases(
		// in the middle between two L1 timestamps
		pectraBlobScheduleTestCfg{ptr(uint64(18)), true},
		helpers.NewForkMatrix(helpers.Holocene),
		testPectraBlobSchedule,
	).AddDefaultTestCases(
		pectraBlobScheduleTestCfg{nil, false},
		helpers.NewForkMatrix(helpers.Holocene, helpers.Isthmus),
		testPectraBlobSchedule,
	)
}

func testPectraBlobSchedule(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	tcfg := testCfg.Custom.(pectraBlobScheduleTestCfg) // two flavors of this test
	t := actionsHelpers.NewDefaultTesting(gt)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.L2GenesisPectraBlobScheduleTimeOffset = (*hexutil.Uint64)(tcfg.offset)
		// set genesis excess blob gas so there are >0 blob base fees for some blocks
		dc.L1GenesisBlockExcessBlobGas = ptr(hexutil.Uint64(1e8))
	}

	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	// sanity check
	if tcfg.offset != nil {
		require.Equal(t, *env.Sd.RollupCfg.PectraBlobScheduleTime, env.Sd.L2Cfg.Timestamp+*tcfg.offset)
	}

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
	if tcfg.offset != nil {
		require.Less(t, l1_1.Time, *env.Sd.RollupCfg.PectraBlobScheduleTime)
	}

	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadUnsafe(t)

	cancunBBF1 := eth.CalcBlobFeeCancun(*l1_1.ExcessBlobGas)
	pragueBBF1 := eth.CalcBlobFeeDefault(l1_1)
	// Make sure they differ.
	require.Less(t, pragueBBF1.Uint64(), cancunBBF1.Uint64())
	opts := &bind.CallOpts{}
	bbf1, err := l1Block.BlobBaseFee(opts)
	require.NoError(t, err)
	t.Logf("BlobBaseFee1: %v", bbf1)
	// This is the critical assertion of this test. With the PectraBlobSchedule set, the blob
	// base fee is still calculated using the Cancun schedule, without it with the same as the
	// Prague schedule of L1.
	if tcfg.expectCancunBBF {
		require.Equal(t, cancunBBF1, bbf1)
	} else {
		require.Equal(t, pragueBBF1, bbf1)
	}

	miner.ActEmptyBlock(t)
	l1_2 := miner.L1Chain().CurrentHeader()
	t.Logf("Header2: Number: %v, Time: %v, ExcessBlobGas: %v", l1_2.Number, l1_2.Time, *l1_2.ExcessBlobGas)
	if tcfg.offset != nil {
		if *tcfg.offset%12 == 0 {
			require.Equal(t, l1_2.Time, *env.Sd.RollupCfg.PectraBlobScheduleTime)
		} else {
			require.Greater(t, l1_2.Time, *env.Sd.RollupCfg.PectraBlobScheduleTime)
		}
	}

	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadUnsafe(t)

	cancunBBF2 := eth.CalcBlobFeeCancun(*l1_2.ExcessBlobGas)
	pragueBBF2 := eth.CalcBlobFeeDefault(l1_2)
	require.Less(t, pragueBBF2.Uint64(), cancunBBF2.Uint64())
	bbf2, err := l1Block.BlobBaseFee(opts)
	require.NoError(t, err)
	t.Logf("BlobBaseFee2: %v", bbf2)
	require.Equal(t, pragueBBF2, bbf2)
	l2UnsafeHead := env.Engine.L2Chain().CurrentHeader()

	env.BatchAndMine(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()
	require.Equal(t, eth.HeaderBlockID(l2SafeHead), eth.HeaderBlockID(l2UnsafeHead), "derivation leads to the same block")

	env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
}

func ptr[T any](v T) *T {
	return &v
}
