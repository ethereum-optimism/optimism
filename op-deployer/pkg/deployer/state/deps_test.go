package state

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// TestNoBundleImport keeps the intent and state types importable by downstream Go modules.
// op-core/superchain/bundle embeds a generated superchain-configs.zip that is not part of the
// published module, so any package whose build closure reaches it does not compile outside this
// repository. Defaults read from the embedded registry belong in standard/registry and are passed
// in, as CombineDeployConfig's upgrade schedule is.
//
// The check is transitive: the edge this guards against arrived through package standard.
func TestNoBundleImport(t *testing.T) {
	depguard.RequireNoTransitiveImport(t, "./...",
		"github.com/ethereum-optimism/optimism/op-core/superchain/bundle")
}
