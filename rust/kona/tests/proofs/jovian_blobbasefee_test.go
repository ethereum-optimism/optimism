package proofs

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestJovianBlobBaseFeeMatchesCanonical checks BLOBBASEFEE through the kona fault-proof program
// after Jovian. The opcode must always be 1 on the OP Stack, but post-Jovian the header's
// blobGasUsed carries the block's DA footprint. The footprint only affects the opcode result, so
// the test records BLOBBASEFEE from a contract in a block whose parent carries a large DA
// footprint, then proves the chain from genesis.
func TestJovianBlobBaseFeeMatchesCanonical(gt *testing.T) {
	// Runtime bytecode: BLOBBASEFEE PUSH0 SSTORE STOP. On any call it stores BLOBBASEFEE into slot 0.
	blobBaseFeeProbeAddr := common.HexToAddress("0x00000000000000000000000000000000c0ffee01")
	blobBaseFeeProbeCode := common.FromHex("0x4a5f5500")

	run := func(gt *testing.T, testCfg *helpers.TestCfg[helpers.DeployConfigOverride]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Pre-seed the BLOBBASEFEE probe contract into the L2 genesis allocation. Copy DefaultAlloc
		// rather than sharing the pointer, since the HonestClaim/JunkClaim variants run in parallel.
		allocsCopy := *actionsHelpers.DefaultAlloc
		testCfg.Allocs = &allocsCopy
		testCfg.Allocs.L2Alloc = map[common.Address]types.Account{
			blobBaseFeeProbeAddr: {
				Code:    blobBaseFeeProbeCode,
				Nonce:   1,
				Balance: new(big.Int),
			},
		}

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testCfg.Custom)
		require.True(t, env.Sequencer.RollupCfg.IsJovian(env.Sequencer.RollupCfg.Genesis.L2Time),
			"Jovian must be active at genesis for the header blobGasUsed field to carry the DA footprint")

		rollupCfg := env.Sequencer.RollupCfg
		gasLimit := rollupCfg.Genesis.SystemConfig.GasLimit
		effectiveScalar := uint64(derive.DAFootprintGasScalarDefault)

		l2Cl := env.Engine.EthClient()
		signer := types.LatestSigner(env.Sd.L2Cfg.Config)
		nonce, err := l2Cl.PendingNonceAt(t.Ctx(), env.Dp.Addresses.Alice)
		require.NoError(t, err)

		// Build a block whose DA footprint is large enough that a parent-derived blob base fee would
		// exceed 1. Deterministic, incompressible calldata keeps the footprint stable across runs.
		const dataSize = 5_000
		payload := make([]byte, dataSize)
		_, err = rand.New(rand.NewSource(42)).Read(payload)
		require.NoError(t, err)

		const targetDAFootprint = 10_000_000
		var runningDAFootprint uint64

		env.Sequencer.ActL2StartBlock(t)
		for runningDAFootprint < targetDAFootprint {
			tx := types.MustSignNewTx(env.Dp.Secrets.Alice, signer, &types.DynamicFeeTx{
				ChainID:   rollupCfg.L2ChainID,
				Nonce:     nonce,
				To:        &common.Address{},
				Gas:       params.TxGas + 40*dataSize, // cover floor calldata gas
				GasFeeCap: big.NewInt(5_000_000_000),
				GasTipCap: big.NewInt(2),
				Value:     big.NewInt(0),
				Data:      payload,
			})
			require.NoError(t, l2Cl.SendTransaction(t.Ctx(), tx))
			_, err = env.Engine.EngineApi.IncludeTx(tx, env.Dp.Addresses.Alice)
			require.NoError(t, err)
			runningDAFootprint += bigs.Uint64Strict(tx.RollupCostData().EstimatedDASize()) * effectiveScalar
			nonce++
		}
		env.Sequencer.ActL2EndBlock(t)

		footprintHeader := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash).Header()
		require.NotNil(t, footprintHeader.BlobGasUsed, "blobGasUsed must be set on Jovian blocks")
		require.Less(t, *footprintHeader.BlobGasUsed, gasLimit, "DA footprint must stay below the block gas limit")
		require.Greater(t, *footprintHeader.BlobGasUsed, uint64(4_258_349),
			"DA footprint must exceed the threshold at which a parent-derived blob base fee exceeds 1")

		// In the next block, call the probe so it records BLOBBASEFEE. This block's blob env depends on
		// its parent, the large-footprint block.
		env.Sequencer.ActL2StartBlock(t)
		env.Bob.L2.ActResetTxOpts(t)
		env.Bob.L2.ActSetTxToAddr(&blobBaseFeeProbeAddr)(t)
		env.Bob.L2.ActSetTxGasLimit(200_000)(t)
		env.Bob.L2.ActMakeTx(t)
		env.Engine.ActL2IncludeTx(env.Bob.Address())(t)
		env.Sequencer.ActL2EndBlock(t)
		env.Bob.L2.ActCheckReceiptStatusOfLastTx(true)(t)

		probeBlockNum := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash).Number()
		storedBlobBaseFee, err := l2Cl.StorageAt(t.Ctx(), blobBaseFeeProbeAddr, common.Hash{}, probeBlockNum)
		require.NoError(t, err)
		require.EqualValues(t, 1, bigs.Uint64Strict(new(big.Int).SetBytes(storedBlobBaseFee)),
			"canonical BLOBBASEFEE on the OP Stack must always be 1")

		// Prove the chain from genesis: the recorded BLOBBASEFEE must match the canonical value through
		// re-execution.
		env.BatchMineAndSync(t)
		l2SafeHead := env.Sequencer.L2Safe()
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[helpers.DeployConfigOverride]()
	matrix.AddDefaultTestCases(
		func(dc *genesis.DeployConfig) { dc.ActivateForkAtGenesis(forks.Jovian) },
		helpers.NewForkMatrix(helpers.Isthmus),
		run,
	)
	matrix.Run(gt)
}
