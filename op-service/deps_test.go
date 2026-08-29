package op_service

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/depguard"
)

// bundleAllowed lists the op-service packages permitted to reach
// op-core/superchain. Everything else under op-service/... is guarded, so a new
// package is covered automatically rather than having to be remembered.
//
// Each entry is a package that cannot be imported by a downstream Go module.
// Adding one is a real cost, not a formality.
var bundleAllowed = []string{
	// Chain-selection flags (--network, --rollup.config, fork overrides). This
	// one is legitimate: the flag help text lists the available networks, so
	// the package is inherently about the registry.
	"github.com/ethereum-optimism/optimism/op-service/flags",

	// The rest all inherit the bundle from op-core/interop/depset, whose
	// FromRegistry loader lives alongside the DependencySet types. These
	// packages use only the types, so splitting depset the way PathFlag was
	// split here would clear the whole group. Tracked by #22678.
	"github.com/ethereum-optimism/optimism/op-service/apis",
	"github.com/ethereum-optimism/optimism/op-service/dial",
	"github.com/ethereum-optimism/optimism/op-service/sources",
	"github.com/ethereum-optimism/optimism/op-service/testutils",
	"github.com/ethereum-optimism/optimism/op-service/txintent",
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings",
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio",
}

// TestOpServiceIsImportableDownstream keeps op-service importable by downstream
// Go modules.
//
// op-core/superchain embeds superchain-configs.zip, which sync-superchain.sh
// generates and .gitignore excludes. The zip is therefore absent from the
// published module, and any package whose build closure reaches it simply does
// not compile outside this repo:
//
//	op-core/superchain/chain.go:21:12: pattern superchain-configs.zip:
//	    no matching files found
//
// op-service is the shared plumbing that services here and in other repos wire
// up, so an accidental edge into the registry silently makes a whole area of it
// unusable downstream. That is how op-service/oppprof broke: it needed only the
// PathFlag value type but took it from op-service/flags, whose --network flag
// interpolates chaincfg.AvailableNetworks() into its usage string. PathFlag now
// lives in the leaf package op-service/cliflags.
//
// The guard is an allowlist over ./... rather than a list of guarded packages,
// so the invariant holds by default. It is checked in both directions: an entry
// in bundleAllowed that no longer reaches the bundle fails too, so the list
// cannot outlive the dependency that justified it.
//
// The walk is transitive. A direct-import check would have missed the original
// break, which arrived through two intermediate packages.
func TestOpServiceIsImportableDownstream(t *testing.T) {
	depguard.RequireNoTransitiveImportExcept(t, "./...", bundleAllowed,
		"github.com/ethereum-optimism/optimism/op-core/superchain")
}
