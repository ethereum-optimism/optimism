package flashblocks

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/stretchr/testify/require"
)

// driveViaTestSequencer explicitly builds a few blocks to ensure the builder/rollup-boost
// have payloads to serve before we start listening for flashblocks.
func driveViaTestSequencer(t devtest.T, sys *presets.SingleChainWithFlashblocks, count int) {
	t.Helper()
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L2Chain.ChainID())
	ctx := t.Ctx()
	builderClient := sys.L2OPRBuilder.Escape().L2EthClient()

	head := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	for range count {
		require.NoError(t, ts.New(ctx, seqtypes.BuildOpts{Parent: head.Hash}))
		require.NoError(t, ts.Next(ctx))
		head = sys.L2EL.BlockRefByLabel(eth.Unsafe)

		// Wait for the builder EL to sync this block via P2P before driving the
		// next one. Without this, the builder may not have the parent when the
		// next FCU arrives, causing "missing parent header" → InvalidPayloadAttributes
		// → CriticalError under CI load.
		waitForBuilderBlock(t, ctx, builderClient, head.Number)
	}
	// Ensure the sequencer EL has produced at least one unsafe block before subscribing.
	sys.L2EL.WaitForBlockNumber(1)

	// Wait until L2 time catches up to wall clock time before relying on flashblocks.
	// During startup the builder may report "unsafe block timestamp is too old" /
	// "FCU arrived too late" while the sequencer is still catching up, which makes
	// early flashblock receipt assertions flaky.
	sys.L2EL.WaitForTime(uint64(time.Now().Unix()))

	// Log the latest unsafe head and L1 origin to confirm block production before listening.
	head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	sys.Log.Info("Pre-listen unsafe head", "unsafe", head)
}

// waitForBuilderBlock polls the builder EL until it has the given block number.
func waitForBuilderBlock(t devtest.T, ctx context.Context, builderClient apis.L2EthClient, blockNum uint64) {
	t.Helper()
	// 50ms: fast enough to not add meaningful delay, slow enough to not spam the builder RPC.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err(), "context expired waiting for builder to sync block %d", blockNum)
		case <-ticker.C:
			_, err := builderClient.L2BlockRefByNumber(ctx, blockNum)
			if err == nil {
				return
			}
		}
	}
}

func startClient(t devtest.T, client *sources.FlashblockClient) {
	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Require().NoError(client.Start(ctx))
	}()
}
