package genesis

import (
	"github.com/ethereum-optimism/optimism/state-surgery/hardhat"
	"github.com/ethereum-optimism/optimism/state-surgery/state"
	"github.com/ethereum/go-ethereum/core"
)

// TODO(tynes): need bindings for all of the L1 contracts
func BuildL1DeveloperGenesis(hh *hardhat.Hardhat, config *DeployConfig) (*core.Genesis, error) {
	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genesis)

	if config.FundDevAccounts {
		FundDevAccounts(db)
	}
	return db.Genesis(), nil
}
