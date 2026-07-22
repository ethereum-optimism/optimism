package sp1

import (
	"os"
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

const konaSP1SuperRangeELFExecutorPathEnv = "KONA_SP1_SUPER_RANGE_ELF_EXECUTOR_PATH"

func TestSP1SuperRangeELFExecutionValidCrossChainMessageSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	executorPath := os.Getenv(konaSP1SuperRangeELFExecutorPathEnv)
	if executorPath == "" {
		t.Skip("kona-sp1 super-range ELF executor not provided; set " + konaSP1SuperRangeELFExecutorPathEnv)
	}

	sys := presets.NewSimpleInterop(t)
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys, sfp.NewSP1FullProofRunner(executorPath))
}
