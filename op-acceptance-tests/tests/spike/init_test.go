package safeheaddb_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainMultiNode(),
		presets.WithConsensusLayerSync(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
