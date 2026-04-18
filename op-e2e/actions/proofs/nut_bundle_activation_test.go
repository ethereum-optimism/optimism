package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestActivationBlockNUTBundle verifies that, for every fork that embeds a NUT
// bundle, the fork's activation block contains exactly the bundle's deposit
// transactions in order.
//
// The test is generic across forks: discovery runs through [forks.All] and
// [derive.UpgradeTransactions], so any future fork that registers a bundle is
// picked up automatically. The only per-fork requirement is that the fork
// immediately preceding it is already registered in [helpers.Hardforks] — a
// one-line entry that is needed for any fork-parametrized test in this package.
func TestActivationBlockNUTBundle(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()

	for i, fork := range forks.All {
		if _, _, err := derive.UpgradeTransactions(fork); err != nil {
			continue
		}
		require.Greater(gt, i, 0, "fork %s has a NUT bundle but is first in forks.All; no pre-fork available", fork)
		preFork := forks.All[i-1]
		preHelper := lookupHardforkHelper(preFork)
		require.NotNil(gt, preHelper,
			"no pre-fork helper registered for NUT-bundle fork %s (prior fork: %s); add %s to helpers.Hardforks",
			fork, preFork, preFork)

		matrix.AddDefaultTestCasesWithName(
			string(fork),
			fork,
			helpers.NewForkMatrix(preHelper),
			testActivationBlockNUTBundle,
		)
	}

	matrix.Run(gt)
}

func testActivationBlockNUTBundle(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	fork := testCfg.Custom
	t := actionsHelpers.NewDefaultTesting(gt)

	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(fork, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	expectedTxs, expectedGas, err := derive.UpgradeTransactions(fork)
	require.NoError(t, err, "load NUT bundle for %s", fork)
	require.NotEmpty(t, expectedTxs, "bundle for %s must contain at least one upgrade tx", fork)

	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset); i++ {
		env.Sequencer.ActL2EmptyBlock(t)
	}

	engine := env.Engine
	actHeader := engine.L2Chain().CurrentHeader()
	blockTime := env.Sd.RollupCfg.BlockTime
	require.Equal(t, fork,
		env.Sd.RollupCfg.IsActivationBlock(actHeader.Time-blockTime, actHeader.Time),
		"expected activation block for %s at time %d", fork, actHeader.Time)

	actBlock := engine.L2Chain().GetBlockByHash(actHeader.Hash())
	txs := actBlock.Transactions()
	// Index 0 is the L1 info deposit; indices 1.. are the NUT upgrade deposits.
	require.Len(t, txs, 1+len(expectedTxs),
		"activation block should have 1 L1 info deposit + %d NUT upgrade txs", len(expectedTxs))

	var totalUpgradeGas uint64
	for i, rawExpected := range expectedTxs {
		actualBytes, err := txs[1+i].MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, []byte(rawExpected), actualBytes, "NUT tx %d byte mismatch", i)

		var expected types.Transaction
		require.NoError(t, expected.UnmarshalBinary(rawExpected))
		totalUpgradeGas += expected.Gas()
	}
	require.Equal(t, expectedGas, totalUpgradeGas, "total NUT gas must equal bundle total")
}

// lookupHardforkHelper resolves a fork name to its [helpers.Hardfork] entry by
// scanning [helpers.Hardforks]. Returns nil when the fork isn't registered.
func lookupHardforkHelper(name forks.Name) *helpers.Hardfork {
	for _, hf := range helpers.Hardforks {
		if forks.Name(hf.Name) == name {
			return hf
		}
	}
	return nil
}
