package genesis

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)

// PrecompileCount represents the number of precompile addresses
// starting from `address(0)` to PrecompileCount that are funded
// with a single wei in the genesis state.
const PrecompileCount = 256

// BuildL1DeveloperGenesis will create a L1 genesis block after creating
// all of the state required for an Optimism network to function.
// It is expected that the dump contains all of the required state to bootstrap
// the L1 chain.
func BuildL1DeveloperGenesis(config *DeployConfig, dump *ForgeAllocs, l1Deployments *L1Deployments) (*core.Genesis, error) {
	log.Info("Building developer L1 genesis block")
	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create L1 developer genesis: %w", err)
	}

	if genesis.Alloc != nil && len(genesis.Alloc) != 0 {
		panic("Did not expect NewL1Genesis to generate non-empty state") // sanity check for dev purposes.
	}
	// copy, for safety when the dump is reused (like in e2e testing)
	genesis.Alloc = dump.Copy().Accounts
	if config.FundDevAccounts {
		FundDevAccounts(genesis)
	}
	SetPrecompileBalances(genesis)

	l1Deployments.ForEach(func(name string, addr common.Address) {
		acc, ok := genesis.Alloc[addr]
		if ok {
			log.Info("Included L1 deployment", "name", name, "address", addr, "balance", acc.Balance, "storage", len(acc.Storage), "nonce", acc.Nonce)
		} else {
			log.Info("Excluded L1 deployment", "name", name, "address", addr)
		}
	})

	return genesis, nil
}

// FundDevAccounts will fund each of the development accounts.
func FundDevAccounts(gen *core.Genesis) {
	for _, account := range DevAccounts {
		acc := gen.Alloc[account]
		if acc.Balance == nil {
			acc.Balance = new(big.Int)
		}
		acc.Balance = acc.Balance.Add(acc.Balance, devBalance)
		gen.Alloc[account] = acc
	}
}

// SetPrecompileBalances will set a single wei at each precompile address.
// This is an optimization to make calling them cheaper.
func SetPrecompileBalances(gen *core.Genesis) {
	for i := 0; i < PrecompileCount; i++ {
		addr := common.BytesToAddress([]byte{byte(i)})
		acc := gen.Alloc[addr]
		if acc.Balance == nil {
			acc.Balance = new(big.Int)
		}
		acc.Balance = acc.Balance.Add(acc.Balance, big.NewInt(1))
		gen.Alloc[addr] = acc
	}
}
