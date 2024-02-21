package l2cl

import (
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/components/l2el"
	"github.com/stretchr/testify/require"

	test "github.com/ethereum-optimism/optimism/op-test"
)

type L2CL interface {
	L2EL() l2el.L2EL
	L2() l2.L2

	RollupClient() *sources.RollupClient

	// TODO sequencer actions (requires L2-chain lock)
}

func Request(t test.Testing, opts ...Option) L2CL {
	var settings Settings
	for i, opt := range opts {
		require.NoError(t, opt.Apply(&settings), "must apply option %d", i)
	}
	switch settings.Kind {
	case test.Live:
		// TODO
	}
	return nil
}
