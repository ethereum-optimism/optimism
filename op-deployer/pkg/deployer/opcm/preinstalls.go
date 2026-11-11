package opcm

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

type PreinstallsScript struct {
	SetPreinstalls func() error
}

// InsertPreinstalls inserts preinstalls in the given host.
// This is part of the L2Genesis already, but is isolated to be reused for L1 dev genesis.
func InsertPreinstalls(host *script.Host) error {
	l2GenesisScript, cleanupL2Genesis, err := script.WithScript[PreinstallsScript](host, "SetPreinstalls.s.sol", "SetPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to load SetPreinstalls script: %w", err)
	}
	defer cleanupL2Genesis()

	if err := l2GenesisScript.SetPreinstalls(); err != nil {
		return fmt.Errorf("failed to set preinstalls: %w", err)
	}
	return nil
}

// InsertL1Preinstalls inserts the shared preinstalls plus the L1-only request queues, which the
// shared L2 preinstall script omits. Released geth requires them for Prague and Amsterdam payloads.
func InsertL1Preinstalls(host *script.Host) error {
	if err := InsertPreinstalls(host); err != nil {
		return err
	}
	dump, err := host.StateDump()
	if err != nil {
		return fmt.Errorf("failed to read L1 state for request queues: %w", err)
	}
	for addr, queue := range requestQueues {
		host.ImportAccount(addr, requestQueueAccount(dump.Accounts[addr], queue))
	}
	return nil
}

// requestQueues are the L1-only request queue preinstalls.
var requestQueues = types.GenesisAlloc{
	params.WithdrawalQueueAddress:    {Code: params.WithdrawalQueueCode},
	params.ConsolidationQueueAddress: {Code: params.ConsolidationQueueCode},
	params.BuilderDepositAddress:     {Code: params.BuilderDepositCode},
	params.BuilderExitAddress:        {Code: params.BuilderExitCode},
}

// requestQueueAccount puts the queue code on the account, and keeps its balance, storage and nonce.
func requestQueueAccount(account types.Account, queue types.Account) types.Account {
	account.Code = queue.Code
	if account.Nonce == 0 {
		account.Nonce = 1
	}
	return account
}

func NewSetPreinstallsForgeCaller(client *forge.Client) forge.ScriptCaller[forge.None, forge.None] {
	return forge.NewScriptCaller(
		client,
		"scripts/SetPreinstalls.s.sol:SetPreinstalls",
		"runWithBytes(bytes)",
		forge.NoneEncoder{},
		forge.NoneDecoder{},
	)
}

// InsertL1PreinstallsViaForge inserts the shared preinstalls plus the L1-only request queues in the
// state of a genesis deployment, using Forge.
func InsertL1PreinstallsViaForge(env *ForgeEnv) error {
	if env.Genesis == nil {
		return errors.New("L1 preinstalls need the state of a genesis deployment")
	}
	if _, err := broadcastScript(env, NewSetPreinstallsForgeCaller, forge.None{}); err != nil {
		return fmt.Errorf("failed to set preinstalls with Forge: %w", err)
	}
	for addr, queue := range requestQueues {
		account := env.Genesis.account(addr)
		env.Genesis.Allocs.Accounts[addr] = requestQueueAccount(account, queue)
	}
	return nil
}
