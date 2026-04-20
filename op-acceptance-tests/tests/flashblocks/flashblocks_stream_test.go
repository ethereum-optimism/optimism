package flashblocks

import (
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/log"
)

// TestFlashblocksStream checks that block numbers and indices always increase across both the
// rollup-boost and op-rbuilder streams.
func TestFlashblocksStream(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-30|22:44:52.250]
	// assertions.go:387:             	Error Trace:	/home/circleci/project/op-devstack/sysgo/l2_cl_kona.go:99
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/mixed_runtime.go:456
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/singlechain_build.go:182
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/singlechain_build.go:276
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/singlechain_flashblocks.go:36
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/singlechain_runtime.go:105
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/sysgo/singlechain_flashblocks.go:53
	// assertions.go:387:             	            				/home/circleci/project/op-devstack/presets/flashblocks.go:43
	// assertions.go:387:             	            				/home/circleci/project/op-acceptance-tests/tests/flashblocks/flashblocks_stream_test.go:26
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	context deadline exceeded
	// assertions.go:387:             	Test:       	TestFlashblocksStream
	// assertions.go:387:             	Messages:   	need user RPC
	sysgo.SkipOnKonaNode(t, "not supported (fail to get user rpc)")
	sys := presets.NewSingleChainWithFlashblocks(t)

	driveViaTestSequencer(t, sys, 3)

	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ensureFlashblocksIncrease(t, sys.L2OPRBuilder.FlashblocksClient(), t.Logger().With("stream_source", "op-rbuilder"))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ensureFlashblocksIncrease(t, sys.L2RollupBoost.FlashblocksClient(), t.Logger().With("stream_source", "rollup-boost"))
	}()

	// Note that rollup boost may deliberately drop flashblocks from rbuilder to mitigate
	// flashblock reorgs. See https://blog.base.dev/flashblocks-deep-dive.
	// Otherwise, we could assert that the streams match (after aligning on the same start and end
	// flashblocks).
}

// ensureFlashblocksIncrease validates that the flashblock stream preserves the documented
// ordering invariants:
//
//   - Each flashblock belongs to a "sequence" identified by its payload_id. A sequence is
//     always anchored at index 0 and indices within a sequence increase monotonically.
//   - A new payload_id starts a new sequence at index 0. This can happen for the SAME
//     block_number when the sequencer rebuilds a block (e.g. a late FCU causes the builder
//     to start a fresh flashblock sequence for the same block number but a different
//     payload). See rust/op-reth/crates/flashblocks/src/sequence.rs which states that
//     "A [`FlashBlock`] with index 0 resets the set."
//   - Block numbers never decrease across sequences.
//
// The old assertion "within the same block_number indices strictly increase" was incorrect:
// under CI load the sequencer can rebuild a block, producing multiple sequences of flashblocks
// sharing the same block_number, which made the stream look like "index 1 then index 0".
func ensureFlashblocksIncrease(t devtest.T, wsClient *client.WSClient, logger log.Logger) {
	const numFlashblocks = 20
	// Size the client's channel buffer to hold roughly 30 seconds of flashblock history.
	// Flashblocks are emitted at a 200 ms cadence (see
	// rust/op-reth/crates/flashblocks/src/cache.rs: FLASHBLOCK_BLOCK_TIME = 200), so
	// 30 s ≈ 150 flashblocks. Under CI load the test goroutine can be stalled for a few
	// seconds at a time while the builder, rollup-boost and sequencer all contend for CPU;
	// a tight buffer risks the reader dropping messages (see flashblock_client.go's
	// "channel full, dropping flashblock" path) before the test has validated
	// numFlashblocks transitions. 30 s of headroom is generous enough to survive realistic
	// stalls without growing memory meaningfully.
	const flashblockBufferCapacity = 150
	client := sources.NewFlashblockClient(wsClient, logger, flashblockBufferCapacity)
	startClient(t, client)

	// The WS connection is opened eagerly during devstack setup, so by the time this loop
	// starts there may already be a backlog of messages from an in-progress sequence — the
	// first message we read is not guaranteed to be index 0. We validate `numFlashblocks`
	// consecutive transitions AFTER we observe an index=0 anchor so the assertions start
	// from a deterministic point.
	anchored := false
	validated := 0
	var lastPayloadID string
	var lastBlockNumber int
	var lastIndex int
	for validated < numFlashblocks {
		select {
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "before %d flashblocks were validated", numFlashblocks)
		case flashblock, ok := <-client.Next():
			t.Require().True(ok, "client channel closed before we validated %d flashblocks", numFlashblocks)
			t.Require().NotNil(flashblock)

			if !anchored {
				// Skip mid-sequence flashblocks until we find a fresh sequence start.
				if flashblock.Index != 0 {
					continue
				}
				anchored = true
			} else if flashblock.PayloadID == lastPayloadID {
				// Continuation of the current sequence: block number is stable, index must
				// strictly increase.
				t.Require().Equal(lastBlockNumber, flashblock.Metadata.BlockNumber,
					"block_number must be stable within a flashblock sequence (payload_id=%s)",
					flashblock.PayloadID)
				t.Require().Greater(flashblock.Index, lastIndex,
					"flashblock index must strictly increase within a sequence (payload_id=%s)",
					flashblock.PayloadID)
			} else {
				// New sequence: index must be 0, and the block_number must not go backwards.
				t.Require().Zero(flashblock.Index,
					"a new flashblock sequence (payload_id %s → %s) must start at index 0",
					lastPayloadID, flashblock.PayloadID)
				t.Require().GreaterOrEqual(flashblock.Metadata.BlockNumber, lastBlockNumber,
					"block_number must not decrease across flashblock sequences (payload_id %s → %s)",
					lastPayloadID, flashblock.PayloadID)
			}

			lastPayloadID = flashblock.PayloadID
			lastBlockNumber = flashblock.Metadata.BlockNumber
			lastIndex = flashblock.Index
			validated++
		}
	}
}
