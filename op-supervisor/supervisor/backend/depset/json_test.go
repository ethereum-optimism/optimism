package depset

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// mockHeaderClient implements headerByHashClient for testing
type mockHeaderClient struct {
	headers map[common.Hash]*types.Header
}

func (m *mockHeaderClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if header, ok := m.headers[hash]; ok {
		return header, nil
	}
	return nil, fmt.Errorf("header not found for %s", hash.Hex())
}

func TestJSONRollupConfigsLoader_LoadRollupConfigSet(t *testing.T) {
	// Create mock headers with timestamps matching our test files
	mockHeaders := map[common.Hash]*types.Header{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"): {
			Number: big.NewInt(1),
			Time:   1000,
		},
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"): {
			Number: big.NewInt(2),
			Time:   2000,
		},
	}

	loader := &JSONRollupConfigsLoader{
		PathPattern: filepath.Join("testfiles", "rollup-*.json"),
	}

	configSet, err := loader.loadRollupConfigSet(context.Background(), &mockHeaderClient{headers: mockHeaders})
	require.NoError(t, err)

	// Verify the configs were loaded correctly
	require.True(t, configSet.HasChain(eth.ChainIDFromUInt64(10)))
	require.True(t, configSet.HasChain(eth.ChainIDFromUInt64(20)))

	// Check first chain config
	genesis1 := configSet.Genesis(eth.ChainIDFromUInt64(10))
	require.Equal(t, uint64(1), genesis1.L1.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), genesis1.L1.Hash)
	require.Equal(t, uint64(1000), genesis1.L1.Timestamp)
	require.Equal(t, uint64(0), genesis1.L2.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"), genesis1.L2.Hash)
	require.Equal(t, uint64(1000), genesis1.L2.Timestamp)

	// Check second chain config
	genesis2 := configSet.Genesis(eth.ChainIDFromUInt64(20))
	require.Equal(t, uint64(2), genesis2.L1.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"), genesis2.L1.Hash)
	require.Equal(t, uint64(2000), genesis2.L1.Timestamp)
	require.Equal(t, uint64(0), genesis2.L2.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"), genesis2.L2.Hash)
	require.Equal(t, uint64(2000), genesis2.L2.Timestamp)
}
