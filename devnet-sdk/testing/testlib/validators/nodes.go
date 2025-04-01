package validators

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
)

// L2NodeCounter is a function type that retrieves the node count from a context.
type L2NodeCounter = func(context.Context) int

// HasSufficientL2Nodes returns a validator that ensures a specific L2 chain has at least the specified number of nodes.
func HasSufficientL2Nodes(chainIdx uint64, minNodes int) systest.PreconditionValidator {
	return func(t systest.T, sys system.System) (context.Context, error) {
		if len(sys.L2s()) <= int(chainIdx) {
			return nil, fmt.Errorf("chain index %d out of range, only %d L2 chains available", chainIdx, len(sys.L2s()))
		}

		chain := sys.L2s()[chainIdx]
		nodeCount := len(chain.Nodes())

		if nodeCount < minNodes {
			return nil, fmt.Errorf("insufficient nodes for L2 chain %d: has %d, requires %d", chainIdx, nodeCount, minNodes)
		}

		return t.Context(), nil
	}
}

// AcquireL2NodeCount returns a node counter function and a validator that ensures an L2 chain
// exists and provides the node count in the context.
func AcquireL2NodeCount(chainIdx uint64) (L2NodeCounter, systest.PreconditionValidator) {
	nodeCountMarker := new(byte)

	validator := func(t systest.T, sys system.System) (context.Context, error) {
		if len(sys.L2s()) <= int(chainIdx) {
			return nil, fmt.Errorf("chain index %d out of range, only %d L2 chains available", chainIdx, len(sys.L2s()))
		}

		chain := sys.L2s()[chainIdx]
		nodeCount := len(chain.Nodes())

		return context.WithValue(t.Context(), nodeCountMarker, nodeCount), nil
	}

	counter := func(ctx context.Context) int {
		return ctx.Value(nodeCountMarker).(int)
	}

	return counter, validator
}
