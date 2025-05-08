package sysgo

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
)

func WithMnemonicKeys(mnemonic string) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()
		hd, err := devkeys.NewMnemonicDevKeys(mnemonic)
		require.NoError(err)
		orch.keys = hd
	})
}
