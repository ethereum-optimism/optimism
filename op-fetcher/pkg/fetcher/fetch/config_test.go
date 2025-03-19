package fetch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadChainTomls(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string // filename -> content
		expectedLen int
		expectError bool
	}{
		{
			name: "valid toml files",
			setupFiles: map[string]string{
				"chain1.toml": `chain_id = 10
                [addresses]
                SystemConfigProxy = "0x1234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x1234567890123456789012345678901234567891"`,
				"chain2.toml": `chain_id = 20
                [addresses]
                SystemConfigProxy = "0x2234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
			},
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "invalid toml format",
			setupFiles: map[string]string{
				"chain1.toml": `chain_id = 10
                [addresses]
                SystemConfigProxy = "0x2234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
				"invalid.toml": `This is not valid TOML`,
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name: "missing chain_id",
			setupFiles: map[string]string{
				"missing_id.toml": `[addresses]
                SystemConfigProxy = "0x2234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name: "missing SystemConfigProxy",
			setupFiles: map[string]string{
				"chain1.toml": `[addresses]
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name: "ignore superchain.toml",
			setupFiles: map[string]string{
				"chain1.toml": `chain_id = 10
                [addresses]
                SystemConfigProxy = "0x2234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
				"superchain.toml": `Some content that should be ignored`,
			},
			expectedLen: 1,
			expectError: false,
		},
		{
			name: "ignore non-toml files",
			setupFiles: map[string]string{
				"chain1.toml": `chain_id = 10
                [addresses]
                SystemConfigProxy = "0x2234567890123456789012345678901234567890"
                L1StandardBridgeProxy = "0x2234567890123456789012345678901234567891"`,
				"readme.md": "# This is documentation",
			},
			expectedLen: 1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "chain-tomls-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test input files
			for name, content := range tt.setupFiles {
				filePath := filepath.Join(tmpDir, name)
				err := os.WriteFile(filePath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file %s: %v", name, err)
				}
			}

			configs, err := readChainTomls(tmpDir)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(configs) != tt.expectedLen {
					t.Errorf("Expected %d configs, got %d", tt.expectedLen, len(configs))
				}
			}
		})
	}
}
