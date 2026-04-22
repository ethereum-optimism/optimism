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
	// TODO(ethereum-optimism/optimism#19883): investigate and unmark.
	t.MarkFlaky("ethereum-optimism/optimism#19883")
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

func ensureFlashblocksIncrease(t devtest.T, wsClient *client.WSClient, logger log.Logger) {
	const numFlashblocks = 20
	client := sources.NewFlashblockClient(wsClient, logger, numFlashblocks)
	startClient(t, client)

	lastBlockNumber := -1
	lastIndex := -1
	for range numFlashblocks {
		select {
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err(), "before %d flashblocks were seen", numFlashblocks)
		case flashblock, ok := <-client.Next():
			t.Require().True(ok, "client channel closed before we saw %d flashblocks", numFlashblocks)
			t.Require().NotNil(flashblock)
			currentBlockNumber := flashblock.Metadata.BlockNumber
			currentIndex := flashblock.Index

			if currentBlockNumber == lastBlockNumber {
				t.Require().Greater(currentIndex, lastIndex)
			} else {
				t.Require().Greater(currentBlockNumber, lastBlockNumber)
				t.Require().Zero(currentIndex)
			}

			lastBlockNumber = currentBlockNumber
			lastIndex = currentIndex
		}
	}
}
