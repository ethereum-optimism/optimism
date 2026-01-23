// Package opcmregistry provides utilities for loading OPCM (OP Contracts Manager)
// information from the superchain-registry. This package is used by both Go code
// and Solidity tests via FFI.
package opcmregistry

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// HTTP client configuration
const (
	httpTimeout     = 30 * time.Second // Request timeout
	maxResponseSize = 10 * 1024 * 1024 // 10MB max response size
)

// httpClient is a shared HTTP client with timeout
var httpClient = &http.Client{
	Timeout: httpTimeout,
}

// Chain ID constants
const (
	MainnetChainID = uint64(1)
	SepoliaChainID = uint64(11155111)
)

// GitHub raw URLs for the standard versions TOML files
const (
	standardVersionsMainnetURL = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/validation/standard/standard-versions-mainnet.toml"
	standardVersionsSepoliaURL = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/validation/standard/standard-versions-sepolia.toml"
)

// Dummy prestates for testing - actual values don't matter for upgrade tests
var (
	DummyCannonPrestate     = crypto.Keccak256Hash([]byte("CANNON"))
	DummyCannonKonaPrestate = crypto.Keccak256Hash([]byte("CANNON_KONA"))
)

// Address is a hex-encoded address used in TOML parsing
type Address common.Address

func (a *Address) UnmarshalText(text []byte) error {
	addr := common.HexToAddress(string(text))
	*a = Address(addr)
	return nil
}

// ContractData represents the version and address information for a contract in the TOML
type ContractData struct {
	Version               string   `toml:"version"`
	Address               *Address `toml:"address,omitempty"`
	ImplementationAddress *Address `toml:"implementation_address,omitempty"`
}

// VersionConfig represents all contracts for a specific release version in the TOML
type VersionConfig struct {
	OPContractsManager *ContractData `toml:"op_contracts_manager,omitempty"`
}

// Versions maps release tags to their contract configurations
type Versions map[string]VersionConfig

// Cache for fetched versions
var (
	versionsCache   = make(map[uint64]Versions)
	versionsCacheMu sync.RWMutex
)

// fetchVersions fetches the standard versions TOML from GitHub for a given chain
func fetchVersions(chainID uint64) (Versions, error) {
	var url string
	switch chainID {
	case MainnetChainID:
		url = standardVersionsMainnetURL
	case SepoliaChainID:
		url = standardVersionsSepoliaURL
	default:
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %d", url, resp.StatusCode)
	}

	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var versions Versions
	if err := toml.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return versions, nil
}

// getVersionsForChain returns the versions for a chain, fetching from GitHub if needed
func getVersionsForChain(chainID uint64) (Versions, error) {
	versionsCacheMu.RLock()
	if v, ok := versionsCache[chainID]; ok {
		versionsCacheMu.RUnlock()
		return v, nil
	}
	versionsCacheMu.RUnlock()

	versionsCacheMu.Lock()
	defer versionsCacheMu.Unlock()

	// Double-check after acquiring write lock
	if v, ok := versionsCache[chainID]; ok {
		return v, nil
	}

	versions, err := fetchVersions(chainID)
	if err != nil {
		return nil, err
	}

	versionsCache[chainID] = versions
	return versions, nil
}

// OPCMInfo contains information about an OPCM from the registry.
// Note: This only contains registry metadata. The actual OPCM version (and whether it's V1/V2)
// must be determined by querying opcm.version() on-chain.
type OPCMInfo struct {
	// ReleaseVersion is the contracts release version from the registry (e.g., "1.6.0").
	// This is NOT the OPCM contract's semver - use opcm.version() on-chain to get that.
	ReleaseVersion string
	Address        common.Address
	ChainID        uint64
}

// Semver represents a parsed semantic version
type Semver struct {
	Major      int
	Minor      int
	Patch      int
	Raw        string
	Prerelease string // e.g., "rc.1", "beta.2", empty for stable releases
}

// ParseSemver parses a semantic version string like "6.0.0" or "6.0.0-rc.1"
func ParseSemver(v string) (Semver, error) {
	base := v
	prerelease := ""
	if idx := strings.Index(v, "-"); idx != -1 {
		base = v[:idx]
		prerelease = v[idx+1:]
	}

	parts := strings.Split(base, ".")
	if len(parts) < 3 {
		return Semver{}, fmt.Errorf("invalid semver: %s", v)
	}

	var major, minor, patch int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return Semver{}, fmt.Errorf("invalid major version: %s", v)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return Semver{}, fmt.Errorf("invalid minor version: %s", v)
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
		return Semver{}, fmt.Errorf("invalid patch version: %s", v)
	}

	return Semver{Major: major, Minor: minor, Patch: patch, Raw: v, Prerelease: prerelease}, nil
}

// IsPrerelease returns true if this is a prerelease version (e.g., rc, beta, alpha)
func (s Semver) IsPrerelease() bool {
	return s.Prerelease != ""
}

