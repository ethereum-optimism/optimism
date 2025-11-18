package flashblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/stretchr/testify/require"
)

type timedMessage struct {
	message   []byte
	timestamp time.Time
}

// TestFlashblocksTransfer checks that a transfer gets reflected in a flashblock before the transaction is confirmed in a block
// This test concurrently:
// - listens to the Flashblocks stream for 20s and collects all those streamed Flashblocks along with the timestamp when they were received.
// - Makes a transaction from Alice to Bob with a pre-known amount and records an approximated txn confirmation time (with upto nanosecond granularity) just after that txn was done.

// Expectations:
// - After flashblock streaming is done (20s), the transaction's already included in a (real) block.
// - There must have been a Flashblock containing a new_account_balance corresponding to Bob's account. This flashblock would be representative of the flashblock including Alice-to-Bob transaction.
// - That Flashblock's time (in seconds) must be less than or equal to the Transaction's block time (in seconds). (Can't check the block time beyond the granularity of seconds)
// - That Flashblock's time in nanoseconds must be before the approximated transaction confirmation time recorded previously.
func TestFlashblocksTransfer(gt *testing.T) {
	t := devtest.SerialT(gt)
	logger := t.Logger()
	tracer := t.Tracer()
	ctx := t.Ctx()
	sys := presets.NewSingleChainWithFlashblocks(t)

	topLevelCtx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	ctx, cancel := context.WithTimeout(topLevelCtx, 45*time.Second)
	defer cancel()
	_, span = tracer.Start(ctx, fmt.Sprintf("test chain %s", sys.L2Chain.String()))
	defer span.End()

	// Drive a couple blocks on the test sequencer so the faucet L2 funding tx has a chance to land before we rely on it.
	driveViaTestSequencer(t, sys, 2)

	alice := sys.FunderL2.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)
	bobAddress := bob.Address().Hex()

	// flashblocks listener - start goroutine and wait for it to be running
	flashblocksClient := sys.L2RollupBoost.FlashblocksClient()
	output := make(chan []byte, 100)
	doneListening := make(chan struct{})
	go func() {
		err := flashblocksClient.ReadAll(ctx, logger.With("stream_source", "rollup-boost"), 20*time.Second, output, doneListening)
		if err != nil {
			t.Require().NoError(err, "failed to listen for flashblocks")
		}
	}()

	var executedTransaction *txplan.PlannedTx
	var transactionApproxConfirmationTime time.Time
	var expectedBobBalance string

	// transactor
	go func() {
		bobBalance := bob.GetBalance()

		depositAmount := eth.OneHundredthEther
		bobAddr := bob.Address()
		executedTransaction = alice.Transact(
			alice.Plan(),
			txplan.WithTo(&bobAddr),
			txplan.WithValue(depositAmount),
		)
		transactionApproxConfirmationTime = time.Now()
		newBobBalance := bobBalance.Add(depositAmount)
		expectedBobBalance = newBobBalance.Hex()
		bob.VerifyBalanceExact(newBobBalance)
	}()

	streamedMessages := make([]timedMessage, 0)
	listening := true
	for listening {
		select {
		case <-doneListening:
			listening = false
		case msg := <-output:
			streamedMessages = append(streamedMessages, timedMessage{message: msg, timestamp: time.Now()})
		}
	}
	require.Greater(t, len(streamedMessages), 0, "should have received at least one message from the flashblocks stream")
	require.NotNil(t, executedTransaction, "should have executed a transaction")

	var bobFlashblockTime time.Time
	var bobFlashblock *Flashblock
	var observedBobBalance string

	for _, msg := range streamedMessages {
		flashblock := &Flashblock{}
		err := json.Unmarshal(msg.message, flashblock)
		require.NoError(t, err, "should be able to unmarshal the message")

		bobBalance := flashblock.Metadata.NewAccountBalances[strings.ToLower(bobAddress)]
		if bobBalance != "" && bobBalance != "0x0" {
			bobFlashblockTime = msg.timestamp
			bobFlashblock = flashblock
			observedBobBalance = bobBalance
			break
		}
	}

	require.NotNil(t, bobFlashblock, "should have received a flashblock corresponding to Bob's receival of the funds")

	txBlock, err := executedTransaction.IncludedBlock.Eval(ctx)
	require.NoError(t, err, "should be able to evaluate the block in which the transaction was included")

	txBlockNum := int(txBlock.Number)                                   // block number of the block in which the transaction was included / confirmed
	flashblockParentBlockNum := int(bobFlashblock.Metadata.BlockNumber) // block number of the parent block of the flashblock which first recorded the update in Bob's account balance (representative of the flashblock which included this transaction)

	txBlockTimeSeconds := int64(txBlock.Time)             // timestamp of the block in which the transaction was included / confirmed
	txFlashblockTimeInSeconds := bobFlashblockTime.Unix() // timestamp of the flashblock which supposedly included Bob's transaction

	require.Equal(t, observedBobBalance, expectedBobBalance, "Bob's balance must be correct as per exactly what Alice transferred to them")
	require.Equal(t, txBlockNum, flashblockParentBlockNum, "the transaction's block number should be the same as the flashblock's parent block number")
	require.LessOrEqual(t, txFlashblockTimeInSeconds, txBlockTimeSeconds, "the transaction's block time (in seconds) should be less than or equal to the flashblock's time (in seconds)")
	require.Less(t, bobFlashblockTime.UnixNano(), transactionApproxConfirmationTime.UnixNano(), "flashblock time should be before the transaction's (approximated) confirmation time")
}
