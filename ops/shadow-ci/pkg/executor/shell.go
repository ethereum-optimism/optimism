package executor

import (
	"os"
	"os/exec"
)

// ShellRunner executes commands via bash, writing output to a log file.
type ShellRunner struct{}

// Run executes a command via bash -c, capturing stdout/stderr to logPath.
func (s *ShellRunner) Run(ctx RunContext) error {
	logFile, err := os.Create(ctx.LogPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command("bash", "-c", ctx.Command)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = os.Environ()
	return cmd.Run()
}
