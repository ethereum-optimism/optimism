package interop

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// TestMain provisions the op-script-engine binary so the interop tests that build genesis through
// the default (Rust) engine can resolve it locally. CI sets RUST_BINARY_PATH_OP_SCRIPT_ENGINE,
// which ProvisionTestBinary honors; otherwise it cargo-builds once when cargo is available.
func TestMain(m *testing.M) {
	rustengine.ProvisionTestBinary()
	os.Exit(m.Run())
}
