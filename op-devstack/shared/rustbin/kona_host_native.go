package rustbin

import (
	"errors"
	"os"
	"os/exec"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// RunKonaNative runs kona-host in --native mode. Returns false if kona disagrees and true
// otherwise.
func RunKonaNative(t require.TestingT, logger log.Logger, vmConfig *vm.Config, dir string, inputs *utils.LocalGameInputs) bool {
	require.NotNil(t, vmConfig)
	require.NotNil(t, inputs)
	args, err := vm.NewNativeKonaExecutor().OracleCommand(*vmConfig, dir, *inputs)
	require.NoError(t, err, "build kona oracle command")
	return runOracleServer(t, logger, "kona-host", args, dir)
}

// RunKonaSP1Range runs the kona-sp1 range-executor, which generates the witness via the kona-host
// preimage server and runs the range guest in SP1 execute mode. Returns false if the guest rejects
// the claim and true otherwise. The executor reuses the kona-host `single` oracle-server flags, so
// vmConfig.Server must point at the range-executor binary.
func RunKonaSP1Range(t require.TestingT, logger log.Logger, vmConfig *vm.Config, dir string, inputs *utils.LocalGameInputs, extraArgs ...string) bool {
	require.NotNil(t, vmConfig)
	require.NotNil(t, inputs)
	// Server-mode flags (--server) satisfy the range-executor's flattened SingleChainHost parser;
	// the executor drives the preimage server itself rather than running a client program.
	args, err := vm.NewKonaExecutor().OracleCommand(*vmConfig, dir, *inputs)
	require.NoError(t, err, "build kona-sp1 oracle command")
	args = append(args, extraArgs...)
	return runOracleServer(t, logger, "kona-sp1-range-executor", args, dir)
}

// RunKonaSP1SuperRange runs the kona-sp1 super-range executor. Returns false if the guest
// rejects the synthesized super-range claim and true otherwise.
func RunKonaSP1SuperRange(t require.TestingT, logger log.Logger, executorPath string, dir string, args ...string) bool {
	require.NotEmpty(t, executorPath)
	argv := append([]string{executorPath}, args...)
	return runOracleServer(t, logger, "kona-sp1-super-range-executor", argv, dir)
}

// runOracleServer executes a kona oracle-server binary and interprets its exit code: exit 1 means
// the program rejected the claim (returns false), exit 0 means it accepted (returns true), and any
// other failure fails the test.
func runOracleServer(t require.TestingT, logger log.Logger, component string, args []string, dir string) bool {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "KONA_LOG_STDOUT_FORMAT=json", "NO_COLOR=1")

	logOut := logpipe.ToLoggerWithMinLevel(logger.New("component", component, "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(logger.New("component", component, "src", "stderr"), log.LevelWarn)
	cmd.Stdout = logpipe.NewLineBuffer(logpipe.LogCallback(func(line []byte) {
		logOut(logpipe.ParseRustStructuredLogs(line))
	}))
	cmd.Stderr = logpipe.NewLineBuffer(logpipe.LogCallback(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	}))

	var exitErr *exec.ExitError
	runErr := cmd.Run()
	if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
		return false
	}
	require.NoError(t, runErr)
	return true
}
