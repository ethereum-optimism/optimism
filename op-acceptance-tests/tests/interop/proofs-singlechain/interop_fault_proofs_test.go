package proofs_singlechain

import (
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestInteropSingleChainFaultProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainInterop(t)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys, proofRunners()...)
}

func TestInteropSingleChainFaultProofsWithSDM(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := sdmtest.NewFixtureSingleChainFaultProofSystem(t)
	sfp.RunSingleChainSuperFaultProofSDMSmokeTest(t, sys)
}

// TestInteropSingleChainFaultProofsWithPremiumSDMThroughCannon is owned and invoked by the
// premium block-producer repository. Keeping the reusable proof harness here lets Cannon execute
// the public kona-client while ensuring the block itself is built only by the external producer.
func TestInteropSingleChainFaultProofsWithPremiumSDMThroughCannon(gt *testing.T) {
	producer := os.Getenv("SDM_CANNON_PRODUCER_BINARY")
	if producer == "" {
		gt.Skip("set SDM_CANNON_PRODUCER_BINARY from the block-producer repository")
	}

	t := devtest.SerialT(gt)
	sysgo.SkipOnKonaNode(t, "super-cannon step disputes are not supported with kona-node")
	sys := sdmtest.NewSingleChainFaultProofSystemWithProducer(
		t,
		producer,
		strings.Fields(os.Getenv("SDM_CANNON_PRODUCER_ARGS"))...,
	)
	sfp.RunSingleChainSuperFaultProofSDMSmokeTest(t, sys, sfp.NewCannonKonaSuperProofRunner())
}

func proofRunners() []sfp.ProofRunner {
	runners := []sfp.ProofRunner{sfp.NewKonaProofRunner()}
	if os.Getenv("RUST_BINARY_PATH_KONA_SP1_SUPER_RANGE_EXECUTOR") != "" || os.Getenv("RUST_JIT_BUILD") != "" {
		runners = append(runners, sfp.NewSP1NativeProofRunner())
	}
	return runners
}
