package broadcaster

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestNoScriptHostImport keeps broadcasters independent of the Go script host, so they
// serve every script engine.
func TestNoScriptHostImport(t *testing.T) {
	depguard.RequireNoTransitiveImport(t, ".",
		"github.com/ethereum-optimism/optimism/op-chain-ops/script")
}
