package pipeline

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func PreinstallL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "preinstall-l1-dev-genesis")
	lgr.Info("Adding preinstalls to L1 dev genesis")

	if err := opcm.InsertPreinstalls(env.L1ScriptHost); err != nil {
		return fmt.Errorf("failed to add preinstalls to L1 dev state: %w", err)
	}
	// L1-only request queues are omitted by the shared L2 preinstall script.
	// Released geth requires them for Prague and Amsterdam payloads.
	queues := types.GenesisAlloc{
		params.WithdrawalQueueAddress:    {Code: params.WithdrawalQueueCode},
		params.ConsolidationQueueAddress: {Code: params.ConsolidationQueueCode},
		params.BuilderDepositAddress:     {Code: params.BuilderDepositCode},
		params.BuilderExitAddress:        {Code: params.BuilderExitCode},
	}
	dump, err := env.L1ScriptHost.StateDump()
	if err != nil {
		return fmt.Errorf("failed to read L1 dev state for request queues: %w", err)
	}
	for addr, queue := range queues {
		// Retain prefunded balances and any existing storage/nonces.
		account := dump.Accounts[addr]
		account.Code = queue.Code
		if account.Nonce == 0 {
			account.Nonce = 1
		}
		env.L1ScriptHost.ImportAccount(addr, account)
	}
	env.L1ScriptHost.Wipe(env.Deployer)

	return nil
}
