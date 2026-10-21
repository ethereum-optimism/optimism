package opgeth

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// opgethAllowed lists the packages permitted to reach this package, as exact package
// paths or "prefix/..." subtrees.
var opgethAllowed = []string{
	// This package.
	"github.com/ethereum-optimism/optimism/op-core/opgeth",

	// Test and devnet infrastructure that runs op-geth in-process as an L2 execution
	// client, and the suites built on it.
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/...",
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/cli",
	"github.com/ethereum-optimism/optimism/op-devstack/...",
	"github.com/ethereum-optimism/optimism/op-e2e/...",
	"github.com/ethereum-optimism/optimism/op-up",
	"github.com/ethereum-optimism/optimism/rust/kona/tests/...",
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/...",
}

// TestReachability keeps op-geth's OP Stack fields out of every package that does not
// run op-geth in-process. This package builds only against op-geth, so a package that
// reaches it can no longer be built by a downstream module against upstream
// go-ethereum.
func TestReachability(t *testing.T) {
	depguard.RequireNoTransitiveImportExcept(t,
		"github.com/ethereum-optimism/optimism/...", opgethAllowed,
		"github.com/ethereum-optimism/optimism/op-core/opgeth")
}
