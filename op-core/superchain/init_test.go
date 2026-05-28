package superchain

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	gethsuperchain "github.com/ethereum/go-ethereum/superchain"
)

// TestEmbeddedSuperchainCommitMatchesOpGeth asserts that the SR commit
// embedded by op-core/superchain matches the one bundled by op-geth. Until
// op-geth's internal superchain-registry copy is removed (#20257), the two
// must agree: a node using one version's SR config with another version's L2
// chain state would produce wrong consensus results.
//
// The canonical pin lives in the packages/contracts-bedrock/lib/superchain-registry
// submodule; this test fails when the submodule (and the rebuilt zip) drift
// from op-geth's pin without the go.mod replace being bumped to match.
func TestEmbeddedSuperchainCommitMatchesOpGeth(t *testing.T) {
	zr, err := zip.NewReader(bytes.NewReader(builtInConfigData), int64(len(builtInConfigData)))
	if err != nil {
		t.Fatalf("opening embedded superchain-configs.zip: %v", err)
	}
	f, err := zr.Open("COMMIT")
	if err != nil {
		t.Fatalf("reading COMMIT entry from embedded zip: %v", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("reading COMMIT entry contents: %v", err)
	}
	zipCommit := strings.TrimSpace(string(raw))

	if gethCommit := gethsuperchain.EmbeddedRegistryCommit(); gethCommit != zipCommit {
		t.Fatalf(
			"op-core/superchain's embedded zip is at commit %s but op-geth bundles commit %s.\n"+
				"Bump the submodule at packages/contracts-bedrock/lib/superchain-registry and run\n"+
				"`just sync-superchain`, or bump the op-geth replace directive in go.mod —\n"+
				"whichever is the stale side.",
			zipCommit, gethCommit)
	}
}
