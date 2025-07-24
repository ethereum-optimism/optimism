package base

import (
	"testing"

	"github.com/HashKeyChain/verse/op-devstack/devtest"
	"github.com/HashKeyChain/verse/op-devstack/presets"
	"github.com/HashKeyChain/verse/op-service/eth"
)

func TestPublicRpcAdvance(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t)

	sys.L2Chain.PublicRPC().Advanced(eth.Unsafe, 5)
}
