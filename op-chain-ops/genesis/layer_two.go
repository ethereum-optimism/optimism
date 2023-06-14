package genesis

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
)

// BuildL2DeveloperGenesis will build the developer Optimism Genesis
// Block. Suitable for devnets.
func BuildL2DeveloperGenesis(config *DeployConfig, l1StartBlock *types.Block) (*core.Genesis, error) {
	genspec, err := NewL2Genesis(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genspec)

	if config.FundDevAccounts {
		FundDevAccounts(db)
	}
	SetPrecompileBalances(db)

	storage, err := NewL2StorageConfig(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	immutable, err := NewL2ImmutableConfig(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	if err := SetL2Proxies(db); err != nil {
		return nil, err
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return nil, err
	}

	if err := SetDevOnlyL2Implementations(db, storage, immutable); err != nil {
		return nil, err
	}

	return db.Genesis(), nil
}

// BuildL2MainnetGenesis will build an L2 Genesis suitable for a Superchain mainnet that does not
// require a pre-bedrock migration & supports optional governance token predeploy. Details:
//
//   - Creates proxies for predeploys in the address space:
//     [0x4200000000000000000000000000000000000000, 0x4200000000000000000000000000000000000800)
//
//   - All predeploy proxies owned by the ProxyAdmin
//
//   - Predeploys as per the spec except for no LegacyERC20ETH predeploy at
//     0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000
//
//   - optional governance token at 0x4200000000000000000000000000000000000042 if
//     config.EnableGovernance is true (& otherwise a no-impl proxy remains at this address)
//
//   - no accounts are pre-funded
func BuildL2MainnetGenesis(config *DeployConfig, l1StartBlock *types.Block) (*core.Genesis, error) {
	genspec, err := NewL2Genesis(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genspec)

	storage, err := NewL2StorageConfig(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	immutable, err := NewL2ImmutableConfig(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	// Set up the proxies
	depBytecode, err := bindings.GetDeployedBytecode("Proxy")
	if err != nil {
		return nil, err
	}
	if len(depBytecode) == 0 {
		return nil, errors.New("Proxy has empty bytecode")
	}
	for i := uint64(0); i <= 2048; i++ {
		bigAddr := new(big.Int).Or(bigL2PredeployNamespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)
		db.CreateAccount(addr)
		db.SetCode(addr, depBytecode)
		db.SetState(addr, AdminSlot, predeploys.ProxyAdminAddr.Hash())
	}

	// Set up the implementations
	deployResults, err := immutables.BuildOptimism(immutable)
	if err != nil {
		return nil, err
	}
	for name, predeploy := range predeploys.Predeploys {
		addr := *predeploy
		if predeploys.IsDeprecated(addr) {
			continue
		}
		if addr == predeploys.GovernanceTokenAddr && !config.EnableGovernance {
			// there is no governance token configured, so skip the governance token predeploy
			log.Warn("Governance is not enabled, skipping governance token predeploy.")
			continue
		}
		codeAddr := addr
		if predeploys.IsProxied(addr) {
			codeAddr, err = AddressToCodeNamespace(addr)
			if err != nil {
				return nil, fmt.Errorf("error converting to code namespace: %w", err)
			}
			db.CreateAccount(codeAddr)
			db.SetState(addr, ImplementationSlot, codeAddr.Hash())
		} else {
			db.DeleteState(addr, AdminSlot)
		}
		if err := setupPredeploy(db, deployResults, storage, name, addr, codeAddr); err != nil {
			return nil, err
		}
		code := db.GetCode(codeAddr)
		if len(code) == 0 {
			return nil, fmt.Errorf("code not set for %s", name)
		}
	}

	return db.Genesis(), nil
}
