package main

import (
	"strings"
	"testing"
)

const goodHash = "0x1111111111111111111111111111111111111111111111111111111111111111"

func TestKonaSP1RegistrySelection(t *testing.T) {
	registry := []byte(`[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "` + goodHash + `"
[[prestates."0.0.5"]]
type = "cannon64-kona"
hash = "other"
[[prestates."0.0.4"]]
type = "kona-sp1"
hash = "` + goodHash + `"
`)
	versions, err := selectKonaSP1Versions(registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0] != "0.0.4" || versions[1] != "0.0.5" {
		t.Fatalf("unexpected versions: %v", versions)
	}
	empty, err := selectKonaSP1Versions([]byte(`[[prestates."0.0.5"]]
type = "cannon64-kona"
hash = "other"
`))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty selection: %v, %v", empty, err)
	}
}

func TestKonaSP1RegistrySelectionRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"duplicate": `[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "` + goodHash + `"
[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "` + goodHash + `"
`,
		"malformed hash": `[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "0x1234"
`,
		"zero hash": `[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "0x` + strings.Repeat("0", 64) + `"
`,
		"invalid version": `[[prestates."../0.0.5"]]
type = "kona-sp1"
hash = "` + goodHash + `"
`,
		"invalid TOML": `[[prestates."0.0.5"]
`,
	}
	for name, registry := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := selectKonaSP1Versions([]byte(registry)); err == nil {
				t.Fatal("expected selection error")
			}
		})
	}
}
