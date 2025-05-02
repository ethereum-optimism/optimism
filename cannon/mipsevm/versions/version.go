package versions

import (
	"errors"
	"slices"
)

type StateVersion uint8

const (
	// VersionSingleThreaded is the version of the Cannon STF found in op-contracts/v1.6.0 - https://github.com/ethereum-optimism/optimism/blob/op-contracts/v1.6.0/packages/contracts-bedrock/src/cannon/MIPS.sol
	VersionSingleThreaded StateVersion = iota
	// VersionMultiThreaded is the original implementation of 32-bit multithreaded cannon, tagged at cannon/v1.3.0
	VersionMultiThreaded
	// VersionSingleThreaded2 is based on VersionSingleThreaded with the addition of support for fcntl(F_GETFD) syscall
	// This is the latest 32-bit single-threaded vm, tagged at cannon/v1.4.0
	VersionSingleThreaded2
	// VersionMultiThreaded64 is the original 64-bit MTCannon implementation (pre-audit), tagged at cannon/v1.2.0
	VersionMultiThreaded64
	// VersionMultiThreaded64_v2 includes an audit fix to ensure futex values are always 32-bit, tagged at cannon/v1.3.0
	VersionMultiThreaded64_v2
	// VersionMultiThreaded_v2 is the latest 32-bit multithreaded vm, tagged at cannon/v1.4.0
	VersionMultiThreaded_v2
	// VersionMultiThreaded64_v3 includes futex handling simplification
	VersionMultiThreaded64_v3
	// VersionMultiThreaded64_v4 is the latest 64-bit multithreaded vm, includes support for new syscall eventfd2 and dclo/dclz instructions
	VersionMultiThreaded64_v4
)

var StateVersionTypes = []StateVersion{
	VersionSingleThreaded,
	VersionMultiThreaded,
	VersionSingleThreaded2,
	VersionMultiThreaded64,
	VersionMultiThreaded64_v2,
	VersionMultiThreaded_v2,
	VersionMultiThreaded64_v3,
	VersionMultiThreaded64_v4,
}

func (s StateVersion) String() string {
	switch s {
	case VersionSingleThreaded:
		return "singlethreaded"
	case VersionMultiThreaded:
		return "multithreaded"
	case VersionSingleThreaded2:
		return "singlethreaded-2"
	case VersionMultiThreaded64:
		return "multithreaded64"
	case VersionMultiThreaded64_v2:
		return "multithreaded64-2"
	case VersionMultiThreaded_v2:
		return "multithreaded-2"
	case VersionMultiThreaded64_v3:
		return "multithreaded64-3"
	case VersionMultiThreaded64_v4:
		return "multithreaded64-4"
	default:
		return "unknown"
	}
}

func ParseStateVersion(ver string) (StateVersion, error) {
	switch ver {
	case "singlethreaded":
		return VersionSingleThreaded, nil
	case "multithreaded":
		return VersionMultiThreaded, nil
	case "singlethreaded-2":
		return VersionSingleThreaded2, nil
	case "multithreaded64":
		return VersionMultiThreaded64, nil
	case "multithreaded64-2":
		return VersionMultiThreaded64_v2, nil
	case "multithreaded-2":
		return VersionMultiThreaded_v2, nil
	case "multithreaded64-3":
		return VersionMultiThreaded64_v3, nil
	case "multithreaded64-4":
		return VersionMultiThreaded64_v4, nil
	default:
		return StateVersion(0), errors.New("unknown state version")
	}
}

func IsValidStateVersion(ver StateVersion) bool {
	return slices.Contains(StateVersionTypes, ver)
}

func GetStateVersionStrings() []string {
	vers := make([]string, len(StateVersionTypes))
	for i, v := range StateVersionTypes {
		vers[i] = v.String()
	}
	return vers
}

// GetCurrentVersion returns the current default version.
func GetCurrentVersion() StateVersion {
	return VersionMultiThreaded64_v4
}

// GetExperimentalVersion returns the newest in-development version of Cannon if it exists, otherwise returns the same
// value as GetCurrentVersion.
func GetExperimentalVersion() StateVersion {
	lastVersionIndex := len(StateVersionTypes) - 1
	return StateVersionTypes[lastVersionIndex]
}

// IsSupportedMultiThreaded64 returns true if the state version is a 64-bit multithreaded VM that is currently supported
func IsSupportedMultiThreaded64(ver StateVersion) bool {
	return ver == GetCurrentVersion() || ver == GetExperimentalVersion()
}

// IsSupported returns true if the state version is currently supported
func IsSupported(ver int) bool {
	stateVer := StateVersion(ver)
	return IsSupportedMultiThreaded64(stateVer)
}
