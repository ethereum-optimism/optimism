package standard

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestNoBundleImport keeps package standard free of op-core/superchain/bundle. The state package
// imports standard, and op-deployer's intent and state types are imported by downstream Go
// modules, which cannot compile the bundle: it embeds a generated superchain-configs.zip that is
// not part of the published module. Values read from the embedded registry belong in
// standard/registry.
func TestNoBundleImport(t *testing.T) {
	depguard.RequireNoTransitiveImport(t, ".",
		"github.com/ethereum-optimism/optimism/op-core/superchain/bundle")
}
