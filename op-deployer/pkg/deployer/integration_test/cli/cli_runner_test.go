package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(runCLITests(m))
}

func runCLITests(m *testing.M) int {
	dir, err := os.MkdirTemp("", "op-deployer-cli-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// Each command needs its own process, but the suite only needs one binary.
	cliTestBinary = filepath.Join(dir, "op-deployer")
	cmd := exec.Command("go", "build", "-o", cliTestBinary, "../../../../cmd/op-deployer")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build op-deployer: %v\n%s", err, output)
		return 1
	}
	return m.Run()
}

func TestCLITestRunnerIsolation(t *testing.T) {
	const chainIDsEnv = "DEPLOYER_L2_CHAIN_IDS"
	t.Setenv(chainIDsEnv, "999")
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	const count = 8
	runners := make([]*CLITestRunner, count)
	for i := range runners {
		runners[i] = NewCLITestRunner(t)
		runners[i].workDir = t.TempDir()
	}

	type result struct {
		index  int
		output string
		err    error
	}
	results := make(chan result, count)
	start := make(chan struct{})
	for i, runner := range runners {
		go func() {
			<-start
			output, err := runner.Run(t.Context(), []string{
				"init", "--workdir", ".", "--l1-chain-id", "11155111",
			}, map[string]string{chainIDsEnv: strconv.Itoa(i + 1)})
			results <- result{index: i, output: output, err: err}
		}()
	}
	close(start)

	completed := make([]result, 0, count)
	for range count {
		completed = append(completed, <-results)
	}
	for _, result := range completed {
		require.NoError(t, result.err, "runner %d: %s", result.index, result.output)
		intent, err := pipeline.ReadIntent(runners[result.index].workDir)
		require.NoError(t, err)
		require.Len(t, intent.Chains, 1)
		require.Equal(t, common.HexToHash(strconv.FormatInt(int64(result.index+1), 16)), intent.Chains[0].ID)
	}

	currentDir, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, originalDir, currentDir)
	require.Equal(t, "999", os.Getenv(chainIDsEnv))
}

func TestCLITestRunnerCancellation(t *testing.T) {
	runner := NewCLITestRunner(t)
	script := filepath.Join(runner.workDir, "wait.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 60 &\necho $! > child.pid\nwait\n"), 0700))
	originalBinary := cliTestBinary
	cliTestBinary = script

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	t.Cleanup(func() {
		cancel()
		<-done
		cliTestBinary = originalBinary
	})
	go func() {
		defer close(done)
		_, err := runner.Run(ctx, nil, nil)
		done <- err
	}()

	var childPID int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(filepath.Join(runner.workDir, "child.pid"))
		if err != nil {
			return false
		}
		childPID, err = strconv.Atoi(strings.TrimSpace(string(data)))
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(10 * time.Second):
		// A surviving child retains stdout and prevents Run from returning.
		_ = syscall.Kill(childPID, syscall.SIGKILL)
		<-done
		t.Fatal("CLI cancellation left its child alive")
	}
}
