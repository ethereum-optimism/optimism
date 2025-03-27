package systemgo

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
)

func WithMnemonicKeys(mnemonic string) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)
		hd, err := devkeys.NewMnemonicDevKeys(mnemonic)
		setup.Require.NoError(err)
		orch.keys = hd
	}
}
