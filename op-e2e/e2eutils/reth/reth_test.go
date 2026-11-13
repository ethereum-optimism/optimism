package reth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
)

func TestInitL2RequiresOneGenesisSource(t *testing.T) {
	for name, tc := range map[string]struct {
		genesis *opgenesis.Genesis
		path    string
	}{
		"none": {},
		"both": {genesis: &opgenesis.Genesis{}, path: "genesis.json"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := InitL2(context.Background(), nil, "test", tc.genesis, "", Config{GenesisJSONPath: tc.path})
			require.ErrorContains(t, err, "set exactly one of genesis or GenesisJSONPath")
		})
	}
}
