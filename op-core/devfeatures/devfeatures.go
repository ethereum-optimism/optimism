package devfeatures

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Development feature flag constants.
var (
	// OptimismPortalInteropFlag enables interop features in OptimismPortal2.
	OptimismPortalInteropFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	// CannonKonaFlag enables Kona as the default cannon prover.
	CannonKonaFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000010")

	// DeployV2DisputeGamesFlag enables deployment of V2 dispute game contracts.
	DeployV2DisputeGamesFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000100")

	// L2CMFlag enables L2CM.
	L2CMFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000100000")

	// ZKDisputeGameFlag enables the ZK dispute game system.
	// TODO(#19432): Use this flag in the OPCM/OPD integration pipeline.
	ZKDisputeGameFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000001000000")

	// SuperRootGamesMigrationFlag enables the super root games migration path in OPCM upgrade.
	SuperRootGamesMigrationFlag = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000010000000")
)

// IsDevFeatureEnabled checks if a specific development feature is enabled in a feature bitmap.
// It performs a bitwise AND between the bitmap and flag to determine if the feature
// is set. This follows the same pattern as the Solidity DevFeatures library.
func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
	b := new(big.Int).SetBytes(bitmap[:])
	f := new(big.Int).SetBytes(flag[:])

	featuresIsNonZero := f.Cmp(big.NewInt(0)) != 0
	bitmapContainsFeatures := new(big.Int).And(b, f).Cmp(f) == 0
	return featuresIsNonZero && bitmapContainsFeatures
}

// EnableDevFeature sets a specific development feature flag in a feature bitmap.
func EnableDevFeature(bitmap, flag common.Hash) common.Hash {
	var result common.Hash
	for i := 0; i < 32; i++ {
		result[i] = bitmap[i] | flag[i]
	}
	return result
}
