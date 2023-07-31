package genesis

import (
	"encoding/json"
	"testing"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestMigrateAlloc(t *testing.T) {
	input := map[common.Address]LegacyGenesisAccount{
		{1}: {
			Code: "0x",
			Storage: map[common.Hash]string{
				{1}: "0x",
			},
			Nonce: 1,
		},
		{2}: {
			Code: "0x",
			Storage: map[common.Hash]string{
				{1}: "0x",
				{2}: "0x",
			},
			Nonce: 2,
		},
	}
	expected := types.GenesisAlloc{
		{1}: {
			Code:    []byte{},
			Balance: common.Big0,
			Nonce:   1,
			Storage: map[common.Hash]common.Hash{
				{1}: {},
			},
		},
		{2}: {
			Code:    []byte{},
			Balance: common.Big0,
			Nonce:   2,
			Storage: map[common.Hash]common.Hash{
				{1}: {},
				{2}: {},
			},
		},
	}
	bytes, err := json.Marshal(input)
	require.NoError(t, err)
	genesisAlloc, err := MigrateAlloc(bytes)
	require.NoError(t, err)
	require.Equal(t, expected, genesisAlloc)
}
