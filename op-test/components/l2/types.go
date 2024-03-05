package l2

import (
	"math/big"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/superchain"
)

type L2 interface {
	ChainID() *big.Int
	ChainConfig() *params.ChainConfig
	RollupConfig() *rollup.Config
	Genesis() *core.Genesis
	Name() string
	L1Deployments() *genesis.L1Deployments

	Superchain() superchain.Superchain
}

func Request(t test.Testing, opts ...Option) L2 {
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
