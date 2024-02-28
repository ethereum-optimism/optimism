package l1el

import (
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	test "github.com/ethereum-optimism/optimism/op-test"
)

type L1EL interface {
	L1() l1.L1

	RPC() client.RPC
	L1Client() *sources.L1Client
}

func Request(t test.Testing, opts ...Option) L1EL {
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
