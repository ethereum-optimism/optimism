package deployer

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/flags"
	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultCacheDir(t *testing.T) {
	cacheDir := flags.DefaultCacheDir()
	require.NotNil(t, cacheDir)
}

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		name          string
		versionStr    string
		targetMajor   int
		targetMinor   int
		targetPatch   int
		expected      bool
		expectError   bool
		errorContains string
	}{
		// Exact matches
		{
			name:        "exact match 7.0.0",
			versionStr:  "7.0.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		{
			name:        "exact match 6.5.3",
			versionStr:  "6.5.3",
			targetMajor: 6,
			targetMinor: 5,
			targetPatch: 3,
			expected:    true,
			expectError: false,
		},
		// Greater versions
		{
			name:        "greater major version",
			versionStr:  "8.0.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		{
			name:        "greater minor version",
			versionStr:  "7.1.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		{
			name:        "greater patch version",
			versionStr:  "7.0.1",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		// Lesser versions
		{
			name:        "lesser major version",
			versionStr:  "6.0.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    false,
			expectError: false,
		},
		{
			name:        "lesser minor version",
			versionStr:  "7.0.0",
			targetMajor: 7,
			targetMinor: 1,
			targetPatch: 0,
			expected:    false,
			expectError: false,
		},
		{
			name:        "lesser patch version",
			versionStr:  "7.0.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 1,
			expected:    false,
			expectError: false,
		},
		// Version with 'v' prefix
		{
			name:        "version with v prefix",
			versionStr:  "v7.0.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		{
			name:        "version with v prefix greater",
			versionStr:  "v7.1.2",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		// Missing patch version
		{
			name:        "missing patch version exact match",
			versionStr:  "7.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 0,
			expected:    true,
			expectError: false,
		},
		{
			name:        "missing patch version less than target patch",
			versionStr:  "7.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 1,
			expected:    false,
			expectError: false,
		},
		{
			name:        "missing patch version greater minor",
			versionStr:  "7.1",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 5,
			expected:    true,
			expectError: false,
		},
		// Edge cases with major/minor only
		{
			name:        "major greater minor less",
			versionStr:  "8.0.0",
			targetMajor: 7,
			targetMinor: 9,
			targetPatch: 9,
			expected:    true,
			expectError: false,
		},
		{
			name:        "same major minor greater patch less",
			versionStr:  "7.1.0",
			targetMajor: 7,
			targetMinor: 0,
			targetPatch: 9,
			expected:    true,
			expectError: false,
		},
		// Invalid formats
		{
			name:          "invalid format no dots",
			versionStr:    "700",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid version format",
		},
		{
			name:          "invalid format single number",
			versionStr:    "7",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid version format",
		},
		{
			name:          "invalid major version not a number",
			versionStr:    "abc.0.0",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid major version",
		},
		{
			name:          "invalid minor version not a number",
			versionStr:    "7.abc.0",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid minor version",
		},
		{
			name:          "invalid patch version not a number",
			versionStr:    "7.0.abc",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid patch version",
		},
		{
			name:          "empty string",
			versionStr:    "",
			targetMajor:   7,
			targetMinor:   0,
			targetPatch:   0,
			expectError:   true,
			errorContains: "invalid version format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsVersionAtLeast(tt.versionStr, tt.targetMajor, tt.targetMinor, tt.targetPatch)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
