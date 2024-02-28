package l2proposer

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l2cl"
	"github.com/stretchr/testify/require"

	test "github.com/ethereum-optimism/optimism/op-test"
)

type L2Proposer interface {
	L2CL() l2cl.L2CL

	// TODO admin access to proposer operations
}

func Request(t test.Testing, opts ...Option) L2Proposer {
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
