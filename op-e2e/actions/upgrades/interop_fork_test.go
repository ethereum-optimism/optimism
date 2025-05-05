package upgrades

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/upgrades/utils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestInteropUpgrade(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	// Run Isthmus until the interop upgrade
	is := dsl.SetupInterop(t, dsl.SetInteropOffsetForAllL2s(uint64(15)))
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)

	////////////////////////////
	// Pre-upgrade Block Production
	////////////////////////////

	// Start op-nodes
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	rollupConfigA := is.Out.L2s[actors.ChainA.ChainID.String()].RollupCfg
	rollupConfigB := is.Out.L2s[actors.ChainB.ChainID.String()].RollupCfg

	// Verify Interop is not active at genesis yet
	l2Head := actors.ChainA.Sequencer.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.True(t, rollupConfigA.IsIsthmus(l2Head.Time), "Isthmus should be active at genesis in chain A")
	require.False(t, rollupConfigA.IsInterop(l2Head.Time), "Interop should not be active at genesis in chain A")

	l2Head = actors.ChainB.Sequencer.L2Unsafe()
	require.NotZero(t, l2Head.Hash)
	require.True(t, rollupConfigB.IsIsthmus(l2Head.Time), "Isthmus should be active at genesis in chain B")
	require.False(t, rollupConfigB.IsInterop(l2Head.Time), "Interop should not be active at genesis in chain B")

	// Submit all and mine for both chains
	actors.ChainA.Sequencer.ActL2EmptyBlock(t)
	actors.ChainB.Sequencer.ActL2EmptyBlock(t)

	currentBlockHeaderA := actors.ChainA.SequencerEngine.L2Chain().CurrentBlock()
	currentBlockHeaderB := actors.ChainB.SequencerEngine.L2Chain().CurrentBlock()
	require.False(t, rollupConfigA.IsInterop(currentBlockHeaderA.Time),
		"Interop should not be active at the first L1 inclusion block in chain A")
	require.False(t, rollupConfigB.IsInterop(currentBlockHeaderB.Time),
		"Interop should not be active at the first L1 inclusion block in chain B")

	// Build a few L2 blocks. We only need the L1 inclusion to advance past Interop and Interop
	// shouldn't activate with L2 time.
	actors.ChainA.Sequencer.ActBuildL2ToInterop(t)
	actors.ChainB.Sequencer.ActBuildL2ToInterop(t)

	////////////////////////////
	// Post-upgrade Block Production
	////////////////////////////

	activationBlockHeaderA := actors.ChainA.SequencerEngine.L2Chain().CurrentBlock()
	activationBlockHeaderB := actors.ChainB.SequencerEngine.L2Chain().CurrentBlock()
	activationBlockIDA := eth.HeaderBlockID(activationBlockHeaderA)
	activationBlockIDB := eth.HeaderBlockID(activationBlockHeaderB)
	require.Equal(t, activationBlockHeaderA.Number.Uint64(), activationBlockHeaderB.Number.Uint64())

	require.True(t, rollupConfigA.IsInteropActivationBlock(activationBlockHeaderA.Time),
		"Interop should be active at the first L1 inclusion block in chain A")
	require.True(t, rollupConfigB.IsInteropActivationBlock(activationBlockHeaderB.Time),
		"Interop should be active at the first L1 inclusion block in chain B")

	activationBlock := actors.ChainA.SequencerEngine.L2Chain().GetBlockByHash(activationBlockHeaderA.Hash())
	activationBlockTxs := activationBlock.Transactions()
	VerifyInteropContractsDeployedCorrectly(t, actors.ChainA, activationBlockTxs, activationBlockIDA)
	VerifyInteropContractsDeployedCorrectly(t, actors.ChainB, activationBlockTxs, activationBlockIDB)

	actors.ActBatchAndMine(t, actors.ChainA, actors.ChainB)

	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	// The node will exhaust L1 data,
	// it needs the supervisor to see the L1 block first,
	// and provide it to the node.
	for i := 0; i < 2; i++ {
		actors.ChainA.Sequencer.ActL2EventsUntil(t, event.Is[derive.ExhaustedL1Event], 100, false)
		actors.ChainB.Sequencer.ActL2EventsUntil(t, event.Is[derive.ExhaustedL1Event], 100, false)
		actors.Supervisor.SignalLatestL1(t)          // supervisor will be aware of latest L1
		actors.ChainA.Sequencer.SyncSupervisor(t)    // supervisor to react to exhaust-L1
		actors.ChainB.Sequencer.SyncSupervisor(t)    // supervisor to react to exhaust-L1
		actors.ChainA.Sequencer.ActL2PipelineFull(t) // node to complete syncing to L1 head.
		actors.ChainB.Sequencer.ActL2PipelineFull(t) // node to complete syncing to L1 head.
	}

	actors.ChainA.Sequencer.ActL1HeadSignal(t)
	actors.ChainB.Sequencer.ActL1HeadSignal(t)

	actors.ChainA.Sequencer.SyncSupervisor(t) // supervisor to react to exhaust-L1
	actors.ChainB.Sequencer.SyncSupervisor(t) // supervisor to react to exhaust-L1

	actors.Supervisor.ProcessFull(t)

	actors.ChainA.Sequencer.ActL2PipelineFull(t) // node to complete syncing to L1 head.
	actors.ChainB.Sequencer.ActL2PipelineFull(t) // node to complete syncing to L1 head.

	// Verify the sync status is correct
	statusA := actors.ChainA.Sequencer.SyncStatus()
	require.Equal(t, activationBlockIDA, statusA.UnsafeL2.ID())
	require.Equal(t, activationBlockIDA, statusA.CrossUnsafeL2.ID())
	require.Equal(t, activationBlockIDA, statusA.LocalSafeL2.ID())
	require.Equal(t, activationBlockIDA, statusA.SafeL2.ID())
	require.Equal(t, uint64(0), statusA.FinalizedL2.Number)

	statusB := actors.ChainB.Sequencer.SyncStatus()
	require.Equal(t, activationBlockIDB, statusB.UnsafeL2.ID())
	require.Equal(t, activationBlockIDB, statusB.CrossUnsafeL2.ID())
	require.Equal(t, activationBlockIDB, statusB.LocalSafeL2.ID())
	require.Equal(t, activationBlockIDB, statusB.SafeL2.ID())
	require.Equal(t, uint64(0), statusB.FinalizedL2.Number)
}

func VerifyInteropContractsDeployedCorrectly(t helpers.Testing, chain *dsl.Chain, activationBlockTxs []*types.Transaction, activationBlockID eth.BlockID) {
	require.Len(t, activationBlockTxs, 5) // 4 upgrade txs + 1 system deposit tx
	upgradeTransactions := activationBlockTxs[1:]
	upgradeTransactionBytes := make([]hexutil.Bytes, len(upgradeTransactions))
	for i, tx := range upgradeTransactions {
		txBytes, err := tx.MarshalBinary()
		require.NoError(t, err)
		upgradeTransactionBytes[i] = txBytes
	}

	expectedUpgradeTransactions, err := derive.InteropNetworkUpgradeTransactions()
	require.NoError(t, err)

	require.Equal(t, upgradeTransactionBytes, expectedUpgradeTransactions)

	utils.RequireContractDeployedAndProxyUpdated(t, chain, derive.CrossL2InboxAddress, predeploys.CrossL2InboxAddr, activationBlockID)
	utils.RequireContractDeployedAndProxyUpdated(t, chain, derive.L2ToL2MessengerAddress, predeploys.L2toL2CrossDomainMessengerAddr, activationBlockID)
}
