package superchain

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/ethereum-optimism/optimism/op-test/test"
)

type Superchain interface {
	L1() l1.L1
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
