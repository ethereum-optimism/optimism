package interopgen

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// provisionEngineBinary makes the op-script-engine binary resolvable for Deploy:
// prefer a CI-provided pre-built binary (Go executors without cargo), else cargo-build the
// debug binary. Under REQUIRE_RUST_ENGINE (CI) a machine with neither fails the gate loudly;
// local dev without cargo skips. It was relocated here from the deleted rust_engine_parity_test.go
// so the interopgen golden gate keeps compiling.
func provisionEngineBinary(t *testing.T) {
	if _, ok := rustengine.PrebuiltEngineBinary(); ok {
		return
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		msg := fmt.Sprintf("no pre-built op-script-engine (%s unset) and cargo unavailable", rustengine.EngineBinaryPathEnv)
		if rustengine.RequireEngine() {
			t.Fatal(msg + " (" + rustengine.RequireEngineEnv + " is set)")
		}
		t.Skip(msg + "; skipping Rust engine golden test")
	}
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = "../../rust"
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "cargo build -p op-script-engine failed:\n%s", out)
	bin, err := filepath.Abs("../../rust/target/debug/op-script-engine")
	require.NoError(t, err)
	t.Setenv(rustengine.EngineBinaryPathEnv, bin)
}
