package pipeline

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

// EIP-8282's L1-only queues are not yet defined in the linked op-geth params.
// These allocations match go-ethereum v1.17.7's DeveloperGenesisBlock:
// https://github.com/ethereum/go-ethereum/blob/v1.17.7/params/protocol_params.go
//
//go:embed fixtures/eip8282_allocs.json
var eip8282AllocsJSON []byte

func PreinstallL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "preinstall-l1-dev-genesis")
	lgr.Info("Adding preinstalls to L1 dev genesis")

	if err := opcm.InsertPreinstalls(env.L1ScriptHost); err != nil {
		return fmt.Errorf("failed to add preinstalls to L1 dev state: %w", err)
	}
	// L1-only request queues are omitted by the shared L2 preinstall script.
	// Released geth requires them for Prague and Amsterdam payloads.
	var queues types.GenesisAlloc
	if err := json.Unmarshal(eip8282AllocsJSON, &queues); err != nil {
		return fmt.Errorf("failed to decode EIP-8282 dev allocations: %w", err)
	}
	queues[params.WithdrawalQueueAddress] = types.Account{Nonce: 1, Code: params.WithdrawalQueueCode}
	queues[params.ConsolidationQueueAddress] = types.Account{Nonce: 1, Code: params.ConsolidationQueueCode}
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
