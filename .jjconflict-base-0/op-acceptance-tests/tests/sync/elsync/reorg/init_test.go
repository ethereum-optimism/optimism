package reorg

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithNewSingleChainMultiNodeWithTestSeq(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithNoDiscovery(),
	)
}
