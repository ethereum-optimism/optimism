package sp1

import (
	"os"
	"path/filepath"
	"testing"

	sfp "github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/superfaultproofs"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

const (
	konaSP1SuperRangeELFExecutorPathEnv = "KONA_SP1_SUPER_RANGE_ELF_EXECUTOR_PATH"
	konaSP1ELFDirEnv                    = "KONA_SP1_ELF_DIR"
)

func TestSP1SuperRangeELFExecutionValidCrossChainMessageSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	executorPath := requireSP1FullELF(t)

	sys := presets.NewSimpleInterop(t)
	sfp.RunConsolidateValidCrossChainMessageTest(t, sys, sfp.NewSP1FullProofRunner(executorPath))
}

func TestSP1SuperRangeELFExecutionSingleChainSmoke(gt *testing.T) {
	t := devtest.SerialT(gt)
	executorPath := requireSP1FullELF(t)

	sys := presets.NewSingleChainInterop(t)
	sfp.RunSingleChainSuperFaultProofSmokeTest(t, sys, sfp.NewSP1FullProofRunner(executorPath))
}

func requireSP1FullELF(t devtest.T) string {
	executorPath := os.Getenv(konaSP1SuperRangeELFExecutorPathEnv)
	elfDir := os.Getenv(konaSP1ELFDirEnv)
	if executorPath == "" && elfDir == "" {
		t.Skip("kona-sp1 full-ELF execution not configured; set " + konaSP1SuperRangeELFExecutorPathEnv)
	}

	t.Require().NotEmpty(executorPath, konaSP1SuperRangeELFExecutorPathEnv+" must be set when "+konaSP1ELFDirEnv+" is set")
	executorInfo, err := os.Stat(executorPath)
	t.Require().NoError(err, "stat kona-sp1 super-range ELF executor")
	t.Require().False(executorInfo.IsDir(), "kona-sp1 super-range ELF executor must be a file")
	t.Require().NotZero(executorInfo.Mode().Perm()&0o111, "kona-sp1 super-range ELF executor must be executable")

	t.Require().NotEmpty(elfDir, konaSP1ELFDirEnv+" must be set for full-ELF execution")
	elfInfo, err := os.Stat(filepath.Join(elfDir, "super-range-elf"))
	t.Require().NoError(err, "stat kona-sp1 super-range ELF")
	t.Require().False(elfInfo.IsDir(), "kona-sp1 super-range ELF must be a file")
	t.Require().Positive(elfInfo.Size(), "kona-sp1 super-range ELF must not be empty")
	return executorPath
}
