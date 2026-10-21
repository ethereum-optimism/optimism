package fetch

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestImportClosure keeps op-fetcher's library packages light, because the
// superchain-registry's ops tooling consumes them as a Go module dependency:
//   - op-core is off limits: op-core/superchain embeds the generated, gitignored superchain
//     config bundle, which is absent from a clean module download, and the OP types in
//     op-core still need op-geth.
//   - op-deployer and the op-chain-ops script host need op-geth, and a fetch needs no EVM of
//     its own.
func TestImportClosure(t *testing.T) {
	const repo = "github.com/ethereum-optimism/optimism"
	for _, pkgs := range []string{repo + "/op-fetcher/fetch/...", repo + "/op-fetcher/chaininfo/..."} {
		depguard.RequireNoTransitiveImportUnder(t, pkgs, repo+"/op-core")
		depguard.RequireNoTransitiveImportUnder(t, pkgs, repo+"/op-deployer")
		depguard.RequireNoTransitiveImport(t, pkgs,
			repo+"/op-chain-ops/script",
			repo+"/op-chain-ops/foundry",
		)
	}
}
