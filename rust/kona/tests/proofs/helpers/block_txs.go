package helpers

import (
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// BlockTxs returns the deposits and the user transactions of the L2 block with the given hash.
// Post-exec (0x7D) transactions are in neither list.
func BlockTxs(t helpers.Testing, engine *helpers.L2Engine, hash common.Hash) ([]*optypes.DepositTx, types.Transactions) {
	cl := engine.SourceClient()
	_, deposits, err := cl.InfoAndDepositsByHash(t.Ctx(), hash)
	require.NoError(t, err)
	_, userTxs, err := cl.InfoAndUserTxsByHash(t.Ctx(), hash)
	require.NoError(t, err)
	return deposits, userTxs
}
