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
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := buildRethUnwindCmd(context.Background(), self, "/data/reth", "op-mainnet", 12345678)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{
		self,
		"stage", "unwind",
		"--datadir", "/data/reth",
		"--chain", "op-mainnet",
		"to-block", "12345678",
	}
	assertArgs(t, cmd.Args, wantArgs)
}

func TestBuildRethUnwindCmd_BinaryNotFound(t *testing.T) {
	_, err := buildRethUnwindCmd(context.Background(), "/nonexistent/reth", "/data", "optimism", 100)
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
}

func TestBuildRethDBCmd_State(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := buildRethDBCmd(context.Background(), self, "/db", "optimism",
		"state", "0xdead", "--format", "json", "--limit", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{
		self,
		"db", "--datadir", "/db", "--chain", "optimism",
		"state", "0xdead", "--format", "json", "--limit", "100",
	}
	assertArgs(t, cmd.Args, wantArgs)
}

func TestBuildRethDBCmd_StageCheckpoints(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := buildRethDBCmd(context.Background(), self, "/db", "dev",
		"stage-checkpoints", "get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{
		self,
		"db", "--datadir", "/db", "--chain", "dev",
		"stage-checkpoints", "get",
	}
	assertArgs(t, cmd.Args, wantArgs)
}

func TestRethRewind_SubprocessExit(t *testing.T) {
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
		err := rethCmdWithHelper(context.Background(), lgr, self, 0, "test")
		if err != nil {
			t.Fatalf("expected success, got: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		err := rethCmdWithHelper(context.Background(), lgr, self, 1, "test")
		if err == nil {
			t.Fatal("expected error for exit code 1, got nil")
		}
	})
}

// rethCmdWithHelper runs a subprocess using the test binary as a fake reth,
// configured to exit with the given code.
func rethCmdWithHelper(ctx context.Context, lgr log.Logger, testBinary string, exitCode int, label string) error {
	cmd := exec.CommandContext(ctx, testBinary,
		"-test.run=TestRethRewind_SubprocessExit",
	)
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		fmt.Sprintf("GO_TEST_EXIT_CODE=%d", exitCode),
	)
	return runRethCmd(lgr, cmd, label)
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], w)
		}
	}
}
