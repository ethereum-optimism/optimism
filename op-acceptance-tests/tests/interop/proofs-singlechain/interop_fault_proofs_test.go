package proofs_singlechain

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInteropSupernodeProofs(t,
		presets.WithChallengerCannonKonaEnabled(),
		presets.WithSuperGameTypeAdded(gameTypes.SuperCannonGameType, sysgo.PrestateForGameType(t, gameTypes.SuperCannonGameType)),
		presets.WithSuperGameTypeAdded(gameTypes.SuperCannonKonaGameType, sysgo.PrestateForGameType(t, gameTypes.SuperCannonKonaGameType)),
	)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys)
}
