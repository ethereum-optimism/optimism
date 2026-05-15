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

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "KONA_LOG_STDOUT_FORMAT=json", "NO_COLOR=1")

	logOut := logpipe.ToLoggerWithMinLevel(logger.New("component", "kona-host", "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(logger.New("component", "kona-host", "src", "stderr"), log.LevelWarn)
	cmd.Stdout = logpipe.NewLineBuffer(logpipe.LogCallback(func(line []byte) {
		logOut(logpipe.ParseRustStructuredLogs(line))
	}))
	cmd.Stderr = logpipe.NewLineBuffer(logpipe.LogCallback(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	}))

	// Distinguish the three legitimate outcomes from an unhandled crash:
	//   nil               -> exit 0: kona accepted the claim (valid).
	//   ExitError code 1  -> exit 1: kona rejected the claim (invalid).
	//   anything else     -> unhandled crash (panic, abort, IO error). The previous
	//                        version of this wrapper silently returned true for these,
	//                        which let HonestClaim tests false-positive when kona-host
	//                        crashed early. Fail loudly so the test surface reflects
	//                        reality.
	runErr := cmd.Run()
	if runErr == nil {
		return true
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return false
		}
		require.FailNowf(t, "kona-host exited with unexpected status",
			"exit code %d (expected 0 for valid claim or 1 for invalid claim); see logs above for stderr/stdout",
			exitErr.ExitCode())
	}
	require.NoErrorf(t, runErr, "kona-host failed to run (non-exit error); see logs above")
	return true // unreachable; FailNow / NoError abort the test
}
