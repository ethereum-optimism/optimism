package l2el

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/test"
)

type L2EL interface {
	L2() l2.L2

	HTTPAuthEndpoint() string
	WSAuthEndpoint() string
	JWTSecret() [32]byte

	WSEndpoint() string
	HTTPEndpoint() string

	// RPC is a direct client, with all RPC namespaces exposed.
	RPC() *rpc.Client
}

func Request(t test.Testing, opts ...Option) L2EL {
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
