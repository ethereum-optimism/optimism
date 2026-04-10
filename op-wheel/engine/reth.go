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

	lgr.Info("Executing reth stage unwind", "binary", cmd.Path, "args", cmd.Args[1:], "toBlock", toBlock)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("reth stage unwind failed with exit code %d: %w", exitErr.ExitCode(), err)
		}
		return fmt.Errorf("failed to execute reth: %w", err)
	}

	lgr.Info("Successfully rewound reth to block", "toBlock", toBlock)
	return nil
}

// buildRethUnwindCmd constructs the exec.Cmd for `reth stage unwind to-block <N>`.
// Validates that the binary exists before returning.
func buildRethUnwindCmd(ctx context.Context, rethBinary string, datadir string, chain string, toBlock uint64) (*exec.Cmd, error) {
	resolvedPath, err := exec.LookPath(rethBinary)
	if err != nil {
		return nil, fmt.Errorf("reth binary not found at %q: %w", rethBinary, err)
	}

	args := []string{
		"stage", "unwind",
		"--datadir", datadir,
		"--chain", chain,
		"to-block", strconv.FormatUint(toBlock, 10),
	}

	return exec.CommandContext(ctx, resolvedPath, args...), nil
}
