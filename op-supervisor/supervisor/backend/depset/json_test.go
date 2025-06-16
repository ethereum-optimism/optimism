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

func TestJSONRollupConfigSetLoader_LoadRollupConfigSet(t *testing.T) {
	loader := &JSONRollupConfigSetLoader{
		Path: filepath.Join("testfiles", "rollup_set.json"),
	}

	configSet, err := loader.LoadRollupConfigSet(context.Background())
	require.NoError(t, err)

	// Verify both chains are present
	require.True(t, configSet.HasChain(eth.ChainIDFromUInt64(10)))
	require.True(t, configSet.HasChain(eth.ChainIDFromUInt64(20)))

	// Check first chain config (chain ID 10)
	genesis1 := configSet.Genesis(eth.ChainIDFromUInt64(10))
	require.Equal(t, uint64(1), genesis1.L1.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), genesis1.L1.Hash)
	require.Equal(t, uint64(1000), genesis1.L1.Timestamp)
	require.Equal(t, uint64(0), genesis1.L2.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"), genesis1.L2.Hash)
	require.Equal(t, uint64(1100), genesis1.L2.Timestamp)

	// Check second chain config (chain ID 20)
	genesis2 := configSet.Genesis(eth.ChainIDFromUInt64(20))
	require.Equal(t, uint64(2), genesis2.L1.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"), genesis2.L1.Hash)
	require.Equal(t, uint64(2000), genesis2.L1.Timestamp)
	require.Equal(t, uint64(0), genesis2.L2.Number)
	require.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"), genesis2.L2.Hash)
	require.Equal(t, uint64(2100), genesis2.L2.Timestamp)

	// Test Interop activation checks
	require.False(t, configSet.IsInterop(eth.ChainIDFromUInt64(10), 1500))
	require.True(t, configSet.IsInterop(eth.ChainIDFromUInt64(10), 2500))
	require.False(t, configSet.IsInterop(eth.ChainIDFromUInt64(20), 2500))
	require.True(t, configSet.IsInterop(eth.ChainIDFromUInt64(20), 3500))

	// Test Interop activation block checks
	require.False(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(10), 1998))
	require.True(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(10), 2000))
	require.False(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(10), 2002))
	require.False(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(20), 2998))
	require.True(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(20), 3000))
	require.False(t, configSet.IsInteropActivationBlock(eth.ChainIDFromUInt64(20), 3002))
}
