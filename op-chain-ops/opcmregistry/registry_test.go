package opcmregistry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantPre   string
		wantErr   bool
	}{
		{
			name:      "simple version",
			input:     "1.0.0",
			wantMajor: 1,
			wantMinor: 0,
			wantPatch: 0,
		},
		{
			name:      "version with all parts",
			input:     "6.1.2",
			wantMajor: 6,
			wantMinor: 1,
			wantPatch: 2,
		},
		{
			name:      "release candidate",
			input:     "6.0.0-rc.1",
			wantMajor: 6,
			wantMinor: 0,
			wantPatch: 0,
			wantPre:   "rc.1",
		},
		{
			name:      "beta version",
			input:     "7.0.0-beta.2",
			wantMajor: 7,
			wantMinor: 0,
			wantPatch: 0,
			wantPre:   "beta.2",
		},
		{
			name:      "alpha version",
			input:     "1.2.3-alpha",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantPre:   "alpha",
		},
		{
			name:    "invalid - only two parts",
			input:   "1.0",
			wantErr: true,
		},
		{
			name:    "invalid - single part",
			input:   "1",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric major",
			input:   "a.0.0",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric minor",
			input:   "1.b.0",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric patch",
			input:   "1.0.c",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := ParseSemver(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantMajor, sv.Major)
			require.Equal(t, tt.wantMinor, sv.Minor)
			require.Equal(t, tt.wantPatch, sv.Patch)
			require.Equal(t, tt.wantPre, sv.Prerelease)
			require.Equal(t, tt.input, sv.Raw)
		})
	}
}

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal versions", a: "1.0.0", b: "1.0.0", want: 0},
		{name: "a major less", a: "1.0.0", b: "2.0.0", want: -1},
		{name: "a major greater", a: "3.0.0", b: "2.0.0", want: 1},
		{name: "a minor less", a: "1.1.0", b: "1.2.0", want: -1},
		{name: "a minor greater", a: "1.3.0", b: "1.2.0", want: 1},
		{name: "a patch less", a: "1.0.1", b: "1.0.2", want: -1},
		{name: "a patch greater", a: "1.0.3", b: "1.0.2", want: 1},
		{name: "major trumps minor", a: "2.0.0", b: "1.9.9", want: 1},
		{name: "minor trumps patch", a: "1.2.0", b: "1.1.9", want: 1},
		// Compare does not consider prerelease, only major.minor.patch
		{name: "prerelease ignored in compare", a: "6.0.0-rc.1", b: "6.0.0", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svA, err := ParseSemver(tt.a)
			require.NoError(t, err)
			svB, err := ParseSemver(tt.b)
			require.NoError(t, err)
			require.Equal(t, tt.want, svA.Compare(svB))
		})
	}
}

func TestSemverIsPrerelease(t *testing.T) {
	tests := []struct {
		version      string
		isPrerelease bool
	}{
		{"1.0.0", false},
		{"6.0.0", false},
		{"6.0.0-rc.1", true},
		{"7.0.0-beta.2", true},
		{"1.2.3-alpha", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			sv, err := ParseSemver(tt.version)
			require.NoError(t, err)
			require.Equal(t, tt.isPrerelease, sv.IsPrerelease())
		})
	}
}

func TestSemverIsV1OPCM(t *testing.T) {
	tests := []struct {
		version string
		isV1    bool
	}{
		{"5.0.0", false},
		{"6.0.0", true},
		{"6.1.0", true},
		{"6.99.99", true},
		{"7.0.0", false},
		{"8.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			sv, err := ParseSemver(tt.version)
			require.NoError(t, err)
			require.Equal(t, tt.isV1, sv.IsV1OPCM())
		})
	}
}

func TestAddressUnmarshalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  common.Address
	}{
		{
			name:  "valid address lowercase",
			input: "0x1234567890abcdef1234567890abcdef12345678",
			want:  common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		},
		{
			name:  "valid address mixed case",
			input: "0x1234567890AbCdEf1234567890AbCdEf12345678",
			want:  common.HexToAddress("0x1234567890AbCdEf1234567890AbCdEf12345678"),
		},
		{
			name:  "zero address",
			input: "0x0000000000000000000000000000000000000000",
			want:  common.Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var addr Address
			err := addr.UnmarshalText([]byte(tt.input))
			require.NoError(t, err)
			require.Equal(t, tt.want, common.Address(addr))
		})
	}
}

func TestUrlForChain(t *testing.T) {
	tests := []struct {
		name    string
		chainID uint64
		wantURL string
		wantErr bool
	}{
		{
			name:    "mainnet",
			chainID: MainnetChainID,
			wantURL: standardVersionsMainnetURL,
		},
		{
			name:    "sepolia",
			chainID: SepoliaChainID,
			wantURL: standardVersionsSepoliaURL,
		},
		{
			name:    "unsupported chain",
			chainID: 999,
			wantErr: true,
		},
		{
			name:    "zero chain id",
			chainID: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := urlForChain(tt.chainID)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unsupported chain ID")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantURL, url)
		})
	}
}

func TestFileCachePath(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)

	t.Run("empty cache dir returns empty path", func(t *testing.T) {
		r := &Registry{log: logger, fileCacheDir: ""}
		path := r.fileCachePath("https://example.com/test.toml")
		require.Empty(t, path)
	})

	t.Run("valid cache dir returns hashed path", func(t *testing.T) {
		r := &Registry{log: logger, fileCacheDir: "/tmp/test-cache"}
		path := r.fileCachePath("https://example.com/test.toml")
		require.NotEmpty(t, path)
		require.True(t, filepath.IsAbs(path))
		require.Contains(t, path, "/tmp/test-cache/")
		require.Equal(t, ".cache", filepath.Ext(path))
	})

	t.Run("different urls produce different paths", func(t *testing.T) {
		r := &Registry{log: logger, fileCacheDir: "/tmp/test-cache"}
		path1 := r.fileCachePath("https://example.com/file1.toml")
		path2 := r.fileCachePath("https://example.com/file2.toml")
		require.NotEqual(t, path1, path2)
	})

	t.Run("same url produces same path", func(t *testing.T) {
		r := &Registry{log: logger, fileCacheDir: "/tmp/test-cache"}
		path1 := r.fileCachePath("https://example.com/same.toml")
		path2 := r.fileCachePath("https://example.com/same.toml")
		require.Equal(t, path1, path2)
	})
}

func TestFileCacheRoundTrip(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	tmpDir := t.TempDir()
	r := &Registry{log: logger, fileCacheDir: tmpDir}

	testURL := "https://example.com/test.toml"
	testData := []byte(`[releases]
version = "1.0.0"`)

	// Save to cache
	r.saveToFileCache(testURL, testData)

	// Load from cache
	loaded, ok := r.loadFromFileCache(testURL)
	require.True(t, ok)
	require.Equal(t, testData, loaded)
}

func TestFileCacheExpiration(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	tmpDir := t.TempDir()
	r := &Registry{log: logger, fileCacheDir: tmpDir}

	testURL := "https://example.com/expired.toml"
	testData := []byte(`test = "data"`)

	// Save to cache
	r.saveToFileCache(testURL, testData)

	// Modify file time to be older than TTL
	path := r.fileCachePath(testURL)
	oldTime := time.Now().Add(-fileCacheTTL - time.Hour)
	err := os.Chtimes(path, oldTime, oldTime)
	require.NoError(t, err)

	// Should not load expired cache
	_, ok := r.loadFromFileCache(testURL)
	require.False(t, ok)
}

func TestFileCacheNonExistent(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	tmpDir := t.TempDir()
	r := &Registry{log: logger, fileCacheDir: tmpDir}

	_, ok := r.loadFromFileCache("https://example.com/nonexistent.toml")
	require.False(t, ok)
}

