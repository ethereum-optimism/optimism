package bundle

import "testing"

// TestSyncSuperchain runs verifyEmbeddedBundle: it asserts the embedded zip's
// SHA256 matches the committed .sha256. The package's init panics on
// mismatch (so any importer fails fast at process start); this test surfaces the
// same check as a clean test failure rather than a panic during package import.
func TestSyncSuperchain(t *testing.T) {
	if err := verifyEmbeddedBundle(); err != nil {
		t.Fatal(err)
	}
}
