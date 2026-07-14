package integration_test

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// TestMain provisions the op-script-engine binary once for the whole package. The Rust engine is
// op-deployer's only script engine, so every genesis-generating test in this package drives it and
// the binary must be resolvable. CI supplies it via
// RUST_BINARY_PATH_OP_SCRIPT_ENGINE (the persisted rust-workspace release binary); locally the binary
// is cargo-built once when that env is unset and cargo is available, then the env is pointed at it. A
// machine with neither a pre-built binary nor cargo lets the genesis tests fail loudly at provisioning
// rather than silently skip.
func TestMain(m *testing.M) {
	rustengine.ProvisionTestBinary()
	os.Exit(m.Run())
}
