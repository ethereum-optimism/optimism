package pipeline

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

// PrefundL2DevGenesis pre-funds accounts in the L2 dev genesis for testing purposes
func PrefundL2DevGenesis(env *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "prefund-l2-dev-genesis")
	lgr.Info("Prefunding accounts in L2 dev genesis")

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisChainState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	if thisIntent.L2DevGenesisParams == nil {
		lgr.Warn("No L2 dev params, will not prefund any accounts")
		return nil
	}
	prefundMap := thisIntent.L2DevGenesisParams.Prefund
	if len(prefundMap) == 0 {
		lgr.Warn("Not prefunding any L2 dev accounts. L2 dev genesis may not be usable.")
		return nil
	}

	for addr, amount := range prefundMap {
		acc := thisChainState.Allocs.Data.Accounts[addr]
		acc.Balance = (*uint256.Int)(amount).ToBig()
		thisChainState.Allocs.Data.Accounts[addr] = acc
	}
	lgr.Info("Prefunded dev accounts on L2", "accounts", len(prefundMap))
	return nil
}