// Compare returns -1 if s < other, 0 if s == other, 1 if s > other
func (s Semver) Compare(other Semver) int {
	if s.Major != other.Major {
		if s.Major < other.Major {
			return -1
		}
		return 1
	}
	if s.Minor != other.Minor {
		if s.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if s.Patch != other.Patch {
		if s.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// IsV1OPCM returns true if this version is a V1 OPCM (6.x.x)
func (s Semver) IsV1OPCM() bool {
	return s.Major == 6
}

// GetOPCMsForChain returns all OPCMs for a given chain ID by fetching from the superchain-registry GitHub.
// Returns unique OPCMs sorted by release version ascending, deduplicated by address.
// Note: ReleaseVersion (e.g., "1.6.0") is NOT the OPCM contract's semver (e.g., "6.0.0").
// The actual OPCM version must be queried on-chain via opcm.version().
// Prerelease versions (e.g., rc, beta) are excluded - only stable releases are returned.
func GetOPCMsForChain(chainID uint64) ([]OPCMInfo, error) {
	versions, err := getVersionsForChain(chainID)
	if err != nil {
		return nil, err
	}

	var opcms []OPCMInfo

	for _, versionConfig := range versions {
		if versionConfig.OPContractsManager == nil {
			continue
		}
		if versionConfig.OPContractsManager.Address == nil {
			continue
		}

		releaseVersion := versionConfig.OPContractsManager.Version

		// Skip prerelease versions (rc, beta, alpha, etc.)
		sv, err := ParseSemver(releaseVersion)
		if err != nil {
			continue
		}
		if sv.IsPrerelease() {
			continue
		}

		opcms = append(opcms, OPCMInfo{
			ReleaseVersion: releaseVersion,
			Address:        common.Address(*versionConfig.OPContractsManager.Address),
			ChainID:        chainID,
		})
	}

	// Sort by release version ascending
	sort.Slice(opcms, func(i, j int) bool {
		vi, _ := ParseSemver(opcms[i].ReleaseVersion)
		vj, _ := ParseSemver(opcms[j].ReleaseVersion)
		return vi.Compare(vj) < 0
	})

	// Deduplicate by address (keep first occurrence which has lowest version)
	seen := make(map[common.Address]bool)
	var result []OPCMInfo
	for _, opcm := range opcms {
		if !seen[opcm.Address] {
			seen[opcm.Address] = true
			result = append(result, opcm)
		}
	}

	return result, nil
}

// FilterOPCMsByReleaseVersion filters OPCMs to only include those with release version > lastVersion.
// If lastVersion is empty, returns all OPCMs.
func FilterOPCMsByReleaseVersion(opcms []OPCMInfo, lastVersion string) ([]OPCMInfo, error) {
	if lastVersion == "" {
		return opcms, nil
	}

	lastSV, err := ParseSemver(lastVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid lastVersion: %w", err)
	}

	var result []OPCMInfo
	for _, opcm := range opcms {
		sv, err := ParseSemver(opcm.ReleaseVersion)
		if err != nil {
			continue
		}
		if sv.Compare(lastSV) > 0 {
			result = append(result, opcm)
		}
	}

	return result, nil
}

// OPCMVersionQuerier is a function that queries the OPCM contract's version() on-chain.
// It takes an OPCM address and returns the OPCM semver string (e.g., "6.0.0", "7.0.0").
type OPCMVersionQuerier func(addr common.Address) (string, error)

// ResolvedOPCM contains an OPCM with its on-chain version resolved via opcm.version().
type ResolvedOPCM struct {
	Address     common.Address
	OPCMVersion Semver // The actual OPCM semver from opcm.version() (e.g., "6.0.0")
	IsV1        bool   // true for 6.x.x, false for 7.x.x+
}

// GetResolvedOPCMs fetches OPCM addresses from the registry, queries their OPCM versions
// on-chain using the provided querier, filters to only include stable versions >= 6.x.x,
// and returns them sorted by OPCM version ascending.
// Prerelease versions (e.g., rc, beta) are excluded.
func GetResolvedOPCMs(chainID uint64, queryOPCMVersion OPCMVersionQuerier) ([]ResolvedOPCM, error) {
	registryOPCMs, err := GetOPCMsForChain(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OPCMs from registry: %w", err)
	}

	var resolved []ResolvedOPCM
	for _, opcm := range registryOPCMs {
		opcmVersion, err := queryOPCMVersion(opcm.Address)
		if err != nil {
			// Skip OPCMs we can't query
			continue
		}

		sv, err := ParseSemver(opcmVersion)
		if err != nil {
			// Skip OPCMs with invalid versions
			continue
		}

		// Skip prerelease versions (rc, beta, alpha, etc.)
		if sv.IsPrerelease() {
			continue
		}

		// Only include versions >= 6.x.x (V1 OPCMs start at 6.x.x)
		if sv.Major < 6 {
			continue
		}

		resolved = append(resolved, ResolvedOPCM{
			Address:     opcm.Address,
			OPCMVersion: sv,
			IsV1:        sv.IsV1OPCM(),
		})
	}

	// Sort by OPCM version ascending
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].OPCMVersion.Compare(resolved[j].OPCMVersion) < 0
	})

	return resolved, nil
}

// FilterByLastUsedOPCMVersion filters resolved OPCMs to only include those with OPCM version > lastVersion.
// If lastVersion is empty, returns all OPCMs.
func FilterByLastUsedOPCMVersion(opcms []ResolvedOPCM, lastVersion string) ([]ResolvedOPCM, error) {
	if lastVersion == "" {
		return opcms, nil
	}

	lastSV, err := ParseSemver(lastVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid lastVersion: %w", err)
	}

	var result []ResolvedOPCM
	for _, opcm := range opcms {
		if opcm.OPCMVersion.Compare(lastSV) > 0 {
			result = append(result, opcm)
		}
	}

	return result, nil
}
