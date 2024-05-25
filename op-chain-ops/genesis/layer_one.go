package genesis

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
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
func BuildL1DeveloperGenesis(config *DeployConfig, dump *gstate.Dump, l1Deployments *L1Deployments, postProcess bool) (*core.Genesis, error) {
	log.Info("Building developer L1 genesis block")
	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create L1 developer genesis: %w", err)
	}

	memDB := state.NewMemoryStateDB(genesis)
	FundDevAccounts(memDB)
	SetPrecompileBalances(memDB)

	if dump != nil {
		for address, account := range dump.Accounts {
			name := "<unknown>"
			if l1Deployments != nil {
				if n := l1Deployments.GetName(address); n != "" {
					name = n
				}
			}
			log.Info("Setting account", "name", name, "address", address.Hex())
			memDB.CreateAccount(address)
			memDB.SetNonce(address, account.Nonce)

			balance, ok := new(big.Int).SetString(account.Balance, 10)
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

		// This should only be used if we are expecting Optimism specific state to be set
		if postProcess {
			if err := PostProcessL1DeveloperGenesis(memDB, l1Deployments); err != nil {
				return nil, fmt.Errorf("failed to post process L1 developer genesis: %w", err)
			}
		}
	}

	return memDB.Genesis(), nil
}

// PostProcessL1DeveloperGenesis will apply post processing to the L1 genesis
// state. This is required to handle edge cases in the genesis generation.
// `block.number` is used during deployment and without specifically setting
// the value to 0, it will cause underflow reverts for deposits in testing.
func PostProcessL1DeveloperGenesis(stateDB *state.MemoryStateDB, deployments *L1Deployments) error {
	log.Info("Post processing state")

	if stateDB == nil {
		return errors.New("cannot post process nil stateDB")
	}
	if deployments == nil {
		return errors.New("cannot post process dump with nil deployments")
	}

	if !stateDB.Exist(deployments.OptimismPortalProxy) {
		return fmt.Errorf("portal proxy doesn't exist at %s", deployments.OptimismPortalProxy)
	}

	layout, err := bindings.GetStorageLayout("OptimismPortal")
	if err != nil {
		return errors.New("failed to get storage layout for OptimismPortal")
	}

	entry, err := layout.GetStorageLayoutEntry("params")
	if err != nil {
		return errors.New("failed to get storage layout entry for OptimismPortal.params")
	}
	slot := common.BigToHash(big.NewInt(int64(entry.Slot)))

	stateDB.SetState(deployments.OptimismPortalProxy, slot, common.Hash{})
	log.Info("Post process update", "address", deployments.OptimismPortalProxy, "slot", slot.Hex(), "value", common.Hash{}.Hex())

	return nil
}
