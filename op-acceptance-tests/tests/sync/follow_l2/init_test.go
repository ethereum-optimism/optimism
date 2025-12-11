package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainTwoVerifiersFollowL2(),
		presets.WithReqRespSyncDisabled(),
		presets.WithNoDiscovery(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
