package genesis

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/core"
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

	return BuildL2Genesis(db, config, l1StartBlock)
}

// BuildL2Genesis will build the L2 Optimism Genesis Block
func BuildL2Genesis(db *state.MemoryStateDB, config *DeployConfig, l1Block *types.Block) (*core.Genesis, error) {
	if err := SetL2Proxies(db); err != nil {
		return nil, err
	}

	storage, err := NewL2StorageConfig(config, l1Block)
	if err != nil {
		return nil, err
	}

	immutable, err := NewL2ImmutableConfig(config, l1Block)
	if err != nil {
		return nil, err
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return nil, err
	}

	return db.Genesis(), nil
}
