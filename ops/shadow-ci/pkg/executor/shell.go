package executor

import (
	"os"
	"os/exec"
)

// ShellRunner executes commands via bash, writing output to a log file.
type ShellRunner struct{}

// Run executes a command via bash -c, capturing stdout/stderr to logPath.
func (s *ShellRunner) Run(category string, command string, logPath string) error {
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = os.Environ()
	return cmd.Run()
}
