package proofs

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// setDAFootprintGasScalarViaSystemConfig sets the DA footprint gas scalar on the L1 SystemConfig and mines it.
func setDAFootprintGasScalarViaSystemConfig(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, scalar uint16) {
	systemConfig, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
	require.NoError(t, err)

	txOpts, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
	require.NoError(t, err)
	t.Logf("Setting DA footprint gas scalar on L1: scalar=%d", scalar)

	env.Miner.ActL1StartBlock(12)(t)
	_, err = systemConfig.SetDAFootprintGasScalar(txOpts, scalar)
	require.NoError(t, err, "SetDAFootprintGasScalar transaction failed")
	env.Miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	env.Miner.ActL1EndBlock(t)
}

func requireL1BlockDAFootprintGasScalarEquals(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, scalar uint16) {
	t.Helper()
	l1Block, err := bindings.NewL1BlockCaller(predeploys.L1BlockAddr, env.Engine.EthClient())
	require.NoError(t, err)
	l1BlockScalar, err := l1Block.DaFootprintGasScalar(nil)
	require.NoError(t, err)
	require.Equal(t, scalar, l1BlockScalar, "L1 block DA footprint gas scalar mismatch")
}

func Test_ProgramAction_JovianDAFootprint(gt *testing.T) {
	const lowDAFootprintGasScalar = 30
	// Builds a block whose DA footprint is high (near but below gas limit),
	// then verifies header BlobGasUsed and next block basefee calculation.
	runStep := func(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, setScalar *uint16) {
		// Optionally set the DA footprint scalar on L1, then propagate to L2.
		effectiveScalar := uint16(derive.DAFootprintGasScalarDefault)
		if setScalar != nil {
			setDAFootprintGasScalarViaSystemConfig(t, env, *setScalar)
			if *setScalar != 0 {
				effectiveScalar = *setScalar
			}

			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActL2PipelineFull(t)
			env.Sequencer.ActBuildToL1Head(t)
			// Move one block past origin transition to apply the config change to new blocks
			env.Sequencer.ActL2EmptyBlock(t)
		}
		isLowScalar := effectiveScalar == lowDAFootprintGasScalar

		// Prepare to assemble a single L2 block close to the gas limit by including
		// multiple user-space txs with deterministic calldata size.
		rollupCfg := env.Sequencer.RollupCfg
		gasLimit := rollupCfg.Genesis.SystemConfig.GasLimit

		l2Cl := env.Engine.EthClient()
		chainID := rollupCfg.L2ChainID
		signer := types.LatestSigner(env.Sd.L2Cfg.Config)

		// Deterministic nonces from pending state
		nonce, err := l2Cl.PendingNonceAt(t.Ctx(), env.Dp.Addresses.Alice)
		require.NoError(t, err)

		// Use a fixed-size deterministic payload; large enough to quickly reach target
		// but small enough to pack several txs.
		const dataSize = 5_000
		payload := make([]byte, dataSize)
		// deterministic fill to avoid compression variance across runs
		rng := rand.New(rand.NewSource(33))
		_, err = rng.Read(payload)
		require.NoError(t, err)

		var runningDAFootprint, runningGas uint64
		var builtTxs []*types.Transaction

		env.Sequencer.ActL2StartBlock(t)

		for {
			// Construct a tx with the given calldata.
			txData := &types.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     nonce,
				To:        &common.Address{},
				Gas:       params.TxGas + 40*dataSize, // cover floor calldata gas
				GasFeeCap: big.NewInt(5_000_000_000),  // 5 gwei
				GasTipCap: big.NewInt(2),
				Value:     big.NewInt(0),
				Data:      payload,
			}
			tx := types.MustSignNewTx(env.Dp.Secrets.Alice, signer, txData)

			// Estimate incremental DA footprint if we include this tx.
			est := tx.RollupCostData().EstimatedDASize().Uint64() * uint64(effectiveScalar)
			if runningDAFootprint+est > gasLimit {
				break
			} else if isLowScalar && runningGas+tx.Gas() > gasLimit {
				break
			}

			// Send to txpool then include in the in-progress L2 block.
			err = l2Cl.SendTransaction(t.Ctx(), tx)
			require.NoError(t, err)
			_, err = env.Engine.EngineApi.IncludeTx(tx, env.Dp.Addresses.Alice)
			require.NoError(t, err)

			runningDAFootprint += est
			runningGas += tx.Gas()
			builtTxs = append(builtTxs, tx)
			nonce++
		}

		// Ensure we actually packed some txs
		require.Greater(t, len(builtTxs), 0)

		env.Sequencer.ActL2EndBlock(t)

		requireL1BlockDAFootprintGasScalarEquals(t, env, effectiveScalar)

		// Inspect the built block and verify header DA footprint and base fee behavior.
		env.Sequencer.L2Unsafe()
		blk := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
		header := blk.Header()
		require.NotNil(t, header.BlobGasUsed, "blobGasUsed must be set on Jovian blocks")
		blockDAFootprint := *header.BlobGasUsed

		// Compute expected DA footprint from the actual included txs (skip deposits and system txs)
		var expectedDAFootprint uint64
		for _, tx := range blk.Transactions() {
			if tx.IsDepositTx() {
				continue
			}
			expectedDAFootprint += tx.RollupCostData().EstimatedDASize().Uint64() * uint64(effectiveScalar)
		}
		require.Equal(t, expectedDAFootprint, blockDAFootprint, "DA footprint mismatch with header")
		require.Less(t, blockDAFootprint, gasLimit, "DA footprint should be below gas limit")
		if !isLowScalar {
			require.Greater(t, blockDAFootprint, header.GasUsed, "DA footprint should exceed gas used")
		} else {
			require.Less(t, blockDAFootprint, header.GasUsed, "DA footprint should be less than gas used for low scalar")
		}

		// Verify base fee update uses DA footprint against the gas target
		gasTarget := gasLimit / rollupCfg.ChainOpConfig.EIP1559Elasticity
		parentBaseFee := blk.BaseFee()
		delta := new(big.Int).SetUint64(blockDAFootprint - gasTarget) // safe: we aim for > gasTarget
		if isLowScalar {
			delta = new(big.Int).SetUint64(header.GasUsed - gasTarget)
		}
		inc := new(big.Int).Mul(delta, parentBaseFee)
		inc.Div(inc, new(big.Int).SetUint64(gasTarget))
		inc.Div(inc, new(big.Int).SetUint64(*rollupCfg.ChainOpConfig.EIP1559DenominatorCanyon))
		if inc.Cmp(common.Big1) < 0 {
			inc = new(big.Int).Set(common.Big1)
		}
		expectedNextBaseFee := new(big.Int).Add(parentBaseFee, inc)

		// Build the next block and compare base fee
		env.Sequencer.ActL2EmptyBlock(t)
		next := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
		require.Equal(t, expectedNextBaseFee, next.BaseFee(), "next block base fee incorrect")
	}

	run := func(gt *testing.T, testCfg *helpers.TestCfg[helpers.DeployConfigOverride]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testCfg.Custom)

		jovianAtGenesis := env.Sequencer.RollupCfg.IsJovian(env.Sequencer.RollupCfg.Genesis.L2Time)
		if !jovianAtGenesis {
			env.Sequencer.ActBuildL2ToFork(t, rollup.Jovian)
		}

		// We run three sub-steps. First, we test the default behavior. Then we update the scalar to
		// test that the update works. Finally, we set the scalar to zero again to test that the
		// default is used again.
		// The first step will also test the activation block.
		runStep(t, env, nil)
		runStep(t, env, ptr(uint16(1000)))
		runStep(t, env, ptr(uint16(0)))
		// special case to test that a low value effectively disables the DA footprint block limit
		runStep(t, env, ptr(uint16(lowDAFootprintGasScalar)))

		// Run the FP program up to the current safe head
		env.BatchMineAndSync(t)
		l2SafeHead := env.Sequencer.L2Safe()
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	tests := map[string]struct {
		genesisConfigFn helpers.DeployConfigOverride
	}{
		"JovianAtGenesis": {
			genesisConfigFn: func(dc *genesis.DeployConfig) { dc.ActivateForkAtGenesis(rollup.Jovian) },
		},
		"JovianAfterGenesis": {
			genesisConfigFn: func(dc *genesis.DeployConfig) { dc.ActivateForkAtOffset(rollup.Jovian, 4) },
		},
	}

	for name, tt := range tests {
		gt.Run(name, func(t *testing.T) {
			matrix := helpers.NewMatrix[helpers.DeployConfigOverride]()
			matrix.AddDefaultTestCasesWithName(
				name,
				tt.genesisConfigFn,
				helpers.NewForkMatrix(helpers.Isthmus),
				run,
			)
			matrix.Run(t)
		})
	}
}
