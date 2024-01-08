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
		// Calculated at checkpoint block=1,000,000 - there is a
		// small overcommittment of the OVM_ETH TotalSupply
		new(big.Int).SetInt64(-94819327096614),
	},
	// Goerli L2
	2888: {
		new(big.Int),
	},
	// Bobabeam
	1294: {
		new(big.Int),
	},
	// Bobaopera
	301: {
		new(big.Int),
	},
}

var CustomLegacyETHSlotCheck = map[int]bool{}
