package l2

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"math/big"

	test "github.com/ethereum-optimism/optimism/op-test"
)

type L2 interface {
	ChainID() *big.Int
	ChainConfig() *params.ChainConfig
	RollupConfig() *rollup.Config
	Name() string
	L1ContractAddrs() struct{} // TODO

	// Fund an account, if not already funded. Abstracts away test-account funding.
	Fund(addr common.Address, amount *big.Int)

	// Lock the chain for breaking changes
	Lock()
	Unlock()
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
