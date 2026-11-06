package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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

// TransactionSubmission is a node's answer to one transaction submitted over its RPC, for the
// test to judge: a refusal may be the behavior under test.
type TransactionSubmission struct {
	// Err is the node's refusal, nil when it accepted the transaction.
	Err error
	// Hash is what the node answered: the transaction's hash when it accepted it.
	Hash common.Hash
	// Method is the RPC method the transaction was submitted with.
	Method string
	Tx     *types.Transaction

	el *L2ELNode
}

// SendRawTransaction submits a signed transaction to the node's pool with eth_sendRawTransaction
// and returns the node's answer. SendTransactions is the variant that requires acceptance.
func (el *L2ELNode) SendRawTransaction(tx *types.Transaction) *TransactionSubmission {
	return el.submitRawTransaction(tx, "eth_sendRawTransaction")
}

// TransactionConditional is the condition eth_sendRawTransactionConditional admits a transaction
// under: inclusive bounds on the number and timestamp of the block that includes it.
type TransactionConditional struct {
	BlockNumberMax *hexutil.Uint64 `json:"blockNumberMax,omitempty"`
	BlockNumberMin *hexutil.Uint64 `json:"blockNumberMin,omitempty"`
	TimestampMax   *hexutil.Uint64 `json:"timestampMax,omitempty"`
	TimestampMin   *hexutil.Uint64 `json:"timestampMin,omitempty"`
}

// SendRawTransactionConditional submits a signed transaction with eth_sendRawTransactionConditional,
// which a node admits only while it can still meet the condition, and returns the node's answer.
func (el *L2ELNode) SendRawTransactionConditional(tx *types.Transaction, condition TransactionConditional) *TransactionSubmission {
	// The condition's account state part is always sent as an empty object: op-reth refuses a
	// null where it expects the map.
	wire := struct {
		TransactionConditional
		KnownAccounts struct{} `json:"knownAccounts"`
	}{TransactionConditional: condition}
	return el.submitRawTransaction(tx, "eth_sendRawTransactionConditional", wire)
}

func (el *L2ELNode) submitRawTransaction(tx *types.Transaction, method string, extraArgs ...any) *TransactionSubmission {
	raw, err := tx.MarshalBinary()
	el.require.NoError(err, "must encode transaction %s", tx.Hash())
	el.log.Info("Submitting a transaction", "node", el, "method", method, "tx", tx.Hash(), "tip", tx.GasTipCap(), "fee_cap", tx.GasFeeCap())
	submission := &TransactionSubmission{Method: method, Tx: tx, el: el}
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	submission.Err = el.rpc().CallContext(ctx, &submission.Hash, method, append([]any{hexutil.Bytes(raw)}, extraArgs...)...)
	if submission.Err != nil {
		el.log.Info("Transaction refused", "node", el, "method", method, "tx", tx.Hash(), "err", submission.Err)
	} else {
		el.log.Info("Transaction accepted", "node", el, "method", method, "tx", tx.Hash(), "hash", submission.Hash)
	}
	return submission
}

// IsAccepted requires the node to have taken the transaction, answering with its hash.
func (s *TransactionSubmission) IsAccepted() *TransactionSubmission {
	s.el.require.NoErrorf(s.Err, "%s must accept transaction %s on %s", s.Method, s.Tx.Hash(), s.el)
	s.el.require.Equal(s.Tx.Hash(), s.Hash, "%s must answer with the transaction's hash", s.Method)
	return s
}

// IsRejectedWith requires the node to have refused the transaction with the given RPC error code
// and message.
func (s *TransactionSubmission) IsRejectedWith(code eth.ErrorCode, message string) *TransactionSubmission {
	s.el.require.Errorf(s.Err, "%s must refuse transaction %s on %s", s.Method, s.Tx.Hash(), s.el)
	var rpcErr rpc.Error
	s.el.require.ErrorAs(s.Err, &rpcErr, "expected an RPC error code from %s, got: %v", s.Method, s.Err)
	s.el.require.Equal(code, eth.ErrorCode(rpcErr.ErrorCode()), "unexpected error code from %s: %v", s.Method, s.Err)
	s.el.require.EqualError(rpcErr, message, "unexpected error message from %s", s.Method)
	return s
}

// lookupTransaction asks the node for the transaction: whether it knows it at all, and the number
// of the block holding it once it is in one.
func (el *L2ELNode) lookupTransaction(hash common.Hash) (known bool, inBlock *hexutil.Big, err error) {
	var tx *struct {
		BlockNumber *hexutil.Big `json:"blockNumber"`
	}
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	if err := el.rpc().CallContext(ctx, &tx, "eth_getTransactionByHash", hash); err != nil {
		return false, nil, fmt.Errorf("looking up transaction %s on %s: %w", hash, el, err)
	}
	if tx == nil {
		return false, nil, nil
	}
	return true, tx.BlockNumber, nil
}

// WaitForPooledTransaction waits until the node holds the transaction in its pool: known to the
// node and not yet in a block. A transaction the node first sees in a block fails the wait.
func (el *L2ELNode) WaitForPooledTransaction(hash common.Hash) {
	el.log.Info("Waiting for a transaction to be pooled", "node", el, "tx", hash)
	var inBlock *hexutil.Big
	err := retry.Do0(el.ctx, 30, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		known, blockNumber, err := el.lookupTransaction(hash)
		if err != nil {
			el.log.Warn("Transaction lookup failed; will retry", "node", el, "tx", hash, "err", err)
			return err
		}
		if !known {
			el.log.Info("Transaction not known yet", "node", el, "tx", hash)
			return fmt.Errorf("transaction %s is not known to %s yet", hash, el)
		}
		inBlock = blockNumber
		return nil
	})
	el.require.NoError(err, "transaction %s did not reach the pool of %s", hash, el)
	el.require.Nilf(inBlock, "transaction %s was first seen in block %s on %s, not in its pool", hash, inBlock, el)
	el.log.Info("Transaction is pooled", "node", el, "tx", hash)
}

// VerifyTransactionUnknown requires the node to know nothing of the transaction, pooled or
// included. A transaction the node refused never becomes known to it, so one look suffices.
func (el *L2ELNode) VerifyTransactionUnknown(hash common.Hash) {
	el.log.Info("Expecting a transaction to be unknown", "node", el, "tx", hash)
	known, inBlock, err := el.lookupTransaction(hash)
	el.require.NoError(err)
	el.require.Falsef(known, "transaction %s must be unknown to %s, but it holds it (in block %s)", hash, el, inBlock)
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
