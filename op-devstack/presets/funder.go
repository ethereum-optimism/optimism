package presets

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// newFunderEOA builds a prefunded funder EOA for a chain. It replaces the former
// op-faucet service + dsl.Funder pair: the funder's key is the genesis-prefunded
// funder account (sysgo.FunderKey), and it mints/funds child EOAs from the given
// test wallet so that funded accounts stay isolated per test.
func newFunderEOA(t devtest.T, keys devkeys.Keys, el dsl.ELNode, wallet *dsl.HDWallet) *dsl.EOA {
	fkey, err := sysgo.FunderKey(keys)
	t.Require().NoError(err, "must derive funder key")
	return dsl.NewFundingEOA(dsl.NewKey(t, fkey), el, wallet)
}
