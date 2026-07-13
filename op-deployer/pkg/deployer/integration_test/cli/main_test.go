package cli

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// TestMain provisions the op-script-engine binary for the whole package: the CLI apply tests run
// the genesis stage on the default (Rust) script engine, and the CLI runner chdirs into temp work
// dirs where rustbin's monorepo discovery cannot resolve the binary. CI supplies it via
// RUST_BINARY_PATH_OP_SCRIPT_ENGINE; locally it is cargo-built once.
func TestMain(m *testing.M) {
	rustengine.ProvisionTestBinary()
	os.Exit(m.Run())
}
