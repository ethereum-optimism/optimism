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

// ensureFlashblocksIncrease validates the flashblock stream protocol: sequences are keyed by
// payload_id (not block_number); within a sequence, index strictly increases; a new sequence
// starts at index 0 and may reuse a block_number when the sequencer rebuilds a block. See
// rust/op-reth/crates/flashblocks/src/sequence.rs ("A [`FlashBlock`] with index 0 resets the
// set.").
func ensureFlashblocksIncrease(t devtest.T, wsClient *client.WSClient, logger log.Logger) {
	const numFlashblocks = 20
	// ~30s of headroom at the 200ms flashblock cadence, so CI stalls don't cause the reader
	// to drop messages (see flashblock_client.go's "channel full, dropping flashblock" path).
	const flashblockBufferCapacity = 150
	client := sources.NewFlashblockClient(wsClient, logger, flashblockBufferCapacity)
	startClient(t, client)

	// Devstack dials the WS eagerly, so the first message may be mid-sequence. Wait for a
	// clean index=0 anchor before counting transitions.
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
				if flashblock.Index != 0 {
					continue
				}
				anchored = true
			} else if flashblock.PayloadID == lastPayloadID {
				t.Require().Equal(lastBlockNumber, flashblock.Metadata.BlockNumber,
					"block_number must be stable within a flashblock sequence (payload_id=%s)",
					flashblock.PayloadID)
				t.Require().Greater(flashblock.Index, lastIndex,
					"flashblock index must strictly increase within a sequence (payload_id=%s)",
					flashblock.PayloadID)
			} else {
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
