package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamModFileFailsWithoutUpstream(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte(`module example.com/x

go 1.24

require github.com/ethereum/go-ethereum v1.17.0

replace github.com/ethereum/go-ethereum => github.com/ethereum-optimism/op-geth v1.101702.4
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.sum"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "x.go"), []byte(
		"package x\n\nimport _ \"github.com/ethereum/go-ethereum/common\"\n"), 0o644))
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GOFLAGS", "-modcacherw")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "off")
	old := retryDelay
	retryDelay = 0
	t.Cleanup(func() { retryDelay = old })

	_, _, err := upstreamModFile(t.Context(), root, t.TempDir(), "v1.17.10")
	require.ErrorContains(t, err, "github.com/ethereum/go-ethereum@v1.17.10: module lookup disabled by GOPROXY=off")
}

func writeAllowlist(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "allowlist.txt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestAllowlistRatchet(t *testing.T) {
	path := writeAllowlist(t, `# header

a | missing go-ethereum/types.X | kept
b | missing go-ethereum/types.Y | gone
# trailing comment
c | missing go-ethereum/types.A | B | reason after the last separator
`)
	allow, err := readAllowlist(path)
	require.NoError(t, err)

	unlisted, stale := allow.diff(map[Use]bool{
		{"a", "missing go-ethereum/types.X"}:     true,
		{"c", "missing go-ethereum/types.A | B"}: true,
		{"d", "missing go-ethereum/types.Z"}:     true,
	})
	require.Equal(t, []Use{{"d", "missing go-ethereum/types.Z"}}, unlisted)
	require.Equal(t, []Use{{"b", "missing go-ethereum/types.Y"}}, stale)

	require.NoError(t, allow.writeWithout(path, stale))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `# header

a | missing go-ethereum/types.X | kept
# trailing comment
c | missing go-ethereum/types.A | B | reason after the last separator
`, string(got))
}

func TestAllowlistRejects(t *testing.T) {
	for content, want := range map[string]string{
		"a | missing go-ethereum/types.X\n":                             "want",
		"a | missing go-ethereum/types.X | \n":                          "needs a reason",
		"a | missing go-ethereum/types.X | " + reasonPlaceholder + "\n": "needs a reason",
		"a | x | one\na | x | two\n":                                    "duplicate entry",
	} {
		_, err := readAllowlist(writeAllowlist(t, content))
		require.ErrorContains(t, err, want, content)
	}
}
