//go:build !ci

package upgrade

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

func newSimpleInterop(t devtest.T) *presets.SimpleInterop {
	offset := uint64(60)
	return presets.NewSimpleInterop(t, presets.WithDeployerOptions(
		func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtOffset(forks.Interop, &offset)
			}
		},
	))
}
