package cannon

import (
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithProofs(),
		presets.WithJovianAtGenesis(),
		presets.WithSafeDBEnabled(),
		// Use a slow game type to be sure the anchor state doesn't update
		presets.WithProposerGameType(faultTypes.PermissionedGameType),
		// Requires access to a challenger config which only sysgo provides
		// These tests would also be exceptionally slow on real L1s
		presets.WithCompatibleTypes(compat.SysGo))
}
