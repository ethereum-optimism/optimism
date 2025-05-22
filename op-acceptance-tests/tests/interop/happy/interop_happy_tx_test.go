package happy

import (
	"math/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// Other setups may be added here, hydrated from the same orchestrator
	presets.DoMain(m, presets.WithSimpleInterop())
}

// TestInteropHappyTx is testing that a valid init message, followed by a valid exec message are correctly
// included in two L2 chains and that the cross-safe ref for both of them progresses as expected beyond
// the block number where the messages were included
func TestInteropHappyTx(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSimpleInterop(t)
	l := sys.Log

	// two EOAs for triggering the init and exec interop txs
	var alice, bob *dsl.EOA
	{
		// alice is on chain A
		pk, err := crypto.GenerateKey()
		require.NoError(t, err)
		alice = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELA)
		sys.FaucetA.Fund(alice.Address(), eth.OneEther)

		// bob is on chain B
		pk, err = crypto.GenerateKey()
		require.NoError(t, err)
		bob = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELB)
		sys.FaucetB.Fund(bob.Address(), eth.OneEther)

		l.Info("alice", "address", alice.Address())
		l.Info("bob", "address", bob.Address())
	}

	sys.L1Network.WaitForBlock()
	sys.L2ChainA.WaitForBlock()

	// deploy event logger on chain A
	var eventLoggerAddress common.Address
	{
		tx := txplan.NewPlannedTx(txplan.Combine(
			alice.Plan(),
			txplan.WithData(common.FromHex(bindings.EventloggerBin)),
		))
		res, err := tx.Included.Eval(ctx)
		require.NoError(t, err)

		eventLoggerAddress = res.ContractAddress
		l.Info("deployed EventLogger", "chainID", tx.ChainID.Value(), "address", eventLoggerAddress)
	}

	sys.L1Network.WaitForBlock()

	var initTrigger *txintent.InitTrigger
	// prepare init trigger (i.e. what logs to emit on chain A)
	{
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		nTopics := 3
		lenData := 10
		initTrigger = interop.RandomInitTrigger(rng, eventLoggerAddress, nTopics, lenData)

		l.Info("created init trigger", "address", eventLoggerAddress, "topics", nTopics, "lenData", lenData)
	}

	// wait for chain B to catch up to chain A if necessary
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	var initTx *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	var initReceipt *types.Receipt
	// prepare and include initiating message on chain A
	{
		initTx = txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](alice.Plan())
		initTx.Content.Set(initTrigger)
		var err error
		initReceipt, err = initTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)

		l.Info("initiating message included", "chain", sys.L2ChainA.ChainID(), "block_number", initReceipt.BlockNumber, "block_hash", initReceipt.BlockHash, "now", time.Now().Unix())
	}

	// at least one block between the init tx on chain A and the exec tx on chain B
	sys.L2ChainB.WaitForBlock()

	var execTx *txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput]
	var execReceipt *types.Receipt
	// prepare and include executing message on chain B
	{
		execTx = txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
		execTx.Content.DependOn(&initTx.Result)
		// single event in tx so index is 0. ExecuteIndexed returns a lambda to transform InteropOutput to a new ExecTrigger
		execTx.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &initTx.Result, 0))
		var err error
		execReceipt, err = execTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, len(execReceipt.Logs))

		l.Info("executing message included", "chain", sys.L2ChainB.ChainID(), "block_number", execReceipt.BlockNumber, "block_hash", execReceipt.BlockHash, "now", time.Now().Unix())
	}

	// confirm that the cross-safe minimum block has been reached
	dsl.CheckAll(t,
		sys.L2CLA.ReachedRef(stypes.CrossSafe, eth.BlockID{
			Number: initReceipt.BlockNumber.Uint64(),
			Hash:   initReceipt.BlockHash,
		}, 30),
		sys.L2CLB.ReachedRef(stypes.CrossSafe, eth.BlockID{
			Number: execReceipt.BlockNumber.Uint64(),
			Hash:   execReceipt.BlockHash,
		}, 30),
	)

	sys.L2ChainA.PrintChain()
	sys.L2ChainB.PrintChain()
	spew.Dump(sys.Supervisor.FetchSyncStatus())
}
