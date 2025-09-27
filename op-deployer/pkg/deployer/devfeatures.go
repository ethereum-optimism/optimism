package deployer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Development feature flag constants that mirror the solidity DevFeatures library.
// These use a 32 byte bitmap for easy integration between op-deployer and contracts.
var (
	// OptimismPortalInterop enables the OptimismPortalInterop contract.
	OptimismPortalInterop = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	// CannonKona enables Kona as the default cannon prover.
	CannonKona = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000010")

	// DeployV2DisputeGames enables deployment of V2 dispute game contracts.
	DeployV2DisputeGames = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000100")
)

// IsDevFeatureEnabled checks if a specific development feature is enabled in a feature bitmap.
// It performs a bitwise AND operation between the bitmap and the feature flag to determine
// if the feature is enabled. This follows the same pattern as the solidity DevFeatures library.
func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
	b := new(big.Int).SetBytes(bitmap[:])
	f := new(big.Int).SetBytes(flag[:])
	return new(big.Int).And(b, f).BitLen() != 0
}
