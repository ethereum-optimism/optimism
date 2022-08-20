package genesis

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core"
)

// BuildOptimismGenesis
func BuildOptimismGenesis(hh *hardhat.Hardhat, config *DeployConfig, chain ethereum.ChainReader) (*core.Genesis, error) {
	genesis, err := NewL2Genesis(config, chain)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genesis)

	if config.FundDevAccounts {
		FundDevAccounts(db)
	}

	// TODO(tynes): need a function for clearing old, unused storage slots.
	// Each deployed contract on L2 needs to have its existing storage
	// inspected and then cleared if they are no longer used.

	if err := SetProxies(hh, db); err != nil {
		return nil, err
	}

	storage, err := NewStorageConfig(hh, config, chain)
	if err != nil {
		return nil, err
	}

	if err := SetImplementations(hh, db, storage); err != nil {
		return nil, err
	}

	if err := MigrateDepositHashes(hh, db); err != nil {
		return nil, err
	}

	return db.Genesis(), nil
}
