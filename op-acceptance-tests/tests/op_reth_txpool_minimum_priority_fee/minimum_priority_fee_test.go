package op_reth_txpool_minimum_priority_fee

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencerRejectsATransactionBelowTheMinimumPriorityFee submits a transfer whose tip is one
// wei under the sequencer's --txpool.minimum-priority-fee, through eth_sendRawTransaction and
// through eth_sendRawTransactionConditional, and requires the sequencer to refuse it with the
// pool's error and to know nothing of it afterwards.
func TestSequencerRejectsATransactionBelowTheMinimumPriorityFee(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "the sequencer under test is this tree's op-reth")
	sys := newSequencerSystem(t)
	for _, c := range sys.submissionMatrix(belowMinimum) {
		c.submit(sys, c.transfer).IsRejectedWith(eth.TransactionRejected,
			fmt.Sprintf("transaction priority fee below minimum required priority fee %d", minimumPriorityFeeWei))
		sys.Sequencer.VerifyTransactionUnknown(c.transfer.Hash())
	}
}

// TestSequencerIncludesATransactionAtTheMinimumPriorityFee submits a transfer whose tip is exactly
// the sequencer's --txpool.minimum-priority-fee, through eth_sendRawTransaction and through
// eth_sendRawTransactionConditional, and requires the sequencer to accept it and include it in a
// block.
func TestSequencerIncludesATransactionAtTheMinimumPriorityFee(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "the sequencer under test is this tree's op-reth")
	sys := newSequencerSystem(t)
	for _, c := range sys.submissionMatrix(atMinimum) {
		submission := c.submit(sys, c.transfer).IsAccepted()
		receipt := sys.Sequencer.WaitForReceipt(c.transfer.Hash())
		t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "the transfer submitted with %s must have succeeded", submission.Method)
	}
}
