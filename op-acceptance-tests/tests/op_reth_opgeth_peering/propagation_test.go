package op_reth_opgeth_peering

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestTransactionCrossesTheOpRethOpGethPeeringIntoABlock submits a transfer to one follower while
// the sequencer is paused and requires the other follower, the sender's only execution-layer
// peer, to hold it in its pool, and then both followers to see it in the same block once the
// sequencer resumes; from op-reth to op-geth and from op-geth to op-reth.
func TestTransactionCrossesTheOpRethOpGethPeeringIntoABlock(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, sender := range followerKinds {
		t.Run(directionName(sender), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newPairSystem(t, sender)
			alice := sys.FunderL2.NewFundedEOA(eth.OneEther).AsEL(sys.Sender.EL)
			// Both followers check the transfer against their own state, which must hold the
			// funding by then: the sender when it plans the transfer, the receiver when it admits
			// it to its pool.
			alice.WaitForBalanceAtLeast(eth.OneEther)
			alice.AsEL(sys.Receiver.EL).WaitForBalanceAtLeast(eth.OneEther)
			bob := sys.Wallet.NewEOA(sys.Sender.EL)

			sys.Sequencer.CL.StopSequencer()
			transfer := alice.Submit(alice.PlanTransfer(bob.Address(), eth.OneHundredthEther))
			sys.Receiver.EL.WaitForPooledTransaction(transfer.Hash())

			sys.Sequencer.CL.StartSequencer()
			onReceiver := sys.Receiver.EL.WaitForReceipt(transfer.Hash())
			onSender := sys.Sender.EL.WaitForReceipt(transfer.Hash())
			t.Require().Equal(onReceiver.BlockHash, onSender.BlockHash, "both followers must see the transfer in the same block")
		})
	}
}
