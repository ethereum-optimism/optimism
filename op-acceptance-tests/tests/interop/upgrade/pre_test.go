package upgrade

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestPreNoInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()

	t.Logger().Info("Starting")

	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		interopTime := net.Escape().ChainConfig().InteropTime
		t.Require().NotNil(interopTime)
		pre := net.LatestBlockBeforeTimestamp(t, *interopTime)
		el := net.Escape().L2ELNode(match.FirstL2EL)
		codeAddr := common.HexToAddress("0xC0D3C0d3C0D3C0d3c0d3c0D3c0D3C0d3C0D30022")
		implCode, err := el.EthClient().CodeAtHash(t.Ctx(), codeAddr, pre.Hash)
		require.NoError(err)
		require.Len(implCode, 0, "needs to be empty")
		implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, pre.Hash.String())
		require.NoError(err)
		require.Equal(common.Address{}, common.BytesToAddress(implAddrBytes[:]))
	})

	// try access the sync-status of the supervisor, assert that the sync-status returns the expected error
	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		interopTime := net.Escape().ChainConfig().InteropTime

		_, err := sys.Supervisor.Escape().QueryAPI().SyncStatus(t.Ctx())
		require.ErrorContains(err, "supervisor status tracker not ready")

		// confirm we are still pre-interop
		require.False(net.IsActivated(*interopTime))
		t.Logger().Info("Timestamps", "interopTime", *interopTime, "now", time.Now().Unix())
	})

	var initReceipt *types.Receipt
	var initTx *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	// try interop before the upgrade, confirm that messages do not get included
	{
		// two EOAs for triggering the init and exec interop txs
		alice := sys.FunderA.NewFundedEOA(eth.OneEther)
		bob := sys.FunderB.NewFundedEOA(eth.OneEther)

		interopTimeA := sys.L2ChainA.Escape().ChainConfig().InteropTime
		interopTimeB := sys.L2ChainB.Escape().ChainConfig().InteropTime

		eventLoggerAddress := alice.DeployEventLogger()

		// wait for chain B to catch up to chain A if necessary
		sys.L2ChainB.CatchUpTo(sys.L2ChainA)

		// send initiating message on chain A
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		initTx, initReceipt = alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)))

		// at least one block between the init tx on chain A and the exec tx on chain B
		sys.L2ChainB.WaitForBlock()

		// send executing message on chain B and confirm we got an error
		execTx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
		execTx.Content.DependOn(&initTx.Result)
		execTx.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &initTx.Result, 0))
		execReceipt, err := execTx.PlannedTx.Included.Eval(sys.T.Ctx())
		require.ErrorContains(err, "implementation not initialized", "error did not contain expected string")
		require.Nil(execReceipt)

		t.Logger().Info("initReceipt", "blocknum", initReceipt.BlockNumber, "txhash", initReceipt.TxHash)

		// confirm we are still pre-interop
		require.False(sys.L2ChainA.IsActivated(*interopTimeA))
		require.False(sys.L2ChainB.IsActivated(*interopTimeB))
		t.Logger().Info("Timestamps", "interopTimeA", *interopTimeA, "interopTimeB", *interopTimeB, "now", time.Now().Unix())
	}

	// check that log events from a block before activation, when converted into an access-list, fail the check-access-list RPC check
	{
		ctx := sys.T.Ctx()

		execTrigger, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &initTx.Result, 0)(ctx)
		require.NoError(err)

		ed := stypes.ExecutingDescriptor{Timestamp: uint64(time.Now().Unix())}
		accessEntries := []stypes.Access{execTrigger.Msg.Access()}
		accessList := stypes.EncodeAccessList(accessEntries)

		err = sys.Supervisor.Escape().QueryAPI().CheckAccessList(ctx, accessList, stypes.CrossSafe, ed)
		require.ErrorContains(err, "conflicting data")
	}

	t.Logger().Info("Done")
}
