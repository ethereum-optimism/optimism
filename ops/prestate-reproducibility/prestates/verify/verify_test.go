package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKonaSP1ReleaseComparison(t *testing.T) {
	const hash = "0x1111111111111111111111111111111111111111111111111111111111111111"
	const other = "0x2222222222222222222222222222222222222222222222222222222222222222"
	registry := `[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "` + hash + `"
[[prestates."0.0.4"]]
type = "cannon64-kona"
hash = "cannon-hash"
`
	dir := t.TempDir()
	binary := filepath.Join(dir, "verify")
	buildOutput, err := exec.Command("go", "build", "-o", binary, "./verify.go").CombinedOutput()
	require.NoError(t, err, string(buildOutput))
	expected := filepath.Join(dir, "registry.toml")
	if err := os.WriteFile(expected, []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, actual string
		succeeds     bool
	}{
		{"matching", `[{"version":"0.0.5","type":"kona-sp1","hash":"` + hash + `"},{"version":"0.0.4","type":"cannon64-kona","hash":"cannon-hash"}]`, true},
		{"mismatching", `[{"version":"0.0.5","type":"kona-sp1","hash":"` + other + `"},{"version":"0.0.4","type":"cannon64-kona","hash":"cannon-hash"}]`, false},
		{"missing", `[{"version":"0.0.4","type":"cannon64-kona","hash":"cannon-hash"}]`, false},
		{"extra", `[{"version":"0.0.5","type":"kona-sp1","hash":"` + hash + `"},{"version":"0.0.4","type":"cannon64-kona","hash":"cannon-hash"},{"version":"0.0.6","type":"kona-sp1","hash":"` + hash + `"}]`, false},
		{"wrong type", `[{"version":"0.0.5","type":"cannon64-kona","hash":"` + hash + `"},{"version":"0.0.4","type":"cannon64-kona","hash":"cannon-hash"}]`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-")+".json")
			if err := os.WriteFile(actual, []byte(tc.actual), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "--input", actual, "--expected", expected)
			output, err := cmd.CombinedOutput()
			if tc.succeeds {
				require.NoError(t, err, string(output))
				return
			}
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, string(output))
			require.Equal(t, 1, exitErr.ExitCode(), string(output))
		})
	}
}
