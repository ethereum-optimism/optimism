package op_reth_opgeth_peering

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	// crossingAttempts bounds, in 1s polls, how long a transfer may take to cross the peering
	// into the receiver's pool.
	crossingAttempts = 30
	// inclusionAttempts bounds, in 1s polls, how long the resumed sequencer may take to include
	// the transfer and each follower to import the block that does.
	inclusionAttempts = 30
)

// TestTransactionCrossesTheOpRethOpGethPeeringIntoABlock submits a transfer to one follower while
// the sequencer is paused and requires the other follower, the sender's only execution-layer
// peer, to hold it in its pool, and then both followers to see it in the same block once the
// sequencer resumes; from op-reth to op-geth and from op-geth to op-reth.
func TestTransactionCrossesTheOpRethOpGethPeeringIntoABlock(gt *testing.T) {
	t := devtest.ParallelT(gt)
	devtest.RunParallel(t, followerCells("%s_to_%s"), func(t devtest.T, sender followerCell) {
		skipUnsupported(t)
		sys := newPairSystem(t, sender.Kind)
		alice := sys.FunderL2.NewFundedEOA(eth.OneEther).AsEL(sys.Sender.EL)
		// Both followers check the transfer against their own state, which must hold the
		// funding by then: the sender when it plans the transfer, the receiver when it admits
		// it to its pool.
		alice.WaitForBalanceAtLeast(eth.OneEther)
		alice.AsEL(sys.Receiver.EL).WaitForBalanceAtLeast(eth.OneEther)
		bob := sys.Wallet.NewEOA(sys.Sender.EL)

		pausedHead := sys.Sequencer.EL.BlockRefByHash(sys.Sequencer.CL.StopSequencer())
		transfer := alice.Submit(alice.PlanTransfer(bob.Address(), eth.OneHundredthEther))
		sys.Receiver.EL.WaitForPooledTransaction(transfer.Hash(), crossingAttempts)

		sys.Sequencer.CL.StartSequencer()
		transfers := []*types.Transaction{transfer}
		sys.Receiver.EL.WaitForIncludedTransactions(transfers, pausedHead.Number+1, len(transfers), inclusionAttempts)
		sys.Sender.EL.WaitForIncludedTransactions(transfers, pausedHead.Number+1, len(transfers), inclusionAttempts)
		onReceiver := sys.Receiver.EL.WaitForReceipt(transfer.Hash())
		onSender := sys.Sender.EL.WaitForReceipt(transfer.Hash())
		t.Require().Equal(onReceiver.BlockHash, onSender.BlockHash, "both followers must see the transfer in the same block")
	})
}
