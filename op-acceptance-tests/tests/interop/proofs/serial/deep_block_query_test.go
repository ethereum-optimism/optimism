package serial

import (
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestInteropFaultProofs_DeepCanonicalBlockQuery(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInteropSupernodeProofs(t,
		presets.WithTimeTravelEnabled(),
		presets.WithChallengerCannonKonaEnabled(),
	)
	sfp.RunDeepCanonicalBlockQueryTest(t, sys)
}
