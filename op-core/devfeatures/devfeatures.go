package devfeatures

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Development feature flag constants.
var (
	// OptimismPortalInterop enables the OptimismPortalInterop contract.
	OptimismPortalInterop = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	// CannonKona enables Kona as the default cannon prover.
	CannonKona = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000010")

	// DeployV2DisputeGames enables deployment of V2 dispute game contracts.
	DeployV2DisputeGames = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000100")

	// OPCMV2 enables the OPContractsManagerV2 contract.
	OPCMV2 = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")

	// L2CM enables L2CM.
	L2CM = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000100000")

	// ZKDisputeGame enables the ZK dispute game system.
	// TODO(#19432): Use this flag in the OPCM/OPD integration pipeline.
	ZKDisputeGame = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000001000000")

	// SuperRootGamesMigration enables the super root games migration path in OPCM upgrade.
	SuperRootGamesMigration = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000010000000")
)

// IsEnabled checks if a specific development feature is enabled in a feature bitmap.
// It performs a bitwise AND between the bitmap and flag to determine if the feature
// is set. This follows the same pattern as the Solidity DevFeatures library.
func IsEnabled(bitmap, flag common.Hash) bool {
	b := new(big.Int).SetBytes(bitmap[:])
	f := new(big.Int).SetBytes(flag[:])

	featuresIsNonZero := f.Cmp(big.NewInt(0)) != 0
	bitmapContainsFeatures := new(big.Int).And(b, f).Cmp(f) == 0
	return featuresIsNonZero && bitmapContainsFeatures
}

// Enable sets a specific development feature flag in a feature bitmap.
func Enable(bitmap, flag common.Hash) common.Hash {
	var result common.Hash
	for i := 0; i < 32; i++ {
		result[i] = bitmap[i] | flag[i]
	}
	return result
}
