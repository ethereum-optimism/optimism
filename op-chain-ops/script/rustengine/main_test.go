package rustengine

import (
	"os"
	"testing"
)

// TestMain provisions the op-script-engine binary once for the whole package so the byte-parity
// gates are guaranteed to run rather than skip on CI convention: it honors a pre-built binary
// (RUST_BINARY_PATH_OP_SCRIPT_ENGINE, how CI supplies it), else cargo-builds it when cargo is
// available. A machine with neither leaves the binary unresolved, and buildEngine then skips (or
// fails loudly under REQUIRE_RUST_ENGINE).
func TestMain(m *testing.M) {
	ProvisionTestBinary()
	os.Exit(m.Run())
}
