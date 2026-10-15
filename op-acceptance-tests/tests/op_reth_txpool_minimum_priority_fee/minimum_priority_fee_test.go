package op_reth_txpool_minimum_priority_fee

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencerRejectsATransactionBelowTheMinimumPriorityFee submits a transfer whose tip is one
// wei under the sequencer's --txpool.minimum-priority-fee, through eth_sendRawTransaction and
// through eth_sendRawTransactionConditional, and requires the sequencer to refuse it with the
// pool's error and to know nothing of it afterwards.
func TestSequencerRejectsATransactionBelowTheMinimumPriorityFee(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range submissionMatrix() {
		t.Run(c.name, func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			if c.localOrigin {
				t.Skip("accepted on develop (#23400): reth 2.5.2 exempts a transaction submitted over the node's own RPC from " +
					"--txpool.minimum-priority-fee, so the sequencer pools and includes it; reth 2.6 holds every origin to the " +
					"minimum (reth#26861) and refuses it with `-32003 transaction priority fee below minimum required priority fee 1000000000`")
			}
			sys := newSequencerSystem(t)
			transfer := sys.fundAndSignTransfer(belowMinimum)

			c.submit(sys, transfer).IsRejectedWith(eth.TransactionRejected,
				fmt.Sprintf("transaction priority fee below minimum required priority fee %d", minimumPriorityFeeWei))
			sys.Sequencer.VerifyTransactionUnknown(transfer.Hash())
		})
	}
}

// TestSequencerIncludesATransactionAtTheMinimumPriorityFee submits a transfer whose tip is exactly
// the sequencer's --txpool.minimum-priority-fee, through eth_sendRawTransaction and through
// eth_sendRawTransactionConditional, and requires the sequencer to accept it and include it in a
// block.
func TestSequencerIncludesATransactionAtTheMinimumPriorityFee(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range submissionMatrix() {
		t.Run(c.name, func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newSequencerSystem(t)
			transfer := sys.fundAndSignTransfer(atMinimum)

			c.submit(sys, transfer).IsAccepted()
			receipt := sys.Sequencer.WaitForReceipt(transfer.Hash())
			t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "the transfer must have succeeded")
		})
	}
}
