package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func TestBuildRethUnwindCmd_Args(t *testing.T) {
	// Use the test binary itself as the "reth binary" so LookPath succeeds.
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := buildRethUnwindCmd(context.Background(), self, "/data/reth", "op-mainnet", 12345678)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// cmd.Args[0] is the resolved binary path, followed by the arguments.
	wantArgs := []string{
		self,
		"stage", "unwind",
		"--datadir", "/data/reth",
		"--chain", "op-mainnet",
		"to-block", "12345678",
	}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(cmd.Args), len(wantArgs), cmd.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if cmd.Args[i] != want {
			t.Errorf("arg[%d] = %q, want %q", i, cmd.Args[i], want)
		}
	}
}

func TestBuildRethUnwindCmd_BinaryNotFound(t *testing.T) {
	_, err := buildRethUnwindCmd(context.Background(), "/nonexistent/reth", "/data", "optimism", 100)
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
}

func TestRethRewind_SubprocessExit(t *testing.T) {
	// TestHelperProcess pattern: re-invoke the test binary as a fake "reth" process.
	if os.Getenv("GO_TEST_HELPER_PROCESS") == "1" {
		code, _ := strconv.Atoi(os.Getenv("GO_TEST_EXIT_CODE"))
		os.Exit(code)
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	lgr := log.NewLogger(log.DiscardHandler())

	t.Run("success", func(t *testing.T) {
		// Override the command builder by calling RethRewind with the test binary.
		// The helper process will exit 0.
		err := rethRewindWithHelper(context.Background(), lgr, self, 0, 100)
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		err := rethRewindWithHelper(context.Background(), lgr, self, 1, 100)
		if err == nil {
			t.Fatal("expected error for exit code 1, got nil")
		}
	})
}

// rethRewindWithHelper runs a subprocess using the test binary as a fake reth,
// configured to exit with the given code.
func rethRewindWithHelper(ctx context.Context, lgr log.Logger, testBinary string, exitCode int, toBlock uint64) error {
	cmd := exec.CommandContext(ctx, testBinary,
		"-test.run=TestRethRewind_SubprocessExit",
	)
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		fmt.Sprintf("GO_TEST_EXIT_CODE=%d", exitCode),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reth stage unwind failed with exit code %d: %w", exitCode, err)
	}

	lgr.Info("Successfully rewound reth to block", "toBlock", toBlock)
	return nil
}
