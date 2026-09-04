package rollup

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestNoSuperchainImport keeps op-node/rollup decoupled from op-core/superchain.
// That package embeds the multi-megabyte superchain config bundle, so anything
// in its build closure must generate the bundle before it compiles. op-node/rollup
// is imported by dozens of packages that only need its types, so it must stay
// bundle-free. Config loading that reads the registry lives in op-node/superchain.
//
// The check is transitive: a direct-import check misses the bundle sneaking in
// through an intermediate package.
func TestNoSuperchainImport(t *testing.T) {
	depguard.RequireNoTransitiveImport(t, "./...",
		"github.com/ethereum-optimism/optimism/op-core/superchain")
}

// TestNoTestutilsImport keeps op-node/rollup out of op-service/testutils.
// testutils is a single package whose closure reaches op-service/apis and the
// libp2p stack, so a production file importing it for one random-data helper
// links that whole tree into every downstream binary. Test-only helpers that
// need it belong in a test package such as op-node/rollup/derive/test.
func TestNoTestutilsImport(t *testing.T) {
	depguard.RequireNoTransitiveImportExcept(t, "./...",
		[]string{
			// A test-helper package, imported only by tests and benchmarks.
			"github.com/ethereum-optimism/optimism/op-node/rollup/derive/test",
		},
		"github.com/ethereum-optimism/optimism/op-service/testutils")
}
