package chain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSingleChainMultiNode(),
	)
}

func TestSpanBatches(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)
	sys.L2ELB.WaitForLabel(eth.Safe, func(info eth.BlockInfo) (bool, error) { return info.NumberU64() > 15, nil })
}
