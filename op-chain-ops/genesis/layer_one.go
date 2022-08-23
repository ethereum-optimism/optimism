package genesis

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/core"
)

// TODO(tynes): need bindings for all of the L1 contracts to be able
// to create a genesis file with the L1 contracts predeployed.
// This would speed up testing as deployments take time when
// running tests.
func BuildL1DeveloperGenesis(config *DeployConfig) (*core.Genesis, error) {
	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genesis)

	FundDevAccounts(db)
	SetPrecompileBalances(db)

	return db.Genesis(), nil
}
