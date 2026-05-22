package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifestHashIsDeterministic(t *testing.T) {
	a := Manifest{
		Structs: map[string]StructDef{
			"Output": {
				Fields: []Field{
					{Name: "proxy", Type: "address", Required: true},
				},
			},
			"Input": {
				Fields: []Field{
					{Name: "guardian", Type: "address", Required: true},
					{Name: "paused", Type: "bool", Required: true},
				},
			},
		},
	}
	b := Manifest{
		Structs: map[string]StructDef{
			"Input":  a.Structs["Input"],
			"Output": a.Structs["Output"],
		},
	}

	aHash, err := a.Hash()
	require.NoError(t, err)
	bHash, err := b.Hash()
	require.NoError(t, err)
	require.Equal(t, aHash, bHash)
	require.Contains(t, aHash, "sha256:")
}

func TestManifestHashExcludesSchemaHash(t *testing.T) {
	manifest := Manifest{
		SchemaHash: "stale",
		Structs: map[string]StructDef{
			"Input": {
				Fields: []Field{
					{Name: "guardian", Type: "address", Required: true},
				},
			},
		},
	}

	hashWithStaleValue, err := manifest.Hash()
	require.NoError(t, err)
	manifest.SchemaHash = "different"
	hashWithDifferentValue, err := manifest.Hash()
	require.NoError(t, err)
	require.Equal(t, hashWithStaleValue, hashWithDifferentValue)
}
