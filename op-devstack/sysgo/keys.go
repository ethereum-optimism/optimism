package sysgo

import (
	"github.com/HashKeyChain/verse/op-chain-ops/devkeys"
	"github.com/HashKeyChain/verse/op-devstack/stack"
)

func WithMnemonicKeys(mnemonic string) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()
		hd, err := devkeys.NewMnemonicDevKeys(mnemonic)
		require.NoError(err)
		orch.keys = hd
	})
}
