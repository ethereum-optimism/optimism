package superchain

import "testing"

// TestSyncSuperchain runs VerifyEmbeddedBundle: it asserts the embedded zip's
// SHA256 matches the committed .sha256. The init() function in init.go panics on
// mismatch (so any importer fails fast at process start); this test surfaces the
// same check as a clean test failure rather than a panic during package import.
func TestSyncSuperchain(t *testing.T) {
	if err := VerifyEmbeddedBundle(); err != nil {
		t.Fatal(err)
	}
}
