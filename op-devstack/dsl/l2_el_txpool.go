package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

const (
	// sendTransactionsBatchSize bounds one eth_sendRawTransaction batch call. It stays under
	// op-geth's default limit of 1000 calls per batch, and keeps a batch of 4 KiB transactions
	// well inside both clients' request size limits.
	sendTransactionsBatchSize = 512
	// sendTransactionsTimeout bounds one batch call: a node validates every transaction of the
	// batch before it answers.
	sendTransactionsTimeout = time.Minute
	// txPoolPollInterval is how often the pool waits look at the pool.
	txPoolPollInterval = time.Second
)

// SendTransactions submits signed transactions to the node's pool in batch calls, back to back,
// and fails on the first one the node refuses. It does not wait for inclusion.
func (el *L2ELNode) SendTransactions(txs []*types.Transaction) {
	el.log.Info("Sending transactions", "node", el, "count", len(txs))
	for start := 0; start < len(txs); start += sendTransactionsBatchSize {
		batch := txs[start:min(start+sendTransactionsBatchSize, len(txs))]
		calls := make([]rpc.BatchElem, len(batch))
		for i, tx := range batch {
			raw, err := tx.MarshalBinary()
			el.require.NoError(err, "must encode transaction %s", tx.Hash())
			calls[i] = rpc.BatchElem{Method: "eth_sendRawTransaction", Args: []any{hexutil.Bytes(raw)}, Result: new(common.Hash)}
		}
		ctx, cancel := context.WithTimeout(el.ctx, sendTransactionsTimeout)
		err := el.rpc().BatchCallContext(ctx, calls)
		cancel()
		el.require.NoError(err, "the batch of %d transactions from index %d must be sent to %s", len(batch), start, el)
		for i, call := range calls {
			el.require.NoError(call.Error, "transaction %s (index %d of %d) must be accepted by %s", batch[i].Hash(), start+i, len(txs), el)
		}
	}
	el.log.Info("Sent transactions", "node", el, "count", len(txs))
}

// WaitForPooledTransaction waits until the node holds the transaction in its pool: known to the
// node and not yet in a block. A transaction the node first sees in a block fails the wait.
func (el *L2ELNode) WaitForPooledTransaction(hash common.Hash) {
	el.log.Info("Waiting for a transaction to be pooled", "node", el, "tx", hash)
	var inBlock *hexutil.Big
	err := retry.Do0(el.ctx, 30, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		var tx *struct {
			BlockNumber *hexutil.Big `json:"blockNumber"`
		}
		ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
		defer cancel()
		if err := el.rpc().CallContext(ctx, &tx, "eth_getTransactionByHash", hash); err != nil {
			el.log.Warn("Transaction lookup failed; will retry", "node", el, "tx", hash, "err", err)
			return fmt.Errorf("looking up transaction %s on %s: %w", hash, el, err)
		}
		if tx == nil {
			el.log.Info("Transaction not known yet", "node", el, "tx", hash)
			return fmt.Errorf("transaction %s is not known to %s yet", hash, el)
		}
		inBlock = tx.BlockNumber
		return nil
	})
	el.require.NoError(err, "transaction %s did not reach the pool of %s", hash, el)
	el.require.Nilf(inBlock, "transaction %s was first seen in block %s on %s, not in its pool", hash, inBlock, el)
	el.log.Info("Transaction is pooled", "node", el, "tx", hash)
}

// TxPoolStatus is how many transactions a node's pool holds.
type TxPoolStatus struct {
	// Pending transactions can go into the next block.
	Pending uint64
	// Queued transactions wait for something, mostly an earlier nonce, before they can.
	Queued uint64
}

func (s TxPoolStatus) Total() uint64 {
	return s.Pending + s.Queued
}

func (s TxPoolStatus) String() string {
	return fmt.Sprintf("%d pending, %d queued", s.Pending, s.Queued)
}

func (el *L2ELNode) txPoolStatus() (TxPoolStatus, error) {
	var status struct {
		Pending hexutil.Uint64 `json:"pending"`
		Queued  hexutil.Uint64 `json:"queued"`
	}
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	if err := el.rpc().CallContext(ctx, &status, "txpool_status"); err != nil {
		return TxPoolStatus{}, fmt.Errorf("reading the pool status of %s: %w", el, err)
	}
	return TxPoolStatus{Pending: uint64(status.Pending), Queued: uint64(status.Queued)}, nil
}

// TxPoolStatus reads how many transactions the node's pool holds right now.
func (el *L2ELNode) TxPoolStatus() TxPoolStatus {
	status, err := el.txPoolStatus()
	el.require.NoError(err)
	return status
}

// txPoolWait polls the pool once a second, up to attempts times, until its status satisfies
// expectation, and returns that status.
func (el *L2ELNode) txPoolWait(attempts int, expectation string, satisfied func(TxPoolStatus) bool) TxPoolStatus {
	el.log.Info("Waiting for the transaction pool", "node", el, "expectation", expectation, "attempts", attempts)
	var status TxPoolStatus
	err := retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		var err error
		status, err = el.txPoolStatus()
		if err != nil {
			el.log.Warn("Pool status lookup failed; will retry", "node", el, "err", err)
			return err
		}
		if !satisfied(status) {
			el.log.Info("Pool not there yet", "node", el, "expectation", expectation, "status", status)
			return fmt.Errorf("the pool of %s holds %s, expecting %s", el, status, expectation)
		}
		return nil
	})
	el.require.NoError(err, "the pool of %s did not reach the expectation", el)
	el.log.Info("Transaction pool reached the expectation", "node", el, "expectation", expectation, "status", status)
	return status
}

// TxPoolReached waits until the pool holds at least count transactions, pending and queued
// together, and returns the status that met the count.
func (el *L2ELNode) TxPoolReached(count uint64, attempts int) TxPoolStatus {
	return el.txPoolWait(attempts, fmt.Sprintf("at least %d transactions", count), func(s TxPoolStatus) bool { return s.Total() >= count })
}

// TxPoolHoldsAtMost waits until the pool holds at most count transactions, pending and queued
// together, and returns the status that met the count.
func (el *L2ELNode) TxPoolHoldsAtMost(count uint64, attempts int) TxPoolStatus {
	return el.txPoolWait(attempts, fmt.Sprintf("at most %d transactions", count), func(s TxPoolStatus) bool { return s.Total() <= count })
}

// TxPoolDrained waits until the pool holds no transaction.
func (el *L2ELNode) TxPoolDrained(attempts int) {
	el.TxPoolHoldsAtMost(0, attempts)
}
