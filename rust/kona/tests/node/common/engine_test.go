package node

import (
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
	"github.com/stretchr/testify/require"
)

func TestEngine(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := newCommonPreset(t)

	// Get the nodes from the network.
	nodes := out.L2CLKonaNodes()

	wg := sync.WaitGroup{}
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()

			done := make(chan struct{})
			go func() {
				defer close(done)
				// Wait for 40 unsafe blocks to be produced.
				node.Advanced(types.LocalUnsafe, 40, 100)
			}()

			queueLens := node_utils.GetDevWS(t, node, "engine_queue_size", done)
			for _, queueLen := range queueLens {
				require.LessOrEqual(t, queueLen, uint64(1), "engine queue length should be 1 or less")
			}
		}(&node)
	}

	wg.Wait()

}
