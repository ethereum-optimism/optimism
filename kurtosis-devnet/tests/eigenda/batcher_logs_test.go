package eigenda

import (
	"context"
	"testing"
	"time"

	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

func TestBatcherFromLogs_Holesky(t *testing.T) {
	deadline, ok := t.Deadline()
	// !ok means no timeout was set, and hence uses golang's default 10min timeout.
	if !ok || time.Until(deadline) < 15*time.Minute {
		t.Logf("TestBatcherFromLogs_Holesky needs a timeout of at least 15 minutes to run.")
		t.FailNow()
	}
	harness := NewHarness(t)
	// Batching time on Holesky can be up to 10 minutes, so we need long time to see a tx getting confirmed.
	testBatcherFromLogs(t, harness, 15*time.Minute)
}

func TestBatcherFromLogs_Memstore(t *testing.T) {
	harness := NewHarness(t)
	// 2 minutes is arbitrary here but should be long enough to observe interesting behavior using memstore.
	// Also need to run the failover test which takes quite a while and can't run in parallel with these tests (or can it..?)
	testBatcherFromLogs(t, harness, 2*time.Minute)
}

// These tests are log driven. The batcher doesn't expose an API to query its state outside of logs and metrics,
// so hard to do much better. We rely on some info logs appearing and some warning/error logs not appearing.
// These tests are not very sophisticated, but are at least a good sanity check...
// FIXME: one issue is that if op changes the log lines then our tests here might just silently pass and we won't know...
// A better approach might be to generate txs from inside the golang test instead of relying on the external tx-fuzzer.
// We could then increase traffic until the point where DA gets throttled, then change batcher parameters to increase blob size, etc.
// Updating the batcher params is currently hard to do however; see comments above the eigenda-devnet-restart-batcher command in the justfile.
func testBatcherFromLogs(t *testing.T, harness *Harness, testTimeout time.Duration) {
	// We stream logs for testTimeout, and run all the below tests in parallel (they read the same log outputs)
	ctxWithTestTimeout, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	// Make sure that no channel is ever timing out (fails to be sent to L1 in timely manner).
	// Make sure the testsTimer is longer than max-channel-duration in the batcher config (found in the eigenda-template-values/ files).
	// Currently max-channel-duration is set to 10 L1 blocks, meaning 10*6 seconds = 60 seconds.
	t.Run("No channel timeout", func(t *testing.T) {
		t.Parallel()
		// Log output is from https://github.com/Layr-Labs/optimism/blob/a5709b435f39cab0d7f5dc879b65e07e2f90a548/op-batcher/batcher/channel.go#L102
		filter := kurtosis_context.NewDoesContainMatchRegexLogLineFilter("channel timed out")
		c := harness.QueryBatcherLogs(ctxWithTestTimeout, true, filter)

		for {
			select {
			case <-ctxWithTestTimeout.Done():
				return
			case logLine := <-c:
				t.Logf("channel timed out on batcher... something went wrong. Log line: %v", logLine)
				t.FailNow()
			}
		}
	})

	t.Run("No DA Throttling", func(t *testing.T) {
		t.Parallel()
		// Log output is from https://github.com/Layr-Labs/optimism/blob/a5709b435f39cab0d7f5dc879b65e07e2f90a548/op-batcher/batcher/driver.go#L540
		filter := kurtosis_context.NewDoesContainMatchRegexLogLineFilter("throttling DA")
		c := harness.QueryBatcherLogs(ctxWithTestTimeout, true, filter)

		for {
			select {
			case <-ctxWithTestTimeout.Done():
				return
			case logLine := <-c:
				t.Logf("da got throttled... something went wrong. Log line: %v", logLine)
				t.FailNow()
			}
		}
	})

	t.Run("Transactions are confirming", func(t *testing.T) {
		t.Parallel()
		// Log line from https://github.com/Layr-Labs/optimism/blob/a5709b435f39cab0d7f5dc879b65e07e2f90a548/op-batcher/batcher/driver.go#L921
		// Actually there's a duplicate log line: https://github.com/Layr-Labs/optimism/blob/a5709b435f39cab0d7f5dc879b65e07e2f90a548/op-service/txmgr/txmgr.go#L780
		// We should prob divide by 2 but leaving as is in case this duplicate gets removed in the future...
		filter := kurtosis_context.NewDoesContainMatchRegexLogLineFilter("Transaction confirmed")
		c := harness.QueryBatcherLogs(ctxWithTestTimeout, true, filter)

		confirmedTxsCount := 0
		for {
			select {
			case <-ctxWithTestTimeout.Done():
				if confirmedTxsCount == 0 {
					t.Logf("no transactions confirmed... something went wrong.")
					t.FailNow()
				}
				t.Logf("%d transactions confirmed", confirmedTxsCount)
				return
			case <-c:
				confirmedTxsCount++
			}
		}
	})
}
