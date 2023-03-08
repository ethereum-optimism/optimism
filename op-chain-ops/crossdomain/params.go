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
	1: {
		// Regenesis 4 (Nov 11 2021) contained a supply bug such that the total OVM ETH
		// supply was 1.628470012 ETH greater than the sum balance of every account migrated
		// / during the regenesis. A further 0.0012 ETH was incorrectly not removed from the
		// total supply by accidental invocations of the Saurik bug (https://www.saurik.com/optimism.html).
		new(big.Int).SetUint64(1627270011999999992),
	},
	5: {
		new(big.Int),
	},
}
