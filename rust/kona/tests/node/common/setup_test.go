package node

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

func newCommonPreset(t devtest.T) *node_utils.MixedOpKonaPreset {
	t.Helper()
	return node_utils.NewMixedOpKona(t)
}
