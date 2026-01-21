package sdm

import (
	"math/rand"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestOutOfConsensusGasReduction(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithTestSeq(t)
	l := t.Logger()

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
	l.Info("Receipt", "gas used", ri.GasUsed)
	l.Info("Receipt", "gas used", ri.GasUsed)
	l.Info("Receipt", "tx index", ri.TransactionIndex)

	sys.L2EL.Advanced(eth.Unsafe, 2)
	sys.L2ELB.Matched(sys.L2EL, types.LocalUnsafe, 5)

	status := sys.L2CL.SyncStatus()
	spew.Dump(status)
}
