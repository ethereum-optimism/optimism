package l1cl

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
	"github.com/stretchr/testify/require"

	test "github.com/ethereum-optimism/optimism/op-test"
)

type L1CL interface {
	EL() l1el.L1EL
	Chain() l1.L1

	// controls below should require the L1 chain to have a Lock
	// TODO time travel (gap in chain to skip ahead to block with future timestamp)
	// TODO block building controls
}

func Request(t test.Testing, opts ...Option) L1CL {
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
