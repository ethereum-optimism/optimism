package sysgo

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
)

func WithMnemonicKeys(mnemonic string) stack.Option {
	return func(setup *stack.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)
		hd, err := devkeys.NewMnemonicDevKeys(mnemonic)
		setup.Require.NoError(err)
		orch.keys = hd
	}
}
