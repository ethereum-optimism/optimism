package unsafe_only

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainTwoVerifiers(),
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithReqRespSyncDisabled(),
		presets.WithNoDiscovery(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithUnsafeOnly(),
	)
}
