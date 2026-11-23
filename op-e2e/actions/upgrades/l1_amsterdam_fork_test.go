package upgrades

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
)

// TestDeriveAcrossL1Amsterdam submits blob batches on both sides of the L1 Amsterdam activation,
// and a deposit on an Amsterdam L1 block, and checks that a verifier derives all of them.
func TestDeriveAcrossL1Amsterdam(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	amsterdamOffset := hexutil.Uint64(36)
	env := helpers.SetupEnv(t, helpers.EnvOpt{DeployConfigMod: func(dc *genesis.DeployConfig) {
		dc.L1AmsterdamTimeOffset = &amsterdamOffset
	}})
	env.Batcher = helpers.NewL2Batcher(env.Log, env.SetupData.RollupCfg, &helpers.BatcherCfg{
		MaxL1TxSize:          128_000,
		BatcherKey:           env.DeployParams.Secrets.Batcher,
		DataAvailabilityType: batcherFlags.BlobsType,
	}, env.Seq.RollupClient(), env.Miner.EthClient(), env.SeqEngine.EngineClient(t, env.SetupData.RollupCfg))
	l1Cfg := env.SetupData.L1Cfg.Config

	sawPreAmsterdamBatch, sawAmsterdamBatch := false, false
	for i := 0; i < 6; i++ {
		env.Miner.ActEmptyBlock(t)
		env.Seq.ActL1HeadSignal(t)
		env.Seq.ActBuildToL1Head(t)

		inclusion := env.ActBatchSubmitAllAndMine(t)
		require.Equal(t, uint8(types.BlobTxType), env.Batcher.LastSubmitted.Type())
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
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
	env.Miner.ActL1EndBlock(t)
	env.Seq.ActL1HeadSignal(t)
	env.Seq.ActBuildToL1Head(t)
	env.Alice.ActCheckDepositStatus(true, true)(t)

	env.ActBatchSubmitAllAndMine(t)
	env.Verifier.ActL1HeadSignal(t)
	env.Verifier.ActL2PipelineFull(t)
	require.Equal(t, env.Seq.L2Unsafe(), env.Verifier.L2Safe())
}
