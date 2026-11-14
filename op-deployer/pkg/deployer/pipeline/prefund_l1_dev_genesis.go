package pipeline

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

// PrefundL1DevGenesis pre-funds accounts in the L1 dev genesis for testing purposes
func PrefundL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "prefund-l1-dev-genesis")
	lgr.Info("Prefunding accounts in L1 dev genesis")

	if intent.L1DevGenesisParams == nil {
		lgr.Warn("No L1 dev params, will not prefund any accounts")
		return nil
	}
	prefundMap := intent.L1DevGenesisParams.Prefund
	if len(prefundMap) == 0 {
		lgr.Warn("Not prefunding any L1 dev accounts. L1 dev genesis may not be usable.")
		return nil
	}
	for addr, amount := range prefundMap {
		env.Forge.Genesis.SetBalance(addr, (*uint256.Int)(amount).ToBig())
	}
	lgr.Info("Prefunded dev accounts on L1", "accounts", len(prefundMap))
	return nil
}

// prefunds reports whether the intent gives an account a positive L1 prefund.
func prefunds(intent *state.Intent, addr common.Address) bool {
	if intent.L1DevGenesisParams == nil {
		return false
	}
	amount := intent.L1DevGenesisParams.Prefund[addr]
	return amount != nil && !(*uint256.Int)(amount).IsZero()
}
