package state_surgery

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Params contains the configuration parameters used for verifying
// the integrity of the migration.
type Params struct {
	// KnownMissingKeys is a set of known OVM ETH storage keys that are unaccounted for.
	KnownMissingKeys map[common.Hash]bool

	// ExpectedSupplyDelta is the expected delta between the total supply of OVM ETH,
	// and ETH we were able to migrate. This is used to account for supply bugs in
	//previous regenesis events.
	ExpectedSupplyDelta *big.Int
}

var ParamsByChainID = map[int]*Params{
	1: {
		// These storage keys were unaccounted for in the genesis state of regenesis 5.
		map[common.Hash]bool{
			common.HexToHash("0x8632b3478ce27e6c2251f16f71bf134373ff9d23cff5b8d5f95475fa6e52fe22"): true,
			common.HexToHash("0x47c25b07402d92e0d7f0cd9e347329fa0d86d16717cf933f836732313929fc1f"): true,
			common.HexToHash("0x2acc0ec5cc86ffda9ceba005a317bcf0e86863e11be3981e923d5b103990055d"): true,
		},
		// Regenesis 4 contained a supply bug.
		new(big.Int).SetUint64(1637102600003999992),
	},
}
