package fusaka

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum/go-ethereum/params"
)

// configureDevstackEnvVars sets the appropriate env vars to use a mise-installed geth binary for
// the L1 EL. This is useful in Osaka acceptance tests since op-geth does not include full Osaka
// support. This is meant to run before presets.DoMain in a TestMain function. It will log to
// stdout. ResetDevstackEnvVars should be used to reset the environment variables when TestMain
// exits.
//
// Note that this is a no-op if either [sysgo.DevstackL1ELKindVar] or [sysgo.GethExecPathEnvVar]
// are set.
//
// The returned callback resets any modified environment variables.
func configureDevstackEnvVars() func() {
	if _, ok := os.LookupEnv(sysgo.DevstackL1ELKindEnvVar); ok {
		return func() {}
	}
	if _, ok := os.LookupEnv(sysgo.GethExecPathEnvVar); ok {
		return func() {}
	}

	cmd := exec.Command("mise", "which", "geth")
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to find mise-installed geth: %v\n", err)
		return func() {}
	}
	execPath := strings.TrimSpace(buf.String())
	fmt.Println("Found mise-installed geth:", execPath)
	_ = os.Setenv(sysgo.GethExecPathEnvVar, execPath)
	_ = os.Setenv(sysgo.DevstackL1ELKindEnvVar, "geth")
	return func() {
		_ = os.Unsetenv(sysgo.GethExecPathEnvVar)
		_ = os.Unsetenv(sysgo.DevstackL1ELKindEnvVar)
	}
}

func TestMain(m *testing.M) {
	resetEnvVars := configureDevstackEnvVars()
	defer resetEnvVars()

	presets.DoMain(m, stack.MakeCommon(stack.Combine[*sysgo.Orchestrator](
		sysgo.DefaultMinimalSystem(&sysgo.DefaultMinimalSystemIDs{}),
		sysgo.WithDeployerOptions(func(_ devtest.P, _ devkeys.Keys, builder intentbuilder.Builder) {
			_, l1Config := builder.WithL1(sysgo.DefaultL1ID)
			l1Config.WithOsakaOffset(0)
			// Make the BPO fork happen after Osaka so we can easily use geth's eip4844.CalcBlobFee
			// to calculate the blob base fee using the Osaka parameters.
			l1Config.WithBPO1Offset(1)
			l1Config.WithL1BlobSchedule(&params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Osaka:  params.DefaultOsakaBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
				BPO1:   params.DefaultBPO1BlobConfig,
			})
		}),
		sysgo.WithBatcherOption(func(_ stack.L2BatcherID, cfg *batcher.CLIConfig) {
			cfg.DataAvailabilityType = flags.BlobsType
			cfg.TxMgrConfig.CellProofTime = 0 // Force cell proofs to be used
		}),
	)))
}
