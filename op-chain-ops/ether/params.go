package ether

import (
	"math/big"
)

// Params contains the configuration parameters used for verifying
// the integrity of the migration.
type Params struct {
	// ExpectedSupplyDelta is the expected delta between the total supply of OVM ETH,
	// and ETH we were able to migrate. This is used to account for supply bugs in
	//previous regenesis events.
	ExpectedSupplyDelta *big.Int
}

var ParamsByChainID = map[int]*Params{
	1: {
		// Regenesis 4 contained a supply bug, and the Saurik bug likely
		// inflated supply further by ~0.0012 ETH.
		new(big.Int).SetUint64(1627270011999999992),
	},
}
