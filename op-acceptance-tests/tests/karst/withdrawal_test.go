package karst

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestWithdrawal_Karst creates a withdrawal from the L2StandardBridge and
// observes the full withdrawal flow, including finalization on L1.
func TestWithdrawal_Karst(gt *testing.T) {
	withdrawal.TestWithdrawal(gt, gameTypes.CannonGameType,
		presets.WithDeployerOptions(sysgo.WithKarstAtGenesis),
	)
}
