package forge

import (
	"bufio"
	"bytes"
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

// anvilStartTimeout bounds the time until anvil serves its RPC.
const anvilStartTimeout = time.Minute

var anvilListeningRegexp = regexp.MustCompile(`Listening on (\S+)`)

// Fork is an ephemeral anvil fork. Every account is unlocked, so forge can send with --unlocked.
type Fork struct {
	cmd         *exec.Cmd
	rpcURL      string
	foundryHome string
}

// NewAnvilBinary returns the standard version of anvil.
func NewAnvilBinary(opts ...StandardBinOpt) (*StandardBin, error) {
	return NewStandardBinary(append([]StandardBinOpt{WithTool("anvil")}, opts...)...)
}

// StartFork starts anvil on a free loopback port as a fork of rpcURL at blockNumber.
func StartFork(ctx context.Context, anvil Binary, rpcURL string, blockNumber uint64) (*Fork, error) {
	if err := anvil.Ensure(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure anvil: %w", err)
	}
	// anvil keeps its state in FOUNDRY_HOME, so each fork gets its own.
	foundryHome, err := os.MkdirTemp("", "op-deployer-anvil-")
	if err != nil {
		return nil, fmt.Errorf("failed to create anvil home: %w", err)
	}
	// Any account can send on the fork, so it listens on loopback only, without CORS.
	cmd := exec.Command(anvil.Path(),
		"--fork-url", rpcURL,
		"--fork-block-number", strconv.FormatUint(blockNumber, 10),
		"--host", "127.0.0.1",
		"--port", "0",
		"--no-cors",
		"--auto-impersonate",
		"--no-storage-caching",
	)
	cmd.Env = append(inheritedEnv(), "FOUNDRY_HOME="+foundryHome)
	stderr := new(tailBuffer)
	cmd.Stderr = stderr
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
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if m := anvilListeningRegexp.FindStringSubmatch(scanner.Text()); m != nil {
				listening <- m[1]
				break
			}
		}
		// Drain stdout, or anvil blocks on a full pipe.
		_, _ = io.Copy(io.Discard, stdout)
	}()

	timer := time.NewTimer(anvilStartTimeout)
	defer timer.Stop()
	select {
	case addr := <-listening:
		fork.rpcURL = "http://" + addr
		return fork, nil
	case <-exited:
		// Stop waits for anvil, so stderr is complete after it.
		_ = fork.Stop()
		return nil, fmt.Errorf("anvil exited before it forked the chain: %s", bytes.TrimSpace(stderr.buf))
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

// anvilStderrLimit bounds the anvil stderr kept for error messages.
const anvilStderrLimit = 4096

// tailBuffer keeps the last anvilStderrLimit bytes written to it.
type tailBuffer struct {
	buf []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if over := len(b.buf) - anvilStderrLimit; over > 0 {
		b.buf = b.buf[over:]
	}
	return len(p), nil
}
