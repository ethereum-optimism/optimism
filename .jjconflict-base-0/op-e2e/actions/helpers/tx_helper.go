package helpers

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"

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
	ctx, cancel := context.WithTimeout(context.Background(), 31*time.Second)
	defer cancel()

	err := retry.Do0(ctx, 10, retry.Exponential(), func() error {
		i = pendingIndices(from)
		txs, q = contentFrom(from)
		// Remove any transactions that have already been included in the head block
		// The tx pool only prunes included transactions async so they may still be in the list

		subCtx, subCancel := context.WithTimeout(ctx, time.Second)
		defer subCancel()

		nonce, err := nonceAt(subCtx, from, nil)
		if err != nil {
			return err
		}
		for len(txs) > 0 && txs[0].Nonce() < nonce {
			t.Logf("Removing already included transaction from list of length %v", len(txs))
			txs = txs[1:]
		}

		if uint64(len(txs)) <= i {
			return fmt.Errorf("no pending txs from %s, and have %d unprocessable queued txs from this account", from, len(q))
		}

		return nil
	})
	require.NoError(t, err)

	return txs[i]
}
