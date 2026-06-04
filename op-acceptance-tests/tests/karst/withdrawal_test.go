package karst

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
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

// TestWithdrawal_KarstUpgrade is the same withdrawal flow but on a network that
// started pre-Karst and activated Karst mid-chain via a scheduled upgrade.
func TestWithdrawal_KarstUpgrade(gt *testing.T) {
	offset := uint64(10) // arbitrary offset to have a few blocks before Karst
	withdrawal.TestWithdrawalAfterUpgrade(gt, gameTypes.CannonGameType, opforks.Karst,
		presets.WithDeployerOptions(sysgo.WithKarstAtOffset(&offset)),
	)
}
