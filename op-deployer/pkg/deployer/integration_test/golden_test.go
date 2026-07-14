package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenDir holds fixtures recorded from the in-process Go script host at the base commit. Small
// outputs (semvers) are committed raw; multi-MB genesis dumps are pinned by the SHA-256 of their
// canonical json.Marshal bytes (encoding/json sorts map keys, so the serialization is deterministic).
// Once the Go host is deleted these are the only record of its behavior — a mismatch means the Rust
// engine diverged. See goldens/README.md for provenance.
const goldenDir = "testdata/goldens"

func requireJSONMatchesGolden(t *testing.T, name string, got any) {
	t.Helper()
	gotJSON, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join(goldenDir, name))
	require.NoError(t, err, "golden %s missing (it pins the deleted Go host's output; see goldens/README.md)", name)
	if jsonCanonicalEqual(want, gotJSON) {
		return
	}
	dumpActual(t, name, gotJSON)
	require.JSONEq(t, string(want), string(gotJSON),
		"golden %s: Rust engine output diverged from the recorded Go host pin", name)
}

// requireHashMatchesGolden pins a large canonical-JSON blob by SHA-256 instead of committing it. The
// caller MUST run its structural non-vacuity guards before calling this so a hash of an empty value
// can never pass. On mismatch the actual canonical JSON is written to a temp file (logged), then the
// test fails.
func requireHashMatchesGolden(t *testing.T, name string, canonical []byte) {
	t.Helper()
	sum := sha256.Sum256(canonical)
	got := hex.EncodeToString(sum[:])
	want := readShaGolden(t, name)
	if got != want {
		p := dumpActual(t, name+".actual.json", canonical)
		t.Fatalf("golden hash %s: Rust engine output diverged from the recorded Go host pin\n"+
			"  got  sha256 = %s\n  want sha256 = %s\n  actual canonical JSON written to %s", name, got, want, p)
	}
}

func readShaGolden(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(goldenDir, name))
	require.NoError(t, err, "golden %s missing (it pins the deleted Go host's output; see goldens/README.md)", name)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return strings.Fields(line)[0]
	}
	t.Fatalf("golden %s contains no hash line", name)
	return ""
}

// goldenName sanitizes a mode name (e.g. "cgt+interop") into a filesystem-safe fixture stem.
func goldenName(mode string) string {
	return strings.ReplaceAll(mode, "+", "-")
}

func jsonCanonicalEqual(a, b []byte) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	ac, err1 := json.Marshal(av)
	bc, err2 := json.Marshal(bv)
	return err1 == nil && err2 == nil && string(ac) == string(bc)
}

func dumpActual(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), strings.ReplaceAll(name, "/", "_"))
	require.NoError(t, os.WriteFile(p, data, 0o644))
	t.Logf("golden %s: actual output written to %s", name, p)
	return p
}
