package helpers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// sp1SuperRangeExecutorPath is the path to a pre-built kona-sp1-super-range-executor binary. The
// binary loads the super-range guest ELF from KONA_SP1_ELF_DIR at runtime.
var sp1SuperRangeExecutorPath string

func init() {
	sp1SuperRangeExecutorPath = os.Getenv("KONA_SP1_SUPER_RANGE_ELF_EXECUTOR_PATH")
}

// SP1SuperRangeExecutorPath returns the configured kona-sp1-super-range-executor binary path, or ""
// if unset. Tests should skip when this is empty, since building the binary requires the SP1
// toolchain and a Dockerized ELF build that is not always available.
func SP1SuperRangeExecutorPath() string {
	return sp1SuperRangeExecutorPath
}

// RunSuperRangeExecutor runs the kona-sp1 super-range program over the super-root transition
// described by fixtureInputs. It returns ErrClaimNotValid when the program rejects the claim.
//
// Unlike the native fault-proof runner, the executor does not take an agreed pre-state and a
// claimed post-state: it derives both itself by querying `superroot_atTimestamp` on
// fixtureInputs.SupernodeAddress for ClaimTimestamp-1 and ClaimTimestamp, then collects range and
// consolidation witnesses through kona's InteropHost. So the honest claim cannot be tampered from
// out here — pass fixtureInputs.CorruptClaim (see WithCorruptClaim) to have the executor corrupt
// the guest's view of the claim after witness collection, which is the invalid-claim path.
// Set fixtureInputs.SP1NativeCore (see WithSP1NativeCore) to replay the collected witnesses
// through the shared native cores instead of executing the SP1 ELF.
func RunSuperRangeExecutor(
	t helpers.Testing,
	workDir string,
	rollupCfgs []*rollup.Config,
	l1chainConfig *params.ChainConfig,
	l1Rpc string,
	l1BeaconRpc string,
	l2Rpcs []string,
	fixtureInputs FixtureInputs,
) error {
	require.NotEmpty(t, fixtureInputs.SupernodeAddress,
		"the super-range executor needs a supernode endpoint; see WithSupernode")

	// Write rollup config(s), the L1 chain config and the dependency set to the work dir, same as
	// the native interop runner: the executor passes all three straight through to InteropHost.
	rollupCfgPaths := make([]string, len(rollupCfgs))
	writeConfigs(t, workDir, "rollup", rollupCfgs, rollupCfgPaths)

	l1chainConfigPath := filepath.Join(workDir, "l1chain.json")
	writeConfig(t, workDir, "l1chain", l1chainConfig, l1chainConfigPath)

	depsetPath := filepath.Join(workDir, "depset.json")
	writeDepset(t, depsetPath, rollupCfgs)

	args := []string{
		"--supernode-address", fixtureInputs.SupernodeAddress,
		"--l1-node-address", l1Rpc,
		"--l1-beacon-address", l1BeaconRpc,
		"--l2-node-addresses", strings.Join(l2Rpcs, ","),
		"--l1-head", fixtureInputs.L1Head.Hex(),
		"--end-timestamp", strconv.FormatUint(fixtureInputs.ClaimTimestamp, 10),
		"--rollup-config-paths", strings.Join(rollupCfgPaths, ","),
		"--l1-config-path", l1chainConfigPath,
		"--depset-cfg", depsetPath,
	}
	if fixtureInputs.CorruptClaim {
		args = append(args, "--corrupt-claimed-root")
	}
	if fixtureInputs.SP1NativeCore {
		args = append(args, "--native-core")
	}

	logger := log.NewLogger(os.Stdout, log.DefaultCLIConfig())

	if !rustbin.RunKonaSP1SuperRange(t, logger, sp1SuperRangeExecutorPath, workDir, args...) {
		return ErrClaimNotValid
	}
	return nil
}
