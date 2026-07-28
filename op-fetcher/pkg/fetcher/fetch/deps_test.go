package fetch

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestNoSuperchainImport keeps op-fetcher's fetch package free of
// op-core/superchain in its whole build closure. That package embeds the
// generated, gitignored superchain config bundle, which is absent from a clean
// module download — and the superchain-registry's ops tooling consumes this
// package as a Go module dependency, so any path to the bundle breaks its build.
func TestNoSuperchainImport(t *testing.T) {
	depguard.RequireNoTransitiveImport(t, ".",
		"github.com/ethereum-optimism/optimism/op-core/superchain")
}
