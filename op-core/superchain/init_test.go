package superchain

import "testing"

// TestSyncSuperchain asserts that the embedded bundle's COMMIT entry agrees
// with op-geth's embedded copy of the superchain-registry. The init() function
// in init.go panics on mismatch (so any importer fails fast at process start);
// this test runs the same check via VerifyEmbeddedCommit so a mismatch surfaces
// as a clean test failure rather than a panic during package import.
func TestSyncSuperchain(t *testing.T) {
	if err := VerifyEmbeddedCommit(); err != nil {
		t.Fatal(err)
	}
}
