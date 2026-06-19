package superchain

import "testing"

// TestSyncSuperchain runs VerifyEmbeddedCommit: it asserts the embedded zip's
// SHA256 matches the committed .sha256 and that op-geth bundles the same
// superchain-registry commit as the zip's COMMIT entry. The init() function in
// init.go panics on mismatch (so any importer fails fast at process start); this
// test surfaces the same check as a clean test failure rather than a panic during
// package import.
func TestSyncSuperchain(t *testing.T) {
	if err := VerifyEmbeddedCommit(); err != nil {
		t.Fatal(err)
	}
}
