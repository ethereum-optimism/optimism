package genesis

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
)

// PrecompileCount represents the number of precompile addresses
// starting from `address(0)` to PrecompileCount that are funded
// with a single wei in the genesis state.
const PrecompileCount = 256

var (
	// uint128Max is type(uint128).max and is set in the init function.
	uint128Max = new(big.Int)
	// The default values for the ResourceConfig, used as part of
	// an EIP-1559 curve for deposit gas.
	DefaultResourceConfig = bindings.ResourceMeteringResourceConfig{
		MaxResourceLimit:            20_000_000,
		ElasticityMultiplier:        10,
		BaseFeeMaxChangeDenominator: 8,
		MinimumBaseFee:              params.GWei,
		SystemTxMaxGas:              1_000_000,
	}
)

func init() {
	var ok bool
	uint128Max, ok = new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
	if !ok {
		panic("bad uint128Max")
	}
	// Set the maximum base fee on the default config.
	DefaultResourceConfig.MaximumBaseFee = uint128Max
}

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
	FundDevAccounts(genesis)
	SetPrecompileBalances(genesis)

	l1Deployments.ForEach(func(name string, addr common.Address) {
		acc := genesis.Alloc[addr]
		if isAccountEmpty(acc) {
			log.Info("Excluded L1 deployment", "name", name, "address", addr)
		} else {
			log.Info("Included L1 deployment", "name", name, "address", addr, "balance", acc.Balance, "storage", len(acc.Storage), "nonce", acc.Nonce)
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

// isAccountEmpty returns true if the account is considered empty.
func isAccountEmpty(acc types.Account) bool {
	isEmptyCode := len(acc.Code) == 0
	isEmptyStorage := len(acc.Storage) == 0
	isZeroNonce := acc.Nonce == 0
	isEmptyBalance := acc.Balance == nil || acc.Balance.Cmp(common.Big0) == 0
	return isEmptyCode && isEmptyStorage && isZeroNonce && isEmptyBalance
}
