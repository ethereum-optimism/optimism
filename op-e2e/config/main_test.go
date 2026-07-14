package config

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// TestMain provisions the op-script-engine binary so op-e2e/config tests that build L2 genesis
// through the default (Rust) engine — the genesis smoke test and the L1/L2 allocs helpers — run
// locally without a pre-supplied binary. CI sets RUST_BINARY_PATH_OP_SCRIPT_ENGINE, which
// ProvisionTestBinary honors; otherwise it cargo-builds once when cargo is available.
func TestMain(m *testing.M) {
	rustengine.ProvisionTestBinary()
	os.Exit(m.Run())
}
