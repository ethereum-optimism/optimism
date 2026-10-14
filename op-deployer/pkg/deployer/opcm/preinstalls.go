package opcm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
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
	queues := types.GenesisAlloc{
		params.WithdrawalQueueAddress:    {Code: params.WithdrawalQueueCode},
		params.ConsolidationQueueAddress: {Code: params.ConsolidationQueueCode},
		params.BuilderDepositAddress:     {Code: params.BuilderDepositCode},
		params.BuilderExitAddress:        {Code: params.BuilderExitCode},
	}
	dump, err := host.StateDump()
	if err != nil {
		return fmt.Errorf("failed to read L1 state for request queues: %w", err)
	}
	for addr, queue := range queues {
		// Retain prefunded balances and any existing storage/nonces.
		account := dump.Accounts[addr]
		account.Code = queue.Code
		if account.Nonce == 0 {
			account.Nonce = 1
		}
		host.ImportAccount(addr, account)
	}
	return nil
}
