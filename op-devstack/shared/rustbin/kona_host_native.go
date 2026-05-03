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

	var exitErr *exec.ExitError
	if runErr := cmd.Run(); errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 {
		return false
	}
	require.NoError(t, err)
	return true
}