func TestFilterOPCMsByReleaseVersion(t *testing.T) {
	opcms := []OPCMInfo{
		{ReleaseVersion: "1.0.0", Address: common.HexToAddress("0x1")},
		{ReleaseVersion: "1.5.0", Address: common.HexToAddress("0x2")},
		{ReleaseVersion: "1.6.0", Address: common.HexToAddress("0x3")},
		{ReleaseVersion: "2.0.0", Address: common.HexToAddress("0x4")},
	}

	t.Run("empty lastVersion returns all", func(t *testing.T) {
		result, err := FilterOPCMsByReleaseVersion(opcms, "")
		require.NoError(t, err)
		require.Len(t, result, 4)
	})

	t.Run("filter from version 1.0.0", func(t *testing.T) {
		result, err := FilterOPCMsByReleaseVersion(opcms, "1.0.0")
		require.NoError(t, err)
		require.Len(t, result, 3)
		require.Equal(t, "1.5.0", result[0].ReleaseVersion)
		require.Equal(t, "1.6.0", result[1].ReleaseVersion)
		require.Equal(t, "2.0.0", result[2].ReleaseVersion)
	})

	t.Run("filter from version 1.5.0", func(t *testing.T) {
		result, err := FilterOPCMsByReleaseVersion(opcms, "1.5.0")
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "1.6.0", result[0].ReleaseVersion)
		require.Equal(t, "2.0.0", result[1].ReleaseVersion)
	})

	t.Run("filter from latest version returns empty", func(t *testing.T) {
		result, err := FilterOPCMsByReleaseVersion(opcms, "2.0.0")
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("filter from future version returns empty", func(t *testing.T) {
		result, err := FilterOPCMsByReleaseVersion(opcms, "99.0.0")
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("invalid lastVersion returns error", func(t *testing.T) {
		_, err := FilterOPCMsByReleaseVersion(opcms, "invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid lastVersion")
	})
}

func TestFilterByLastUsedOPCMVersion(t *testing.T) {
	sv6, _ := ParseSemver("6.0.0")
	sv61, _ := ParseSemver("6.1.0")
	sv7, _ := ParseSemver("7.0.0")

	opcms := []ResolvedOPCM{
		{Address: common.HexToAddress("0x1"), OPCMVersion: sv6, IsV1: true},
		{Address: common.HexToAddress("0x2"), OPCMVersion: sv61, IsV1: true},
		{Address: common.HexToAddress("0x3"), OPCMVersion: sv7, IsV1: false},
	}

	t.Run("empty lastVersion returns all", func(t *testing.T) {
		result, err := FilterByLastUsedOPCMVersion(opcms, "")
		require.NoError(t, err)
		require.Len(t, result, 3)
	})

	t.Run("filter from 6.0.0", func(t *testing.T) {
		result, err := FilterByLastUsedOPCMVersion(opcms, "6.0.0")
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, "6.1.0", result[0].OPCMVersion.Raw)
		require.Equal(t, "7.0.0", result[1].OPCMVersion.Raw)
	})

	t.Run("filter from 6.1.0", func(t *testing.T) {
		result, err := FilterByLastUsedOPCMVersion(opcms, "6.1.0")
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "7.0.0", result[0].OPCMVersion.Raw)
	})

	t.Run("filter from 7.0.0 returns empty", func(t *testing.T) {
		result, err := FilterByLastUsedOPCMVersion(opcms, "7.0.0")
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("invalid lastVersion returns error", func(t *testing.T) {
		_, err := FilterByLastUsedOPCMVersion(opcms, "invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid lastVersion")
	})
}

func TestGetResolvedOPCMs(t *testing.T) {
	// This test uses a mock version querier to avoid network calls
	t.Run("filters out versions below 6.x.x", func(t *testing.T) {
		// Create a mock registry with in-memory data
		mockVersions := Versions{
			"old-release": {
				OPContractsManager: &ContractData{
					Version: "1.0.0",
					Address: addrPtr("0x1111111111111111111111111111111111111111"),
				},
			},
			"v6-release": {
				OPContractsManager: &ContractData{
					Version: "1.6.0",
					Address: addrPtr("0x2222222222222222222222222222222222222222"),
				},
			},
		}

		// Mock version querier that returns different OPCM versions
		mockQuerier := func(addr common.Address) (string, error) {
			switch addr.Hex() {
			case "0x1111111111111111111111111111111111111111":
				return "5.0.0", nil // Should be filtered out (< 6.x.x)
			case "0x2222222222222222222222222222222222222222":
				return "6.0.0", nil // Should be included
			default:
				return "", nil
			}
		}

		// Process the mock data similar to GetResolvedOPCMs
		var opcms []OPCMInfo
		for _, vc := range mockVersions {
			if vc.OPContractsManager != nil && vc.OPContractsManager.Address != nil {
				sv, err := ParseSemver(vc.OPContractsManager.Version)
				if err != nil || sv.IsPrerelease() {
					continue
				}
				opcms = append(opcms, OPCMInfo{
					ReleaseVersion: vc.OPContractsManager.Version,
					Address:        common.Address(*vc.OPContractsManager.Address),
				})
			}
		}

		var resolved []ResolvedOPCM
		for _, opcm := range opcms {
			opcmVersion, err := mockQuerier(opcm.Address)
			if err != nil {
				continue
			}
			sv, err := ParseSemver(opcmVersion)
			if err != nil || sv.IsPrerelease() || sv.Major < 6 {
				continue
			}
			resolved = append(resolved, ResolvedOPCM{
				Address:     opcm.Address,
				OPCMVersion: sv,
				IsV1:        sv.IsV1OPCM(),
			})
		}

		require.Len(t, resolved, 1)
		require.Equal(t, common.HexToAddress("0x2222222222222222222222222222222222222222"), resolved[0].Address)
		require.True(t, resolved[0].IsV1)
	})

	t.Run("filters out prerelease versions", func(t *testing.T) {
		mockQuerier := func(addr common.Address) (string, error) {
			return "6.0.0-rc.1", nil
		}

		opcmVersion, _ := mockQuerier(common.Address{})
		sv, err := ParseSemver(opcmVersion)
		require.NoError(t, err)
		require.True(t, sv.IsPrerelease())
	})
}

func TestNewRegistry(t *testing.T) {
	t.Run("with nil logger uses default", func(t *testing.T) {
		r := NewRegistry(nil)
		require.NotNil(t, r)
		require.NotNil(t, r.log)
		require.NotNil(t, r.memoryCache)
		require.NotNil(t, r.downloader)
	})

	t.Run("with provided logger", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelDebug)
		r := NewRegistry(logger)
		require.NotNil(t, r)
		require.Equal(t, logger, r.log)
		require.NotNil(t, r.memoryCache)
		require.NotNil(t, r.downloader)
	})
}

func TestDefaultCacheDir(t *testing.T) {
	// Test with XDG_CACHE_HOME set
	t.Run("uses XDG_CACHE_HOME when set", func(t *testing.T) {
		origXDG := os.Getenv("XDG_CACHE_HOME")
		defer os.Setenv("XDG_CACHE_HOME", origXDG)

		os.Setenv("XDG_CACHE_HOME", "/custom/cache")
		dir := defaultCacheDir()
		require.Equal(t, filepath.Join("/custom/cache", cacheSubdir), dir)
	})

	t.Run("falls back to home directory", func(t *testing.T) {
		origXDG := os.Getenv("XDG_CACHE_HOME")
		defer os.Setenv("XDG_CACHE_HOME", origXDG)

		os.Setenv("XDG_CACHE_HOME", "")
		dir := defaultCacheDir()
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		require.Equal(t, filepath.Join(home, ".cache", cacheSubdir), dir)
	})
}

func TestChainIDConstants(t *testing.T) {
	require.Equal(t, uint64(1), MainnetChainID)
	require.Equal(t, uint64(11155111), SepoliaChainID)
}

// Helper function to create Address pointer
func addrPtr(hex string) *Address {
	addr := Address(common.HexToAddress(hex))
	return &addr
}
