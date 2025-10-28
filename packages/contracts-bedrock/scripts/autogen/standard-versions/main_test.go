package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractVersionFromSource(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantVersion string
		wantErr     bool
	}{
		{
			name: "function version pattern - single line",
			content: `contract Test {
    function version() public pure returns (string memory) { return "1.2.3"; }
}`,
			wantVersion: "1.2.3",
			wantErr:     false,
		},
		{
			name: "function version pattern - multi line",
			content: `contract Test {
    function version() public pure returns (string memory) {
        return "2.0.0";
    }
}`,
			wantVersion: "2.0.0",
			wantErr:     false,
		},
		{
			name: "constant version pattern - public",
			content: `contract Test {
    string public constant version = "3.1.0";
}`,
			wantVersion: "3.1.0",
			wantErr:     false,
		},
		{
			name: "constant version pattern - private",
			content: `contract Test {
    string private constant version = "4.5.6";
}`,
			wantVersion: "4.5.6",
			wantErr:     false,
		},
		{
			name: "constant version pattern - internal",
			content: `contract Test {
    string internal constant version = "1.0.0-beta.1";
}`,
			wantVersion: "1.0.0-beta.1",
			wantErr:     false,
		},
		{
			name: "version in complex contract",
			content: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

contract MyContract {
    uint256 public value;

    function version() public pure returns (string memory) {
        return "5.0.0";
    }

    function getValue() public view returns (uint256) {
        return value;
    }
}`,
			wantVersion: "5.0.0",
			wantErr:     false,
		},
		{
			name:        "no version found",
			content:     `contract Test { }`,
			wantVersion: "",
			wantErr:     true,
		},
		{
			name: "version variable but not the right pattern",
			content: `contract Test {
    string public name = "version 1.0.0";
}`,
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "Test.sol")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Test
			version, err := extractVersionFromSource(tmpFile)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "version not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVersion, version)
			}
		})
	}
}

func TestExtractVersionFromSource_FileErrors(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		_, err := extractVersionFromSource("/nonexistent/path/Test.sol")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})
}

func TestIsZeroAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{
			name: "zero address with 0x prefix",
			addr: "0x0000000000000000000000000000000000000000",
			want: true,
		},
		{
			name: "zero address without 0x prefix",
			addr: "0000000000000000000000000000000000000000",
			want: true,
		},
		{
			name: "zero address uppercase",
			addr: "0x0000000000000000000000000000000000000000",
			want: true,
		},
		{
			name: "non-zero address",
			addr: "0x1234567890123456789012345678901234567890",
			want: false,
		},
		{
			name: "non-zero address without 0x",
			addr: "1234567890123456789012345678901234567890",
			want: false,
		},
		{
			name: "address with one non-zero digit",
			addr: "0x0000000000000000000000000000000000000001",
			want: false,
		},
		{
			name: "empty string",
			addr: "",
			want: true,
		},
		{
			name: "only 0x",
			addr: "0x",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isZeroAddress(tt.addr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTomlKeyFromABIName(t *testing.T) {
	tests := []struct {
		name    string
		abiName string
		want    string
		wantErr bool
	}{
		{
			name:    "superchainConfigImpl",
			abiName: "superchainConfigImpl",
			want:    "superchain_config",
			wantErr: false,
		},
		{
			name:    "protocolVersionsImpl",
			abiName: "protocolVersionsImpl",
			want:    "protocol_versions",
			wantErr: false,
		},
		{
			name:    "l1ERC721BridgeImpl",
			abiName: "l1ERC721BridgeImpl",
			want:    "l1_erc721_bridge",
			wantErr: false,
		},
		{
			name:    "optimismPortalImpl",
			abiName: "optimismPortalImpl",
			want:    "optimism_portal",
			wantErr: false,
		},
		{
			name:    "optimismPortalInteropImpl",
			abiName: "optimismPortalInteropImpl",
			want:    "optimism_portal_interop",
			wantErr: false,
		},
		{
			name:    "disputeGameFactoryImpl",
			abiName: "disputeGameFactoryImpl",
			want:    "dispute_game_factory",
			wantErr: false,
		},
		{
			name:    "mipsImpl",
			abiName: "mipsImpl",
			want:    "mips",
			wantErr: false,
		},
		{
			name:    "preimageOracleImpl",
			abiName: "preimageOracleImpl",
			want:    "preimage_oracle",
			wantErr: false,
		},
		{
			name:    "faultDisputeGameV2Impl",
			abiName: "faultDisputeGameV2Impl",
			want:    "fault_dispute_game_v2",
			wantErr: false,
		},
		{
			name:    "permissionedDisputeGameV2Impl",
			abiName: "permissionedDisputeGameV2Impl",
			want:    "permissioned_dispute_game_v2",
			wantErr: false,
		},
		{
			name:    "unknown mapping",
			abiName: "unknownImpl",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			abiName: "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tomlKeyFromABIName(tt.abiName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown ABI field name")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNameMappingCompleteness(t *testing.T) {
	// Verify that all entries in nameMapping are valid TOML keys
	for abiName, tomlKey := range nameMapping {
		t.Run(abiName, func(t *testing.T) {
			// TOML keys should be snake_case and not empty
			assert.NotEmpty(t, tomlKey, "TOML key should not be empty")
			assert.NotContains(t, tomlKey, " ", "TOML key should not contain spaces")
			assert.NotContains(t, tomlKey, "-", "TOML key should use underscore, not hyphen")

			// Verify the mapping is bidirectional (can look up via function)
			result, err := tomlKeyFromABIName(abiName)
			assert.NoError(t, err)
			assert.Equal(t, tomlKey, result)
		})
	}
}

func TestImplementationStruct(t *testing.T) {
	t.Run("create implementation", func(t *testing.T) {
		impl := Implementation{
			Name:    "testImpl",
			Address: "0x1234567890123456789012345678901234567890",
		}

		assert.Equal(t, "testImpl", impl.Name)
		assert.Equal(t, "0x1234567890123456789012345678901234567890", impl.Address)
	})
}

func TestABIStructures(t *testing.T) {
	t.Run("ABIComponent creation", func(t *testing.T) {
		comp := ABIComponent{
			Name: "testField",
			Type: "address",
		}
		assert.Equal(t, "testField", comp.Name)
		assert.Equal(t, "address", comp.Type)
	})

	t.Run("ABIOutput creation", func(t *testing.T) {
		output := ABIOutput{
			Components: []ABIComponent{
				{Name: "field1", Type: "address"},
				{Name: "field2", Type: "uint256"},
			},
			Name: "result",
			Type: "tuple",
		}
		assert.Equal(t, "result", output.Name)
		assert.Equal(t, "tuple", output.Type)
		assert.Len(t, output.Components, 2)
	})

	t.Run("ABIFunction creation", func(t *testing.T) {
		fn := ABIFunction{
			Name: "implementations",
			Outputs: []ABIOutput{
				{
					Name: "impls",
					Type: "tuple",
				},
			},
			StateMutability: "view",
			Type:            "function",
		}
		assert.Equal(t, "implementations", fn.Name)
		assert.Equal(t, "view", fn.StateMutability)
		assert.Equal(t, "function", fn.Type)
		assert.Len(t, fn.Outputs, 1)
	})
}
