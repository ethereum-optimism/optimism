package helpers

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// firstValidTx finds the first transaction that is valid for inclusion from the specified address.
// It uses a waiter and filtering of already included transactions to avoid race conditions with the async
// updates to the transaction pool.
func firstValidTx(
	t Testing,
	from common.Address,
	pendingIndices func(common.Address) uint64,
	contentFrom func(common.Address) ([]*types.Transaction, []*types.Transaction),
	nonceAt func(context.Context, common.Address, *big.Int) (uint64, error),
) *types.Transaction {
	var i uint64
	var txs []*types.Transaction
	var q []*types.Transaction
	// Wait for the tx to be in the pending tx queue
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		i = pendingIndices(from)
		txs, q = contentFrom(from)
		// Remove any transactions that have already been included in the head block
		// The tx pool only prunes included transactions async so they may still be in the list
		nonce, err := nonceAt(ctx, from, nil)
		if err != nil {
			return false, err
		}
		for len(txs) > 0 && txs[0].Nonce() < nonce {
			t.Logf("Removing already included transaction from list of length %v", len(txs))
			txs = txs[1:]
		}
		return uint64(len(txs)) > i, nil
	})
	require.NoError(t, err,
		"no pending txs from %s, and have %d unprocessable queued txs from this account: %w", from, len(q), err)

	return txs[i]
}
