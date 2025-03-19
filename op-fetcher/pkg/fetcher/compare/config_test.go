package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestReadChainConfigs(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]script.ChainConfig // filename -> content
		expectedLen int
		expectError bool
	}{
		{
			name: "valid json files",
			setupFiles: map[string]script.ChainConfig{
				"10.json": {
					ChainId:   10,
					ChainName: "optimism",
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
						L1StandardBridgeProxy: common.HexToAddress("0x1234567890123456789012345678901234567891"),
					},
				},
				"20.json": {
					ChainId:   20,
					ChainName: "base",
					Addresses: script.Addresses{
						SystemConfigProxy:     common.HexToAddress("0x2234567890123456789012345678901234567890"),
						L1StandardBridgeProxy: common.HexToAddress("0x2234567890123456789012345678901234567891"),
					},
				},
			},
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "error on files with zero chain ID",
			setupFiles: map[string]script.ChainConfig{
				"123.json": {
					ChainName: "zero",
				},
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name: "invalid json format",
			setupFiles: map[string]script.ChainConfig{
				"10.json": {
					ChainId:   10,
					ChainName: "optimism",
				},
				"invalid.json": {}, // Will be overwritten with invalid JSON
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name:        "empty directory",
			setupFiles:  map[string]script.ChainConfig{},
			expectedLen: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "chain-configs-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for name, config := range tt.setupFiles {
				filePath := filepath.Join(tmpDir, name)
				var data []byte

				if name == "invalid.json" {
					// Write invalid JSON for the invalid test case
					data = []byte("This is not valid JSON")
				} else {
					// Marshal the config to JSON
					data, err = json.MarshalIndent(config, "", "  ")
					if err != nil {
						t.Fatalf("Failed to marshal config: %v", err)
					}
				}

				err := os.WriteFile(filePath, data, 0644)
				if err != nil {
					t.Fatalf("Failed to write test file %s: %v", name, err)
				}
			}

			configs, err := readChainConfigs(tmpDir)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, configs, tt.expectedLen)

				for _, expectedConfig := range tt.setupFiles {
					actualConfig, exists := configs[expectedConfig.ChainId]
					require.True(t, exists, "Config for chain ID %d should exist", expectedConfig.ChainId)
					require.Equal(t, expectedConfig.ChainName, actualConfig.ChainName)
					require.Equal(t, expectedConfig.ChainId, actualConfig.ChainId)
					require.Equal(t, expectedConfig.Addresses.SystemConfigProxy, actualConfig.Addresses.SystemConfigProxy)
					require.Equal(t, expectedConfig.Addresses.L1StandardBridgeProxy, actualConfig.Addresses.L1StandardBridgeProxy)
				}
			}
		})
	}
}

func TestReadChainList(t *testing.T) {
	tests := []struct {
		name        string
		entries     []ChainListEntry
		expectedLen int
		expectError bool
	}{
		{
			name: "valid chain list",
			entries: []ChainListEntry{
				{
					ChainID: 10,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      true,
						Permissionless:    false,
						RespectedGameType: 1,
					},
				},
				{
					ChainID: 20,
					FaultProofStatus: script.FaultProofStatus{
						Permissioned:      false,
						Permissionless:    true,
						RespectedGameType: 2,
					},
				},
			},
			expectedLen: 2,
			expectError: false,
		},
		{
			name:        "empty chain list",
			entries:     []ChainListEntry{},
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for the chain list
			tmpFile, err := os.CreateTemp("", "chain-list-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write the chain list entries to the file
			data, err := json.MarshalIndent(tt.entries, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal chain list: %v", err)
			}

			_, err = tmpFile.Write(data)
			if err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			chainList, err := readChainList(tmpFile.Name())
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, chainList, tt.expectedLen)

				for _, expectedEntry := range tt.entries {
					actualEntry, exists := chainList[expectedEntry.ChainID]
					require.True(t, exists, "Entry for chain ID %d should exist", expectedEntry.ChainID)
					require.Equal(t, expectedEntry.ChainID, actualEntry.ChainID)
					require.Equal(t, expectedEntry.FaultProofStatus.Permissioned, actualEntry.FaultProofStatus.Permissioned)
					require.Equal(t, expectedEntry.FaultProofStatus.Permissionless, actualEntry.FaultProofStatus.Permissionless)
					require.Equal(t, expectedEntry.FaultProofStatus.RespectedGameType, actualEntry.FaultProofStatus.RespectedGameType)
				}
			}
		})
	}

	t.Run("file not found", func(t *testing.T) {
		_, err := readChainList("nonexistent-file.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read chain list file")
	})

	t.Run("invalid json", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-json-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("This is not valid JSON")
		if err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		_, err = readChainList(tmpFile.Name())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal chain list file")
	})
}
