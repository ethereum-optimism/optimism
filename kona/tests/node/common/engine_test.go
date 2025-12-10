package node

import (
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

func TestEngine(gt *testing.T) {
	t := devtest.ParallelT(gt)

	out := node_utils.NewMixedOpKona(t)

	// Get the nodes from the network.
	nodes := out.L2CLKonaNodes()

	wg := sync.WaitGroup{}
	for _, node := range nodes {
		wg.Add(1)
		go func(node *dsl.L2CLNode) {
			defer wg.Done()

			queue := make(chan []uint64)

			// Spawn a task that gets the engine queue length with a ws connection.
			go func() {
				done := make(chan struct{})
				go func() {
					// Wait for 40 unsafe blocks to be produced.
					node.Advanced(types.LocalUnsafe, 40, 100)
					done <- struct{}{}
				}()

				queue <- node_utils.GetDevWS(t, node, "engine_queue_size", done)
			}()

			q := <-queue
			for _, q := range q {
				require.LessOrEqual(t, q, uint64(1), "engine queue length should be 1 or less")
			}
		}(&node)
	}

	wg.Wait()

}
