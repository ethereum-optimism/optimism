package proofs_singlechain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSingleChainSuperInteropSupernode(),
		presets.WithL2NetworkCount(1),
		stack.MakeCommon(sysgo.WithChallengerCannonKonaEnabled()),
	)
}
