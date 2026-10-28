package rustbin

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/stretchr/testify/require"
)

// konaHostTimeout bounds a kona-host run. The host retries a preimage it cannot fetch
// indefinitely, so one its L1 or L2 node cannot serve would otherwise hang the test until the go
// test timeout.
const konaHostTimeout = 10 * time.Minute

// RunKonaNative runs kona-host in --native mode. Returns false if kona disagrees and true
// otherwise.
func RunKonaNative(t require.TestingT, logger log.Logger, vmConfig *vm.Config, dir string, inputs *utils.LocalGameInputs) bool {
	require.NotNil(t, vmConfig)
	require.NotNil(t, inputs)
	args, err := vm.NewNativeKonaExecutor().OracleCommand(*vmConfig, dir, *inputs)
	require.NoError(t, err, "build kona oracle command")
	return runOracleServer(t, logger, "kona-host", args, dir, konaHostTimeout)
}

// RunKonaSuperNative runs kona-host in `super --native` mode, i.e. the interop client program over
// a super-root state transition. Returns false if kona disagrees and true otherwise.
func RunKonaSuperNative(t require.TestingT, logger log.Logger, vmConfig *vm.Config, dir string, inputs *utils.LocalGameInputs) bool {
	require.NotNil(t, vmConfig)
	require.NotNil(t, inputs)
	args, err := vm.NewNativeKonaSuperExecutor().OracleCommand(*vmConfig, dir, *inputs)
	require.NoError(t, err, "build kona super oracle command")
	return runOracleServer(t, logger, "kona-host-super", args, dir, konaHostTimeout)
}

// RunKonaSP1SuperRange runs the kona-sp1 super-range executor. Returns false if the guest
// rejects the synthesized super-range claim and true otherwise.
func RunKonaSP1SuperRange(t require.TestingT, logger log.Logger, executorPath string, dir string, args ...string) bool {
	require.NotEmpty(t, executorPath)
	argv := append([]string{executorPath}, args...)
	return runOracleServer(t, logger, "kona-sp1-super-range-executor", argv, dir, 0)
}

// runOracleServer executes a kona oracle-server binary and interprets its exit code: exit 1 means
// the program rejected the claim (returns false), exit 0 means it accepted (returns true), and any
// other failure fails the test. A non-zero timeout bounds the run; a run still going when it
// expires fails the test.
func runOracleServer(t require.TestingT, logger log.Logger, component string, args []string, dir string, timeout time.Duration) bool {
	exePath, err := exec.LookPath(args[0])
	require.NoError(t, err, "resolve %s executable", component)
	exePath, err = filepath.Abs(exePath)
	require.NoError(t, err, "make %s executable path absolute", component)

	contextualT, ok := t.(interface{ Ctx() context.Context })
	require.True(t, ok, "%s test handle must provide a context", component)
	if !ok {
		return false
	}
	ctx := contextualT.Ctx()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, exePath, args[1:]...)
	cmd.Dir = dir
	// Bounds Wait after a kill: a grandchild holding the output pipes would otherwise block it.
	cmd.WaitDelay = 10 * time.Second
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
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		require.FailNowf(t, "timeout", "%s did not finish within %s", component, timeout)
	}
	require.NoError(t, ctx.Err(), "%s run cancelled", component)
	if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
		return false
	}
	require.NoError(t, runErr)
	return true
}
