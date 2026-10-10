package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
[[prestates."0.0.5-rc.1"]]
type = "kona-sp1"
hash = "` + goodHash + `"
`)
	versions, err := selectKonaSP1Versions(registry)
	require.NoError(t, err)
	require.Equal(t, []string{"0.0.4", "0.0.5", "0.0.5-rc.1"}, versions)
	empty, err := selectKonaSP1Versions([]byte(`[[prestates."0.0.5"]]
type = "cannon64-kona"
hash = "other"
`))
	require.NoError(t, err)
	require.Empty(t, empty)
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
		"uppercase hash": `[[prestates."0.0.5"]]
type = "kona-sp1"
hash = "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
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
			_, err := selectKonaSP1Versions([]byte(registry))
			require.Error(t, err)
		})
	}
}
