package rustengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenDir holds fixtures recorded from the in-process Go script host. Each fixture pins the exact
// output the Go host produced for a fixed input at the base commit; the golden tests replay the same
// input through the Rust engine and require the output to match. Once the Go host is deleted these
// fixtures are the only surviving record of its behavior, so there is nothing to regenerate from — a
// mismatch means the Rust engine diverged, not that the fixture is stale. See goldens/README.md for
// the exact recording provenance.
const goldenDir = "testdata/goldens"

// requireJSONMatchesGolden marshals got and compares it (canonically) to the committed golden fixture
// `name`. On mismatch it first writes the actual JSON to a temp file (logged) so a divergence is
// debuggable without the deleted Go host, then fails with a full JSON diff.
func requireJSONMatchesGolden(t *testing.T, name string, got any) {
	t.Helper()
	gotJSON, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)
	requireJSONBytesMatchesGolden(t, name, gotJSON)
}

// requireJSONBytesMatchesGolden compares an already-serialized JSON blob to the golden fixture. Used
// where the value is a raw json.RawMessage (e.g. the fork diff) and must not be re-marshaled.
func requireJSONBytesMatchesGolden(t *testing.T, name string, gotJSON []byte) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(goldenDir, name))
	require.NoError(t, err, "golden %s missing (it pins the deleted Go host's output; see goldens/README.md)", name)
	if jsonCanonicalEqual(want, gotJSON) {
		return
	}
	// Mismatch: dump the actual output for offline inspection, then fail with a full JSON diff.
	dumpActual(t, name, gotJSON)
	require.JSONEq(t, string(want), string(gotJSON),
		"golden %s: Rust engine output diverged from the recorded Go host pin", name)
}

// jsonCanonicalEqual reports whether two JSON blobs are equal ignoring key order and whitespace.
func jsonCanonicalEqual(a, b []byte) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	ac, err1 := json.Marshal(av)
	bc, err2 := json.Marshal(bv)
	return err1 == nil && err2 == nil && string(ac) == string(bc)
}

// requireTextMatchesGolden compares a plain-text output (e.g. a contract version() string) to the
// golden fixture verbatim.
func requireTextMatchesGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(goldenDir, name))
	require.NoError(t, err, "golden %s missing (it pins the deleted Go host's output; see goldens/README.md)", name)
	require.Equal(t, string(want), got,
		"golden %s: Rust engine output diverged from the recorded Go host pin", name)
}

// dumpActual writes actual output to a temp file so a golden mismatch is inspectable, and returns the
// path. On a match the temp dir is cleaned up automatically by the test framework.
func dumpActual(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), strings.ReplaceAll(name, "/", "_"))
	require.NoError(t, os.WriteFile(p, data, 0o644))
	t.Logf("golden %s: actual output written to %s", name, p)
	return p
}
