package flashblocks

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
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

	head := sys.L2EL.BlockRefByLabel(eth.Unsafe)
	for range count {
		require.NoError(t, ts.New(ctx, seqtypes.BuildOpts{Parent: head.Hash}))
		require.NoError(t, ts.Next(ctx))
		head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	}
	// Ensure the sequencer EL has produced at least one unsafe block before subscribing.
	sys.L2EL.WaitForBlockNumber(1)

	// Log the latest unsafe head and L1 origin to confirm block production before listening.
	head = sys.L2EL.BlockRefByLabel(eth.Unsafe)
	sys.Log.Info("Pre-listen unsafe head", "unsafe", head)
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
