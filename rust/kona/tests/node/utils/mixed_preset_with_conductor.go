package node_utils

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

type MinimalWithConductors = presets.MinimalWithConductors

func NewMixedOpKonaWithConductors(t devtest.T) *MinimalWithConductors {
	return presets.NewMinimalWithConductors(t)
}

func NewMixedOpKonaWithConductorsForConfig(t devtest.T, _ L2NodeConfig, opts ...presets.Option) *MinimalWithConductors {
	return presets.NewMinimalWithConductors(t, opts...)
}
