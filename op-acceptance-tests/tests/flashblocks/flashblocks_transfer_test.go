//go:build !ci

// use a tag prefixed with "!". Such tag ensures that the default behaviour of this test would be to be built/run even when the go toolchain (go test) doesn't specify any tag filter.
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
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/log"
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
	sys := presets.NewSimpleFlashblocks(t)
	logger := testlog.Logger(t, log.LevelInfo).With("Test", "TestFlashblocksTransfer")
	tracer := t.Tracer()
	ctx := t.Ctx()
	logger.Info("Started Flashblocks Transfer test")

	topLevelCtx, span := tracer.Start(ctx, "test chains")
	defer span.End()

	// Test all L2 chains in the system
	for l2Chain, funder := range sys.Funders {
		ctx, cancel := context.WithTimeout(topLevelCtx, 45*time.Second)
		defer cancel()
		_, span = tracer.Start(ctx, fmt.Sprintf("test chain %s", l2Chain.String()))
		defer span.End()

		t.Run(fmt.Sprintf("L2_Chain_%s", l2Chain.String()), func(tt devtest.T) {
			if len(sys.FlashblocksBuilderSets[l2Chain]) == 0 && len(sys.FlashblocksWebsocketProxies[l2Chain]) == 0 {
				tt.Skip("no flashblocks builders or websocket proxies found for chain", l2Chain.String())
			}

			doneListening := make(chan struct{})
			output := make(chan []byte, 100)

			alice := funder.NewFundedEOA(eth.ThreeHundredthsEther)
			bob := sys.Wallet.NewEOA(sys.L2ELNodes[l2Chain])
			bobAddress := bob.Address().Hex()

			// flashblocks listener
			fbWsProxies := sys.FlashblocksWebsocketProxies[l2Chain]
			if len(fbWsProxies) > 0 {
				fbWsProxy := fbWsProxies[0]
				logger.Info("Listening for flashblocks via websocket proxy", "proxy", fbWsProxy.String())
				go fbWsProxy.ListenFor(logger, 20*time.Second, output, doneListening) //nolint:errcheck
			} else {
				leaderFbBuilder := sys.FlashblocksBuilderSets[l2Chain].Leader()
				require.NotNil(tt, leaderFbBuilder, "should have a leader", "chain", l2Chain.String())

				logger.Info("Listening for flashblocks via flashblocks builder", "builder", leaderFbBuilder.String())

				go leaderFbBuilder.ListenFor(logger, 20*time.Second, output, doneListening) //nolint:errcheck
			}

			var executedTransaction *txplan.PlannedTx
			var transactionApproxConfirmationTime time.Time
			var expectedBobBalance string

			// transactor
			go func() {
				time.Sleep(6 * time.Second) // warm up for the websocket handshake to be established
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
			for {
				select {
				case <-doneListening:
					goto done
				case msg := <-output:
					streamedMessages = append(streamedMessages, timedMessage{message: msg, timestamp: time.Now()})
				}
			}
		done:
			require.Greater(tt, len(streamedMessages), 0, "should have received at least one message from the flashblocks stream")
			require.NotNil(tt, executedTransaction, "should have executed a transaction")

			var bobFlashblockTime time.Time
			var bobFlashblock *Flashblock
			var observedBobBalance string

			for _, msg := range streamedMessages {
				flashblock := &Flashblock{}
				err := json.Unmarshal(msg.message, flashblock)
				require.NoError(tt, err, "should be able to unmarshal the message")

				bobBalance := flashblock.Metadata.NewAccountBalances[strings.ToLower(bobAddress)]
				if bobBalance != "" && bobBalance != "0x0" {
					bobFlashblockTime = msg.timestamp
					bobFlashblock = flashblock
					observedBobBalance = bobBalance
					break
				}
			}

			require.NotNil(tt, bobFlashblock, "should have received a flashblock corresponding to Bob's receival of the funds")

			txBlock, err := executedTransaction.IncludedBlock.Eval(ctx)
			require.NoError(tt, err, "should be able to evaluate the block in which the transaction was included")

			txBlockNum := int(txBlock.Number)                                   // block number of the block in which the transaction was included / confirmed
			flashblockParentBlockNum := int(bobFlashblock.Metadata.BlockNumber) // block number of the parent block of the flashblock which first recorded the update in Bob's account balance (representative of the flashblock which included this transaction)

			txBlockTimeSeconds := int64(txBlock.Time)             // timestamp of the block in which the transaction was included / confirmed
			txFlashblockTimeInSeconds := bobFlashblockTime.Unix() // timestamp of the flashblock which supposedly included Bob's transaction

			require.Equal(tt, observedBobBalance, expectedBobBalance, "Bob's balance must be correct as per exactly what Alice transferred to them")
			require.Equal(tt, txBlockNum, flashblockParentBlockNum, "the transaction's block number should be the same as the flashblock's parent block number")
			require.LessOrEqual(tt, txFlashblockTimeInSeconds, txBlockTimeSeconds, "the transaction's block time (in seconds) should be less than or equal to the flashblock's time (in seconds)")
			require.Less(tt, bobFlashblockTime.UnixNano(), transactionApproxConfirmationTime.UnixNano(), "flashblock time should be before the transaction's (approximated) confirmation time")
		})
	}
}
