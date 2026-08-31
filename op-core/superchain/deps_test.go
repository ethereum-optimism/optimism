package superchain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// bundleAllowed lists the packages permitted to reach op-core/superchain,
// as exact package paths or "prefix/..." subtrees. Everything else in the
// monorepo is guarded, so a new package is covered automatically rather than
// having to be remembered.
//
// Each entry names code that cannot be imported by a downstream Go module
// (see the test doc below), so adding one is a real cost, not a formality.
// Prefix entries deliberately trade granularity for maintainability: they mark
// a whole component as a registry consumer.
var bundleAllowed = []string{
	// This package.
	"github.com/ethereum-optimism/optimism/op-core/superchain",

	// Registry loaders (op-node/superchain, op-node/chaincfg) and the node
	// wiring built on them. Enumerated exactly rather than as op-node/...
	// so that op-node/rollup/... — the type surface downstream modules import
	// most — stays guarded (op-node/rollup's own TestNoSuperchainImport pins
	// that invariant where a regression would be introduced).
	"github.com/ethereum-optimism/optimism/op-node",
	"github.com/ethereum-optimism/optimism/op-node/chaincfg",
	"github.com/ethereum-optimism/optimism/op-node/cmd/...",
	"github.com/ethereum-optimism/optimism/op-node/config",
	"github.com/ethereum-optimism/optimism/op-node/flags",
	"github.com/ethereum-optimism/optimism/op-node/node",
	"github.com/ethereum-optimism/optimism/op-node/p2p/cli",
	"github.com/ethereum-optimism/optimism/op-node/superchain",

	// Runs registry chains selected by name (--network, VM executor), and
	// resolves dispute contract addresses from registry data.
	"github.com/ethereum-optimism/optimism/op-challenger/...",
	"github.com/ethereum-optimism/optimism/op-dispute-mon/...",

	// Resolves --network via op-node's rollup-config loading.
	"github.com/ethereum-optimism/optimism/op-conductor/...",

	// Reads superchain and chain records for standard deployments.
	"github.com/ethereum-optimism/optimism/op-deployer/...",

	// Embeds op-node, which loads registry configs.
	"github.com/ethereum-optimism/optimism/op-supernode/...",

	// Loads rollup configs for registry networks by name.
	"github.com/ethereum-optimism/optimism/op-interop-filter/...",

	// Builds registry bundles for arbitrary registry commits.
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/...",

	// Deploys via op-deployer, which reads registry records.
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen",

	// Chain-selection flags: the --network help text lists the available
	// networks, so the package is inherently about the registry.
	"github.com/ethereum-optimism/optimism/op-service/flags",

	// Uses chaincfg's registry-derived chain-ID→name map in its startup log
	// banner only; its rollup config comes from the rollup node RPC. Slated to
	// log the bare chain ID instead, which would remove these entries (#22678).
	"github.com/ethereum-optimism/optimism/op-batcher/batcher",
	"github.com/ethereum-optimism/optimism/op-batcher/cmd",

	// Generates flag documentation for the registry consumers above.
	"github.com/ethereum-optimism/optimism/docs/public-docs/scripts/gen-flags",

	// Test and devnet infrastructure that stands up registry consumers.
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/...",
	"github.com/ethereum-optimism/optimism/op-devstack/...",
	"github.com/ethereum-optimism/optimism/op-e2e/...",
	"github.com/ethereum-optimism/optimism/op-up",
	"github.com/ethereum-optimism/optimism/ops/scripts/prefork-state-dump",
	"github.com/ethereum-optimism/optimism/rust/kona/tests/...",
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/...",
}

// TestBundleReachability keeps every package outside bundleAllowed importable
// by downstream Go modules.
//
// op-core/superchain embeds superchain-configs.zip, which sync-superchain.sh
// generates and .gitignore excludes. The zip is therefore absent from the
// published module, and any package whose build closure reaches it simply does
// not compile outside this repo:
//
//	op-core/superchain/chain.go:21:12: pattern superchain-configs.zip:
//	    no matching files found
//
// An accidental edge into this package silently makes a whole area of the
// module unusable downstream. That is how op-service/oppprof broke: it needed
// only the PathFlag value type but took it from op-service/flags, whose
// --network flag interpolates chaincfg.AvailableNetworks() into its usage
// string. It is also how the op-service client stack (apis, dial, sources) and
// the op-node derivation pipeline broke: op-core/interop/depset coupled the
// DependencySet types with the registry-backed loader that now lives in
// op-node/superchain.
//
// The guard is an allowlist over the whole monorepo rather than a list of
// guarded packages, so the invariant holds by default. It is checked in both
// directions: an entry in bundleAllowed that no longer reaches the bundle
// fails too, so the list cannot outlive the dependency that justified it.
//
// The walk is transitive. A direct-import check would miss most of the breaks
// above, which arrived through intermediate packages.
func TestBundleReachability(t *testing.T) {
	depguard.RequireNoTransitiveImportExcept(t,
		"github.com/ethereum-optimism/optimism/...", bundleAllowed,
		"github.com/ethereum-optimism/optimism/op-core/superchain")
}
