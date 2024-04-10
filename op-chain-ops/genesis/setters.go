package genesis

import (
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// PrecompileCount represents the number of precompile addresses
// starting from `address(0)` to PrecompileCount that are funded
// with a single wei in the genesis state.
const PrecompileCount = 256

// FundDevAccounts will fund each of the development accounts.
func FundDevAccounts(db vm.StateDB) {
	for _, account := range DevAccounts {
		if !db.Exist(account) {
			db.CreateAccount(account)
		}
		db.AddBalance(account, uint256.MustFromBig(devBalance))
	}
}

// SetPrecompileBalances will set a single wei at each precompile address.
// This is an optimization to make calling them cheaper.
func SetPrecompileBalances(db vm.StateDB) {
	for i := 0; i < PrecompileCount; i++ {
		addr := common.BytesToAddress([]byte{byte(i)})
		db.CreateAccount(addr)
		db.AddBalance(addr, uint256.NewInt(1))
	}
}
