package genesis

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	gstate "github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
)

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
func BuildL1DeveloperGenesis(config *DeployConfig, dump *gstate.Dump, l1Deployments *L1Deployments) (*core.Genesis, error) {
	log.Info("Building developer L1 genesis block")
	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create L1 developer genesis: %w", err)
	}

	memDB := state.NewMemoryStateDB(genesis)
	FundDevAccounts(memDB)
	SetPrecompileBalances(memDB)

	if dump != nil {
		for addrstr, account := range dump.Accounts {
			if !common.IsHexAddress(addrstr) {
				// Changes in https://github.com/ethereum/go-ethereum/pull/28504
				// add accounts to the Dump with "pre(<AddressHash>)" as key
				// if the address itself is nil.
				// So depending on how `dump` was created, this might be a
				// pre-image key, which we skip.
				continue
			}
			address := common.HexToAddress(addrstr)
			name := "<unknown>"
			if l1Deployments != nil {
				if n := l1Deployments.GetName(address); n != "" {
					name = n
				}
			}
			log.Info("Setting account", "name", name, "address", address.Hex())
			memDB.CreateAccount(address)
			memDB.SetNonce(address, account.Nonce)

			balance, ok := math.ParseBig256(account.Balance)
			if !ok {
				return nil, fmt.Errorf("failed to parse balance for %s", address)
			}
			memDB.AddBalance(address, balance)
			memDB.SetCode(address, account.Code)
			for key, value := range account.Storage {
				log.Info("Setting storage", "name", name, "key", key.Hex(), "value", value)
				memDB.SetState(address, key, common.HexToHash(value))
			}
		}
	}

	return memDB.Genesis(), nil
}
