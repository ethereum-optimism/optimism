package withdrawal

import (
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal(),
		presets.WithProposerGameType(faultTypes.FastGameType),
		// Fast game for test
		presets.WithFastGame(),
		// Deployer must be L1PAO to make op-deployer happy
		presets.WithDeployerMatchL1PAO(),
		// Guardian must be L1PAO to make AnchorStateRegistry's setRespectedGameType method work
		presets.WithGuardianMatchL1PAO(),
		// Fast finalization for fast withdrawal
		presets.WithFinalizationPeriodSeconds(2),
		// Satisfy OptimismPortal2 PROOF_MATURITY_DELAY_SECONDS check, avoid OptimismPortal_ProofNotOldEnough() revert
		presets.WithProofMaturityDelaySeconds(12),
		// Satisfy AnchorStateRegistry DISPUTE_GAME_FINALITY_DELAY_SECONDS check, avoid OptimismPortal_InvalidRootClaim() revert
		presets.WithDisputeGameFinalityDelaySeconds(6),
	)
}
