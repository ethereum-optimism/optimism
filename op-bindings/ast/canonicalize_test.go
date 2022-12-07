package ast

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
	"github.com/stretchr/testify/require"
)

type astIDTest struct {
	In  *solc.StorageLayout `json:"in"`
	Out *solc.StorageLayout `json:"out"`
}

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			"simple",
			"simple.json",
		},
		{
			"remap public variables",
			"public-variables.json",
		},
		{
			"values in storage",
			"values-in-storage.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(path.Join("testdata", tt.filename))
			require.NoError(t, err)
			dec := json.NewDecoder(f)
			var testData astIDTest
			require.NoError(t, dec.Decode(&testData))
			require.NoError(t, f.Close())

			// Run 100 times to make sure that we aren't relying
			// on random map iteration order.
			for i := 0; i < 100; i++ {
				require.Equal(t, testData.Out, CanonicalizeASTIDs(testData.In))
			}
		})
	}
}
