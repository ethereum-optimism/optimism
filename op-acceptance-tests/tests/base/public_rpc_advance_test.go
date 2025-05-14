package base

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestPublicRpcAdvance(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)

	sys.L2Chain.PublicRPC().Advanced(eth.Unsafe, 5)
}
