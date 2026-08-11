package devfeatures

import (
	"github.com/ethereum/go-ethereum/common"
)

// Development feature flag constants.
//
// IMPORTANT: each value here MUST be byte-for-byte identical to the matching constant in
// packages/contracts-bedrock/src/libraries/DevFeatures.sol. These values are independent
// literals. No compile-time check keeps them in sync. A mismatch silently diverges Go and Solidity
// behavior.
//
// When adding a dev feature, update:
// - packages/contracts-bedrock/scripts/libraries/Config.sol
// - packages/contracts-bedrock/test/setup/FeatureFlags.sol
// - .circleci/continue/main.yml (&features_matrix)
//
// Use packages/contracts-bedrock/src/libraries/DevFeatures.sol as the full checklist.
var (
	// OptimismPortalInteropFlag enables interop features in OptimismPortal2.
	OptimismPortalInteropFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	// DeployV2DisputeGamesFlag enables deployment of V2 dispute game contracts.
	DeployV2DisputeGamesFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000100")

	// ZKDisputeGameFlag enables the ZK dispute game system.
	// TODO(#19432): Use this flag in the OPCM/OPD integration pipeline.
	ZKDisputeGameFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000001000000")

	// SuperRootGamesMigrationFlag enables the super root games migration path in OPCM upgrade.
	SuperRootGamesMigrationFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000010000000")

	// OutputRootGamesFlag selects output root dispute games instead of the default super root games.
	OutputRootGamesFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000001000000000")
)

// IsDevFeatureEnabled checks if a specific development feature is enabled in a feature bitmap.
// It performs a bitwise AND between the bitmap and flag to determine if the feature
// is set. This follows the same pattern as the Solidity DevFeatures library.
func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
	// SuperRootGamesMigration is enabled by default, but fresh chains may explicitly select
	// output root games. If both flags are set, the explicit super root flag takes precedence.
	// TODO(#21662): remove with the broader SuperRootGamesMigrationFlag cleanup.
	if hasFlag(flag, SuperRootGamesMigrationFlag) &&
		(!hasFlag(bitmap, OutputRootGamesFlag) || hasFlag(bitmap, SuperRootGamesMigrationFlag)) {
		return true
	}
	return flag != (common.Hash{}) && hasFlag(bitmap, flag)
}

// hasFlag reports whether all bits of flag are set in features.
func hasFlag(features, flag common.Hash) bool {
	for i := 0; i < 32; i++ {
		if features[i]&flag[i] != flag[i] {
			return false
		}
	}
	return true
}

// EnableDevFeature sets a specific development feature flag in a feature bitmap.
func EnableDevFeature(bitmap, flag common.Hash) common.Hash {
	var result common.Hash
	for i := 0; i < 32; i++ {
		result[i] = bitmap[i] | flag[i]
	}
	return result
}
