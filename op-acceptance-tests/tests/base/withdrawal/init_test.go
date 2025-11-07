package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		// TODO(infra#401): Re-enable the test when the sysext missing toolset is implemented
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithMinimal(),
		presets.WithTimeTravel(),
		presets.WithFinalizationPeriodSeconds(1),
		// Satisfy OptimismPortal2 PROOF_MATURITY_DELAY_SECONDS check, avoid OptimismPortal_ProofNotOldEnough() revert
		presets.WithProofMaturityDelaySeconds(2),
		// Satisfy AnchorStateRegistry DISPUTE_GAME_FINALITY_DELAY_SECONDS check, avoid OptimismPortal_InvalidRootClaim() revert
		presets.WithDisputeGameFinalityDelaySeconds(2),
	)
}
