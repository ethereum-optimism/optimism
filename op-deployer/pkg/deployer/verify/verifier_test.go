package verify

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func bootstrapContractAddresses() map[string]common.Address {
	addresses := make(map[string]common.Address)
	names := []string{
		"SuperchainConfigProxy",
		"SuperchainProxyAdmin",
		"OpcmV2",
		"OptimismPortalImpl",
		"SystemConfigImpl",
	}
	for i, name := range names {
		addresses[name] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}

	return addresses
}

func TestGetContractBundle(t *testing.T) {
	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	bundle := bootstrapContractAddresses()
	bundleFile := filepath.Join(testCacheDir, "contracts.json")
	bundleData, err := json.Marshal(bundle)
	require.NoError(t, err)
	err = os.WriteFile(bundleFile, bundleData, 0o644)
	require.NoError(t, err)

	retrievedBundle, err := getContractBundle(bundleFile)
	require.NoError(t, err)
	require.Equal(t, bundle, retrievedBundle)
	require.Greater(t, len(retrievedBundle), 0, "contract bundle should not be empty")
}

func TestFieldNameToContractName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name",
			input:    "SuperchainConfigProxy",
			expected: "superchain_config_proxy",
		},
		{
			name:     "With numbers",
			input:    "L1StandardBridgeProxy",
			expected: "l1_standard_bridge_proxy",
		},
		{
			name:     "Single word",
			input:    "Opcm",
			expected: "opcm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldNameToContractName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
