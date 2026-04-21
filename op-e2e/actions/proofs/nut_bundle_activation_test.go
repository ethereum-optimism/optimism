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

// TestActivationBlockNUTBundle verifies that, for every fork from Karst onward,
// the fork's activation block contains exactly the bundle's deposit transactions
// in order and that every upgrade tx executes successfully.
//
// Discovery runs through [forks.From]([forks.Karst]), so any future fork is
// covered automatically — and required to have a NUT bundle registered with
// [derive.UpgradeTransactions]. The only per-fork requirement beyond that is
// that the fork immediately preceding it is registered in [helpers.Hardforks]
// — a one-line entry needed for any fork-parametrized test in this package.
func TestActivationBlockNUTBundle(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()

	// Resolve Karst's index in forks.All so we can reach the immediately preceding
	// fork for each entry yielded by forks.From(Karst).
	karstIdx := -1
	for i, f := range forks.All {
		if f == forks.Karst {
			karstIdx = i
			break
		}
	}
	require.Greater(gt, karstIdx, 0, "Karst must not be first in forks.All")

	for i, fork := range forks.From(forks.Karst) {
		_, _, err := derive.UpgradeTransactions(fork)
		require.NoError(gt, err, "fork %s from Karst onward must have a NUT bundle", fork)

		preFork := forks.All[karstIdx+i-1]
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

	// Every tx in the activation block — the L1 info deposit and all NUT upgrade
	// deposits — must execute successfully. A reverted upgrade tx would leave the
	// chain in a broken fork-activation state.
	receipts := engine.L2Chain().GetReceiptsByHash(actHeader.Hash())
	require.Len(t, receipts, len(txs), "receipt count must match tx count")
	for i, r := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, r.Status,
			"activation-block tx %d reverted", i)
	}
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
