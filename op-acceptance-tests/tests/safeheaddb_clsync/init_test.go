package safeheaddb_elsync

import (
	"testing"

	"github.com/HashKeyChain/verse/op-devstack/compat"
	"github.com/HashKeyChain/verse/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainMultiNode(),
		presets.WithConsensusLayerSync(),
		presets.WithSafeDBEnabled(),
		// Destructive test that requiring an in-memory only geth database
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
