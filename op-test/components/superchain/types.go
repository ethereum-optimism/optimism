package superchain

import (
	"github.com/stretchr/testify/require"

	test "github.com/ethereum-optimism/optimism/op-test"
)

type Superchain interface {
	// Members()
	// L1ContractAddrs()
}

func Request(t test.Testing, opts ...Option) Superchain {
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
