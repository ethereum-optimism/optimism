package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
			cmd := exec.Command("go", "run", "./verify.go", "--input", actual, "--expected", expected)
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.succeeds {
				t.Fatalf("unexpected result: %v\n%s", err, output)
			}
		})
	}
}
