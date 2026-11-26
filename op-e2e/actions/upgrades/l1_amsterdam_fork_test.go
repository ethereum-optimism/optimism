package upgrades

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestDeriveAcrossL1Amsterdam submits batches on both sides of the L1 Amsterdam activation, and a
// deposit on an Amsterdam L1 block, and checks that a verifier derives all of them.
func TestDeriveAcrossL1Amsterdam(gt *testing.T) {
	gt.Run("Calldata", func(gt *testing.T) {
		testDeriveAcrossL1Amsterdam(gt, batcherFlags.CalldataType, types.DynamicFeeTxType)
	})
	gt.Run("Blobs", func(gt *testing.T) {
		testDeriveAcrossL1Amsterdam(gt, batcherFlags.BlobsType, types.BlobTxType)
	})
}

func testDeriveAcrossL1Amsterdam(gt *testing.T, daType batcherFlags.DataAvailabilityType, batchTxType uint8) {
	t := helpers.NewDefaultTesting(gt)
	amsterdamOffset := hexutil.Uint64(36)
	env := helpers.SetupEnv(t, helpers.EnvOpt{DeployConfigMod: func(dc *genesis.DeployConfig) {
		dc.L1AmsterdamTimeOffset = &amsterdamOffset
	}})
	batcherCfg := helpers.DefaultBatcherCfg(env.DeployParams)
	batcherCfg.DataAvailabilityType = daType
	env.Batcher = helpers.NewL2Batcher(env.Log, env.SetupData.RollupCfg, batcherCfg,
		env.Seq.RollupClient(), env.Miner.EthClient(), env.SeqEngine.EngineClient(t, env.SetupData.RollupCfg))
	l1Cfg := env.SetupData.L1Cfg.Config

	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&env.DeployParams.Addresses.Bob)(t)
	rng := rand.New(rand.NewSource(1))
	sawPreAmsterdamBatch, sawAmsterdamBatch := false, false
	for i := 0; i < 6; i++ {
		env.Miner.ActEmptyBlock(t)
		env.Seq.ActL1HeadSignal(t)
		// Incompressible L2 calldata makes the batch large enough for the Amsterdam calldata floor
		// (EIP-7976) to exceed the pre-Amsterdam one.
		env.Alice.L2.ActSetTxCalldata(testutils.RandomData(rng, 1_000))(t)
		env.Alice.L2.ActMakeTx(t)
		env.Seq.ActL2StartBlock(t)
		env.SeqEngine.ActL2IncludeTx(env.Alice.Address())(t)
		env.Seq.ActL2EndBlock(t)
		env.Seq.ActBuildToL1Head(t)

		inclusion := env.ActBatchSubmitAllAndMine(t)
		require.Equal(t, batchTxType, env.Batcher.LastSubmitted.Type())
		if l1Cfg.IsAmsterdam(inclusion.Number(), inclusion.Time()) {
			require.NotNil(t, inclusion.Header().BlockAccessListHash)
			sawAmsterdamBatch = true
		} else {
			sawPreAmsterdamBatch = true
		}

		env.Verifier.ActL1HeadSignal(t)
		env.Verifier.ActL2PipelineFull(t)
		require.Equal(t, env.Seq.L2Unsafe(), env.Verifier.L2Safe(), "verifier derives the batch in L1 block %d", inclusion.NumberU64())
	}
	require.True(t, sawPreAmsterdamBatch, "a batch lands before Amsterdam")
	require.True(t, sawAmsterdamBatch, "a batch lands after Amsterdam")

	// A deposit on an Amsterdam L1 block is derived.
	env.Alice.L1.ActResetTxOpts(t)
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&env.DeployParams.Addresses.Bob)(t)
	env.Alice.ActDeposit(t)
	env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
	depositBlock := env.Miner.ActL1EndBlock(t)
	require.True(t, l1Cfg.IsAmsterdam(depositBlock.Number(), depositBlock.Time()), "the deposit lands on an Amsterdam L1 block")
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActBuildToL1Head(t)
	env.Alice.ActCheckDepositStatus(true, true)(t)

	env.ActBatchSubmitAllAndMine(t)
	env.Verifier.ActL1HeadSignal(t)
	env.Verifier.ActL2PipelineFull(t)
	require.Equal(t, env.Seq.L2Unsafe(), env.Verifier.L2Safe())
}
