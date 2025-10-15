package depreqres

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSingleChainMultiNode(),
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithReqRespSyncDisabled(),
	)
}
