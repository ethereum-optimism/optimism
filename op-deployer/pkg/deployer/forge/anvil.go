package forge

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// anvilStartTimeout bounds the time that anvil takes to fork the chain and serve its RPC.
const anvilStartTimeout = time.Minute

var anvilListeningRegexp = regexp.MustCompile(`Listening on (\S+)`)

// Fork is an ephemeral anvil fork of a chain. Every account is unlocked, so that a forge script can
// send from an account with --unlocked, without its private key.
type Fork struct {
	cmd         *exec.Cmd
	rpcURL      string
	foundryHome string
}

// NewAnvilBinary returns the standard version of anvil, which the forge release archive includes.
func NewAnvilBinary(opts ...StandardBinOpt) (*StandardBin, error) {
	return NewStandardBinary(append([]StandardBinOpt{WithTool("anvil")}, opts...)...)
}

// StartFork starts anvil on a free local port, as a fork of the chain at rpcURL at blockNumber.
func StartFork(ctx context.Context, anvil Binary, rpcURL string, blockNumber uint64) (*Fork, error) {
	if err := anvil.Ensure(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure anvil: %w", err)
	}
	// anvil keeps its state in FOUNDRY_HOME, so each fork gets its own.
	foundryHome, err := os.MkdirTemp("", "op-deployer-anvil-")
	if err != nil {
		return nil, fmt.Errorf("failed to create anvil home: %w", err)
	}
	cmd := exec.Command(anvil.Path(),
		"--fork-url", rpcURL,
		"--fork-block-number", strconv.FormatUint(blockNumber, 10),
		"--port", "0",
		"--auto-impersonate",
		"--no-storage-caching",
	)
	cmd.Env = append(os.Environ(), "FOUNDRY_HOME="+foundryHome)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(foundryHome)
		return nil, fmt.Errorf("failed to read anvil output: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(foundryHome)
		return nil, fmt.Errorf("failed to start anvil: %w", err)
	}
	fork := &Fork{cmd: cmd, foundryHome: foundryHome}

	listening := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if m := anvilListeningRegexp.FindStringSubmatch(scanner.Text()); m != nil {
				listening <- m[1]
				break
			}
		}
		// Keep reading, so that anvil never blocks on a full pipe.
		_, _ = io.Copy(io.Discard, stdout)
	}()

	timer := time.NewTimer(anvilStartTimeout)
	defer timer.Stop()
	select {
	case addr := <-listening:
		fork.rpcURL = "http://" + addr
		return fork, nil
	case <-timer.C:
		_ = fork.Stop()
		return nil, errors.New("anvil did not fork the chain in time")
	case <-ctx.Done():
		_ = fork.Stop()
		return nil, ctx.Err()
	}
}

// RPCURL returns the RPC URL of the fork.
func (f *Fork) RPCURL() string {
	return f.rpcURL
}

// Stop stops anvil and removes its state.
func (f *Fork) Stop() error {
	defer os.RemoveAll(f.foundryHome)
	if err := f.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("failed to stop anvil: %w", err)
	}
	_ = f.cmd.Wait()
	return nil
}
