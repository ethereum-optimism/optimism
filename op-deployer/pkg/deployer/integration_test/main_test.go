package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	op_service "github.com/ethereum-optimism/optimism/op-service"
)

// TestMain provisions the op-script-engine binary once for the whole package. The Rust engine is
// op-deployer's default script engine (env.DefaultScriptEngine), so every genesis-generating test in
// this package drives it and the binary must be resolvable. CI supplies it via
// RUST_BINARY_PATH_OP_SCRIPT_ENGINE (the persisted rust-workspace release binary); locally the binary
// is cargo-built once when that env is unset and cargo is available, then the env is pointed at it. A
// machine with neither a pre-built binary nor cargo lets the genesis tests fail loudly at provisioning
// rather than silently skip.
func TestMain(m *testing.M) {
	provisionRustScriptEngine()
	os.Exit(m.Run())
}

func provisionRustScriptEngine() {
	if _, ok := rustengine.PrebuiltEngineBinary(); ok {
		return
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	root, err := op_service.FindMonorepoRoot(wd)
	if err != nil {
		return
	}
	rustDir := filepath.Join(root, "rust")
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = rustDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to build op-script-engine for integration tests: %v", err))
	}
	if err := os.Setenv(rustengine.EngineBinaryPathEnv, filepath.Join(rustDir, "target", "debug", "op-script-engine")); err != nil {
		panic(fmt.Sprintf("failed to set %s: %v", rustengine.EngineBinaryPathEnv, err))
	}
}
