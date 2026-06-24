package helpers

import (
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/params"
)

// sp1RangeExecutorPath is the path to a pre-built kona-sp1-range-executor binary. The binary embeds
// the range guest ELF, so it must be built after the ELFs (see rust/kona/sp1/justfile build-elfs).
var sp1RangeExecutorPath string

func init() {
	sp1RangeExecutorPath = os.Getenv("KONA_SP1_RANGE_EXECUTOR_PATH")
}

// SP1RangeExecutorPath returns the configured kona-sp1-range-executor binary path, or "" if unset.
// Tests should skip when this is empty, since building the binary requires the SP1 toolchain and a
// Dockerized ELF build that is not always available.
func SP1RangeExecutorPath() string {
	return sp1RangeExecutorPath
}

// RunRangeExecutor runs the kona-sp1 range program for the state transition described by
// fixtureInputs. It mirrors RunKonaNative but, instead of running the native client, generates the
// witness and executes the range ELF (or the shared native range core when SP1NativeCore is set).
// It returns ErrClaimNotValid when the program rejects the claim.
//
// fixtureInputs.L2Claim must be the real/honest output root: witness generation runs on it, and a
// wrong root is rejected host-side before the guest runs (surfacing as an infra error, not a claim
// verdict). To test an invalid claim, keep the honest claim and set fixtureInputs.CorruptClaim
// (see WithCorruptClaim), which tampers the claim in the witness so the guest rejects it.
// Set fixtureInputs.SP1NativeCore (see WithSP1NativeCore) to run the shared range-program core
// natively instead of executing the SP1 ELF.
func RunRangeExecutor(
	t helpers.Testing,
	workDir string,
	rollupCfgs []*rollup.Config,
	l1chainConfig *params.ChainConfig,
	l1Rpc string,
	l1BeaconRpc string,
	l2Rpcs []string,
	fixtureInputs FixtureInputs,
) error {
	// Write rollup config(s) and the L1 chain config to the work dir, same as the native runner.
	rollupCfgPaths := make([]string, len(rollupCfgs))
	writeConfigs(t, workDir, "rollup", rollupCfgs, rollupCfgPaths)

	l1chainConfigPath := filepath.Join(workDir, "l1chain.json")
	writeConfig(t, workDir, "l1chain", l1chainConfig, l1chainConfigPath)

	vmCfg := vm.Config{
		L1:                l1Rpc,
		L1Beacon:          l1BeaconRpc,
		L2s:               l2Rpcs,
		RollupConfigPaths: rollupCfgPaths,
		L1GenesisPath:     l1chainConfigPath,
		Server:            sp1RangeExecutorPath,
	}
	inputs := utils.LocalGameInputs{
		L1Head:           fixtureInputs.L1Head,
		L2Head:           fixtureInputs.L2Head,
		L2Claim:          fixtureInputs.L2Claim,
		L2SequenceNumber: big.NewInt(int64(fixtureInputs.L2BlockNumber)),
		L2OutputRoot:     fixtureInputs.L2OutputRoot,
	}

	logger := log.NewLogger(os.Stdout, log.DefaultCLIConfig())

	var extraArgs []string
	if fixtureInputs.CorruptClaim {
		extraArgs = append(extraArgs, "--corrupt-claimed-root")
	}
	if fixtureInputs.SP1NativeCore {
		extraArgs = append(extraArgs, "--native-core")
	}

	if !rustbin.RunKonaSP1Range(t, logger, &vmCfg, workDir, &inputs, extraArgs...) {
		return ErrClaimNotValid
	}
	return nil
}
