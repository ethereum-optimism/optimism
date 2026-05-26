package derive_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInteropActivationDeposits_GoRustDiff asserts the Go and Rust dump programs produce byte-identical stdout, catching cross-language drift in Interop activation deposits or upgrade-gas accounting.
func TestInteropActivationDeposits_GoRustDiff(t *testing.T) {
	repoRoot := repoRoot(t)

	goOut := runCmd(t, repoRoot,
		"go", "run", "./op-node/cmd/interop-deposits-dump",
	)

	rustOut := runCmd(t, filepath.Join(repoRoot, "rust", "kona"),
		"cargo", "run", "--quiet",
		"-p", "kona-hardforks",
		"--example", "interop-deposits-dump",
	)

	if !bytes.Equal(goOut, rustOut) {
		t.Fatalf("Go and Rust Interop activation dumps differ.\n--- Go (%d bytes) ---\n%s\n--- Rust (%d bytes) ---\n%s",
			len(goOut), goOut, len(rustOut), rustOut)
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v in %s failed: %v\nstderr:\n%s", name, args, dir, err, stderr.String())
	}
	return stdout.Bytes()
}

// repoRoot returns the optimism monorepo root, located by walking up from this test file
// until a directory containing both `go.mod` and a `rust/` subdirectory is found.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	dir := filepath.Dir(file)
	for {
		_, modErr := os.Stat(filepath.Join(dir, "go.mod"))
		_, rustErr := os.Stat(filepath.Join(dir, "rust"))
		if modErr == nil && rustErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root from %s", filepath.Dir(file))
		}
		dir = parent
	}
}
