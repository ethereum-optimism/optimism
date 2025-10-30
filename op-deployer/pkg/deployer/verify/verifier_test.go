package verify

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func bootstrapContractAddresses() map[string]common.Address {
	addrType := reflect.TypeOf(common.Address{})
	structTypes := []reflect.Type{
		reflect.TypeOf((*opcm.DeploySuperchainOutput)(nil)).Elem(),
		reflect.TypeOf((*opcm.DeployImplementationsOutput)(nil)).Elem(),
	}

	addresses := make(map[string]common.Address)
	index := int64(1)

	for _, structType := range structTypes {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.Type == addrType {
				addresses[field.Name] = common.BigToAddress(big.NewInt(index))
				index++
			}
		}
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

func TestGetArtifactPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OptimismPortal",
			input:    "OptimismPortal",
			expected: "OptimismPortal2.sol/OptimismPortal2.json",
		},
		{
			name:     "Regular contract",
			input:    "SystemConfig",
			expected: "SystemConfig.sol/SystemConfig.json",
		},
		{
			name:     "Contract with Address suffix",
			input:    "protocolVersionsProxyAddress",
			expected: "Proxy.sol/Proxy.json",
		},
		{
			name:     "Proxy contract",
			input:    "superchainConfigProxyAddress",
			expected: "Proxy.sol/Proxy.json",
		},
		{
			name:     "Implementation with Impl suffix",
			input:    "protocolVersionsImplAddress",
			expected: "ProtocolVersions.sol/ProtocolVersions.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArtifactPath(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
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
