package rollup

import (
	"go/build"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNoSuperchainImport keeps op-node/rollup decoupled from op-core/superchain.
// That package embeds the multi-megabyte superchain config bundle, so anything
// in its build closure must generate the bundle before it compiles. op-node/rollup
// is imported by dozens of packages that only need its types, so it must stay
// bundle-free. Config loading that reads the registry lives in op-node/superchain.
func TestNoSuperchainImport(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)

	const forbidden = "github.com/ethereum-optimism/optimism/op-core/superchain"
	imports := append([]string{}, pkg.Imports...)
	imports = append(imports, pkg.TestImports...)
	imports = append(imports, pkg.XTestImports...)
	require.NotContains(t, imports, forbidden,
		"op-node/rollup must not import op-core/superchain; registry-backed config loading belongs in op-node/superchain")
}
