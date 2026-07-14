package interopgen

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// provisionEngineBinary makes the op-script-engine binary resolvable for DeployWithEngine:
// prefer a CI-provided pre-built binary (Go executors without cargo), else cargo-build the
// debug binary, else skip (a machine with neither cannot run the Rust engine). It was relocated
// here from the deleted rust_engine_parity_test.go so the interopgen golden gate keeps compiling.
func provisionEngineBinary(t *testing.T) {
	if _, ok := rustengine.PrebuiltEngineBinary(); ok {
		return
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skipf("no pre-built op-script-engine (%s unset) and cargo unavailable", rustengine.EngineBinaryPathEnv)
	}
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = "../../rust"
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cargo build -p op-script-engine failed:\n%s", out)
	bin, err := filepath.Abs("../../rust/target/debug/op-script-engine")
	require.NoError(t, err)
	t.Setenv(rustengine.EngineBinaryPathEnv, bin)
}
