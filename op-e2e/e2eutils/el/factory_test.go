package el

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"
)

func TestInitL2GenesisInputValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  L2Config
		wantErr string
	}{
		{
			name:   "op-reth in-memory genesis",
			config: L2Config{Kind: services.ELKindOpReth, Genesis: &core.Genesis{}},
		},
		{
			name:   "op-reth genesis path",
			config: L2Config{Kind: services.ELKindOpReth, GenesisJSONPath: "genesis.json"},
		},
		{
			name:    "op-reth missing genesis",
			config:  L2Config{Kind: services.ELKindOpReth},
			wantErr: "set exactly one of Genesis or GenesisJSONPath",
		},
		{
			name: "op-reth conflicting genesis inputs",
			config: L2Config{
				Kind:            services.ELKindOpReth,
				Genesis:         &core.Genesis{},
				GenesisJSONPath: "genesis.json",
			},
			wantErr: "set exactly one of Genesis or GenesisJSONPath",
		},
		{
			name:   "op-geth in-memory genesis",
			config: L2Config{Kind: services.ELKindOpGeth, Genesis: &core.Genesis{}},
		},
		{
			name:    "op-geth genesis path",
			config:  L2Config{Kind: services.ELKindOpGeth, GenesisJSONPath: "genesis.json"},
			wantErr: "op-geth requires an in-memory genesis",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateGenesisInput(test.config)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}
