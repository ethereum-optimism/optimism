package crossdomain

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
	// Ethereum Mainnet L2
	288: {
		new(big.Int).SetInt64(-94819327096614),
	},
	// Ethereum Sepolia L2
	28882: {
		new(big.Int),
	},
}

var CustomLegacyETHSlotCheck = map[int]bool{}
