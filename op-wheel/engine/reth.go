package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/ethereum/go-ethereum/log"
)

// RethRewind performs an offline rewind of a reth node by executing
// `reth stage unwind to-block <N>` as a subprocess.
// The reth node must be stopped before calling this function.
func RethRewind(ctx context.Context, lgr log.Logger, rethBinary string, datadir string, chain string, toBlock uint64) error {
	cmd, err := buildRethUnwindCmd(ctx, rethBinary, datadir, chain, toBlock)
	if err != nil {
		return err
	}
	return runRethCmd(lgr, cmd, "reth stage unwind")
}

// RethState runs `reth db state <address>` to inspect account state offline.
func RethState(ctx context.Context, lgr log.Logger, rethBinary string, datadir string, chain string, address string, block string, limit uint64) error {
	cmd, err := buildRethDBCmd(ctx, rethBinary, datadir, chain, "state", address, "--format", "json", "--limit", strconv.FormatUint(limit, 10))
	if err != nil {
		return err
	}
	if block != "" {
		cmd.Args = append(cmd.Args, "--block", block)
	}
	return runRethCmd(lgr, cmd, "reth db state")
}

// RethHead runs `reth db stage-checkpoints get` to show the current head (stage checkpoints).
func RethHead(ctx context.Context, lgr log.Logger, rethBinary string, datadir string, chain string) error {
	cmd, err := buildRethDBCmd(ctx, rethBinary, datadir, chain, "stage-checkpoints", "get")
	if err != nil {
		return err
	}
	return runRethCmd(lgr, cmd, "reth db stage-checkpoints")
}

// runRethCmd executes a reth command, streaming output and handling exit codes.
// Context cancellation is handled by cmd itself (built via exec.CommandContext).
func runRethCmd(lgr log.Logger, cmd *exec.Cmd, label string) error {
	lgr.Info("Executing "+label, "binary", cmd.Path, "args", cmd.Args[1:])

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%s failed with exit code %d: %w", label, exitErr.ExitCode(), err)
		}
		return fmt.Errorf("failed to execute reth: %w", err)
	}

	lgr.Info(label + " completed successfully")
	return nil
}

// resolveRethBinary validates the reth binary exists and returns its resolved path.
func resolveRethBinary(rethBinary string) (string, error) {
	resolvedPath, err := exec.LookPath(rethBinary)
	if err != nil {
		return "", fmt.Errorf("reth binary not found at %q: %w", rethBinary, err)
	}
	return resolvedPath, nil
}

// buildRethUnwindCmd constructs the exec.Cmd for `reth stage unwind to-block <N>`.
func buildRethUnwindCmd(ctx context.Context, rethBinary string, datadir string, chain string, toBlock uint64) (*exec.Cmd, error) {
	resolvedPath, err := resolveRethBinary(rethBinary)
	if err != nil {
		return nil, err
	}

	args := []string{
		"stage", "unwind",
		"--datadir", datadir,
		"--chain", chain,
		"to-block", strconv.FormatUint(toBlock, 10),
	}

	return exec.CommandContext(ctx, resolvedPath, args...), nil
}

// buildRethDBCmd constructs an exec.Cmd for `reth db --datadir <dir> --chain <chain> <subcommand> [args...]`.
func buildRethDBCmd(ctx context.Context, rethBinary string, datadir string, chain string, subArgs ...string) (*exec.Cmd, error) {
	resolvedPath, err := resolveRethBinary(rethBinary)
	if err != nil {
		return nil, err
	}

	// reth db --datadir <dir> --chain <chain> <subcommand> [args...]
	args := []string{"db", "--datadir", datadir, "--chain", chain}
	args = append(args, subArgs...)

	return exec.CommandContext(ctx, resolvedPath, args...), nil
}
