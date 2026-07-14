package fetch

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
)

// TestMain provisions the op-script-engine binary once for the whole package: FetchChainInfo runs
// on the Rust engine's fork mode by default.
func TestMain(m *testing.M) {
	rustengine.ProvisionTestBinary()
	os.Exit(m.Run())
}
