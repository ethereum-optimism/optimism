package sdm

import (
	"math/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestOutOfConsensusGasReduction(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)
	l := t.Logger()

	faucetPk := "0x6a0dbabf7507bcd11c21aea16ee75051d6fc1f40d0f8e5057e19f78da0e49cae"

	eoaFromFaucetPkL2 := dsl.NewKeyFromHex(t, faucetPk).User(sys.L2EL)
	l.Info("L2 Faucet EOA", "address", eoaFromFaucetPkL2.Address(), "balance", eoaFromFaucetPkL2.GetBalance())

	eoaFromFaucetPkL1 := dsl.NewKeyFromHex(t, faucetPk).User(sys.L1EL)
	l.Info("L1 Faucet EOA", "address", eoaFromFaucetPkL1.Address(), "balance", eoaFromFaucetPkL1.GetBalance())

	// Send funds from L1 to L2 via OptimismPortal (eoaFromFaucetPkL1 has funds on L1)
	depositAmount := eth.TenEther
	initialL2Balance := eoaFromFaucetPkL2.GetBalance()
	portalAddr := sys.L2Chain.DepositContractAddr()
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t),
	)
	depositCall := portal.DepositTransaction(eoaFromFaucetPkL2.Address(), depositAmount, 300_000, false, []byte{})
	depositReceipt := contract.Write(eoaFromFaucetPkL1, depositCall, txplan.WithValue(depositAmount))
	t.Require().Eventually(func() bool {
		head := sys.L2CL.HeadBlockRef(types.LocalUnsafe)
		return head.L1Origin.Number >= bigs.Uint64Strict(depositReceipt.BlockNumber)
	}, sys.L1EL.TransactionTimeout(), time.Second, "awaiting deposit to be processed by L2")
	eoaFromFaucetPkL2.WaitForBalance(initialL2Balance.Add(depositAmount))
	l.Info("L1→L2 deposit completed", "amount", depositAmount, "L2 balance", eoaFromFaucetPkL2.GetBalance())

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	cathrine := sys.FunderL2.NewFundedEOA(eth.ZeroWei)

	l.Info("Alice", "address", alice.Address())
	l.Info("Catherine", "address", cathrine.Address())

	// EventLogger is used to generate a random tx with ~100k gas, parts of which we want to refund with the sequencer
	{
		rng := rand.New(rand.NewSource(1234))

		eventLoggerAddress := alice.DeployEventLogger()

		// Intent to initiate message with multiple messages, all included in single tx
		eventCnt := 10
		initCalls := make([]txintent.Call, eventCnt)
		for index := range eventCnt {
			initCalls[index] = interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(100))
		}

		txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
		txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

		receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
		require.NoError(t, err)
		l.Info("EventLogger message included", "block", receiptA.BlockHash, "gas_used", receiptA.GasUsed)
		require.Equal(t, eventCnt, len(receiptA.Logs))
	}

	amount := eth.OneHundredthEther
	receipt := alice.Transfer(cathrine.Address(), amount)

	ri := receipt.Included.Value()
	l.Info("Receipt", "gas_used", ri.GasUsed)
	l.Info("Receipt", "tx index", ri.TransactionIndex)

	sys.L2EL.Advanced(eth.Unsafe, 21)
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	l.Info("Unsafe advanced, waiting for Safe to advance")

	l.Info("Chain status after Unsafe sync")
	status := sys.L2CL.SyncStatus()
	spew.Dump(status)

	status = sys.L2CLB.SyncStatus()
	spew.Dump(status)

	ref := status.UnsafeL2

	// l.Info("Starting Batcher")

	// sys.L2Batcher.Start()

	sys.L2EL.Advanced(eth.Safe, 2)
	sys.L2ELB.Advanced(eth.Safe, 2)

	l.Info("Safe advanced, waiting for Safe to match between the nodes")

	sys.L2ELB.Matched(sys.L2EL, types.LocalSafe, 5)

	l.Info("Chain status after Safe sync")
	status = sys.L2CL.SyncStatus()
	spew.Dump(status)

	status = sys.L2CLB.SyncStatus()
	spew.Dump(status)

	// confirm that the safe safety passed previously known unsafe and didn't reorg
	sys.L2CLB.ReachedRef(types.LocalSafe, eth.BlockID{
		Number: ref.Number,
		Hash:   ref.Hash,
	}, 5)
}
