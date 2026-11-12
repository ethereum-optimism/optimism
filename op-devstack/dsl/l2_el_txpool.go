package dsl

import (
	"context"
	"fmt"
	"maps"
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
	// txPoolContentBatchSize bounds one txpool_contentFrom batch call. Its answer for senders of
	// 4 KiB transactions stays a few MB, well inside the client's 32 MB websocket read limit.
	txPoolContentBatchSize = 512
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
	return el.submitRawTransaction(tx, "eth_sendRawTransactionConditional", condition)
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

// WaitForPooledTransaction waits, polling once a second up to attempts times, until the node
// holds the transaction in its pool: known to the node and not yet in a block. A transaction the
// node first sees in a block fails the wait.
func (el *L2ELNode) WaitForPooledTransaction(hash common.Hash, attempts int) {
	el.log.Info("Waiting for a transaction to be pooled", "node", el, "tx", hash, "attempts", attempts)
	var inBlock *hexutil.Big
	err := retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
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
	where := "pooled"
	if inBlock != nil {
		where = fmt.Sprintf("in block %s", inBlock)
	}
	el.require.Falsef(known, "transaction %s must be unknown to %s, but it holds it (%s)", hash, el, where)
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

// transactionHashes returns the set of the hashes of txs.
func transactionHashes(txs []*types.Transaction) map[common.Hash]struct{} {
	hashes := make(map[common.Hash]struct{}, len(txs))
	for _, tx := range txs {
		hashes[tx.Hash()] = struct{}{}
	}
	return hashes
}

// collectIncludedTransactions adds to included the hashes of those of wanted that the node's
// canonical block number includes.
func (el *L2ELNode) collectIncludedTransactions(wanted, included map[common.Hash]struct{}, number uint64) error {
	el.log.Debug("Fetching block transactions", "node", el, "block", number)
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	_, blockTxs, err := el.EthClient().InfoAndTxsByNumber(ctx, number)
	cancel()
	if err != nil {
		el.log.Error("Failed to fetch block transactions", "node", el, "block", number, "err", err)
		return fmt.Errorf("fetching block %d from %s: %w", number, el, err)
	}
	var includedInBlock int
	for _, tx := range blockTxs {
		if _, ok := wanted[tx.Hash()]; ok {
			included[tx.Hash()] = struct{}{}
			includedInBlock++
		}
	}
	el.log.Debug("Fetched block transactions", "node", el, "block", number, "transactions", len(blockTxs), "included", includedInBlock)
	return nil
}

// IncludedTransactions returns the hashes of those of txs that the node's canonical blocks from
// fromBlock through toBlock include.
func (el *L2ELNode) IncludedTransactions(txs []*types.Transaction, fromBlock, toBlock uint64) map[common.Hash]struct{} {
	el.log.Info("Collecting included transactions", "node", el, "count", len(txs), "from", fromBlock, "to", toBlock)
	wanted := transactionHashes(txs)
	included := make(map[common.Hash]struct{})
	for number := fromBlock; number <= toBlock; number++ {
		el.require.NoError(el.collectIncludedTransactions(wanted, included, number))
	}
	el.log.Info("Collected included transactions", "node", el, "count", len(txs), "from", fromBlock, "to", toBlock, "included", len(included))
	return included
}

// WaitForIncludedTransactions waits until the node's canonical blocks from fromBlock up to its
// unsafe head include at least count of txs, and returns the hashes of those of txs the blocks it
// read include, with the number of the first block it did not read. It reads every block once,
// so a reorg of a block it has read goes unnoticed.
func (el *L2ELNode) WaitForIncludedTransactions(txs []*types.Transaction, fromBlock uint64, count int, attempts int) (map[common.Hash]struct{}, uint64) {
	el.log.Info("Waiting for transactions to be included", "node", el, "count", count, "of", len(txs), "from", fromBlock, "attempts", attempts)
	wanted := transactionHashes(txs)
	included := make(map[common.Hash]struct{})
	nextBlock := fromBlock
	err := retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		head, err := el.blockRefByLabel(eth.Unsafe)
		if err != nil {
			el.log.Warn("Unsafe head lookup failed; will retry", "node", el, "err", err)
			return err
		}
		for ; nextBlock <= head.Number; nextBlock++ {
			if err := el.collectIncludedTransactions(wanted, included, nextBlock); err != nil {
				return err
			}
		}
		if len(included) < count {
			el.log.Info("Not enough transactions included yet", "node", el, "included", len(included), "count", count, "head", head.Number)
			return fmt.Errorf("blocks %d through %d of %s include %d of the transactions, expecting %d", fromBlock, head.Number, el, len(included), count)
		}
		return nil
	})
	el.require.NoError(err, "%s did not include %d of the transactions", el, count)
	el.log.Info("Transactions included", "node", el, "included", len(included), "count", count, "through", nextBlock-1)
	return included, nextBlock
}

// pooledTransaction is the part of a txpool_contentFrom entry that identifies the transaction.
type pooledTransaction struct {
	Hash common.Hash `json:"hash"`
}

// senderPoolContent is a txpool_contentFrom answer: one sender's pooled transactions by nonce.
type senderPoolContent struct {
	Pending map[string]pooledTransaction `json:"pending"`
	Queued  map[string]pooledTransaction `json:"queued"`
}

// txPoolLookup is a set of transactions to look up in a pool by sender.
type txPoolLookup struct {
	hashes map[common.Hash]struct{}
	// senders are the distinct senders of the transactions, in the order they first sign one.
	senders []common.Address
}

func newTxPoolLookup(txs []*types.Transaction) (txPoolLookup, error) {
	lookup := txPoolLookup{hashes: transactionHashes(txs)}
	seen := make(map[common.Address]struct{}, len(txs))
	for _, tx := range txs {
		sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		if err != nil {
			return txPoolLookup{}, fmt.Errorf("recovering the sender of transaction %s: %w", tx.Hash(), err)
		}
		if _, ok := seen[sender]; !ok {
			seen[sender] = struct{}{}
			lookup.senders = append(lookup.senders, sender)
		}
	}
	return lookup, nil
}

// withSenders returns the lookup narrowed to senders.
func (lookup txPoolLookup) withSenders(senders []common.Address) txPoolLookup {
	return txPoolLookup{hashes: lookup.hashes, senders: senders}
}

// poolHoldings is what a node's pool holds of a lookup's transactions.
type poolHoldings struct {
	// holders are the senders of the lookup with at least one of its transactions in the pool.
	holders []common.Address

	pending map[common.Hash]struct{}
	queued  map[common.Hash]struct{}
}

// all returns the hashes of the held transactions, pending or queued.
func (holdings poolHoldings) all() map[common.Hash]struct{} {
	all := maps.Clone(holdings.pending)
	maps.Copy(all, holdings.queued)
	return all
}

// pooledTransactions reads what the node's pool holds right now of the lookup's transactions,
// from the pool contents of the lookup's senders, in batch calls.
func (el *L2ELNode) pooledTransactions(lookup txPoolLookup) (poolHoldings, error) {
	holdings := poolHoldings{pending: make(map[common.Hash]struct{}), queued: make(map[common.Hash]struct{})}
	for start := 0; start < len(lookup.senders); start += txPoolContentBatchSize {
		batch := lookup.senders[start:min(start+txPoolContentBatchSize, len(lookup.senders))]
		contents := make([]senderPoolContent, len(batch))
		calls := make([]rpc.BatchElem, len(batch))
		for i, sender := range batch {
			calls[i] = rpc.BatchElem{Method: "txpool_contentFrom", Args: []any{sender}, Result: &contents[i]}
		}
		el.log.Debug("Reading the pool contents of senders", "node", el, "from_index", start, "senders", len(batch))
		ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
		err := el.rpc().BatchCallContext(ctx, calls)
		cancel()
		if err != nil {
			el.log.Error("Failed to read the pool contents of senders", "node", el, "from_index", start, "senders", len(batch), "err", err)
			return poolHoldings{}, fmt.Errorf("reading the pool contents of %d senders from index %d on %s: %w", len(batch), start, el, err)
		}
		for i, call := range calls {
			if call.Error != nil {
				el.log.Error("Failed to read the pool contents of a sender", "node", el, "sender", batch[i], "err", call.Error)
				return poolHoldings{}, fmt.Errorf("reading the pool contents of sender %s on %s: %w", batch[i], el, call.Error)
			}
			pendingFound := collectWanted(lookup.hashes, contents[i].Pending, holdings.pending)
			queuedFound := collectWanted(lookup.hashes, contents[i].Queued, holdings.queued)
			if pendingFound || queuedFound {
				holdings.holders = append(holdings.holders, batch[i])
			}
		}
		el.log.Debug("Read the pool contents of senders", "node", el, "from_index", start, "senders", len(batch))
	}
	return holdings, nil
}

// collectWanted adds to found the hashes of those of subPool's transactions that are in wanted,
// and reports whether there was any.
func collectWanted(wanted map[common.Hash]struct{}, subPool map[string]pooledTransaction, found map[common.Hash]struct{}) bool {
	var foundAny bool
	for _, tx := range subPool {
		if _, ok := wanted[tx.Hash]; ok {
			found[tx.Hash] = struct{}{}
			foundAny = true
		}
	}
	return foundAny
}

// PooledTransactions returns the hashes of those of txs that the node's pool holds right now,
// pending or queued.
func (el *L2ELNode) PooledTransactions(txs []*types.Transaction) map[common.Hash]struct{} {
	el.log.Info("Reading pooled transactions", "node", el, "count", len(txs))
	lookup, err := newTxPoolLookup(txs)
	el.require.NoError(err)
	holdings, err := el.pooledTransactions(lookup)
	el.require.NoError(err)
	pooled := holdings.all()
	el.log.Info("Read pooled transactions", "node", el, "count", len(txs), "pooled", len(pooled))
	return pooled
}

// WaitForPooledTransactions waits, polling once a second up to attempts times, until the node's
// pool holds at least count of txs, pending or queued, and returns the hashes of those it holds
// then. It reads the pool contents only once the pool holds count transactions in all.
func (el *L2ELNode) WaitForPooledTransactions(txs []*types.Transaction, count int, attempts int) map[common.Hash]struct{} {
	el.log.Info("Waiting for transactions to be pooled", "node", el, "count", count, "of", len(txs), "attempts", attempts)
	lookup, err := newTxPoolLookup(txs)
	el.require.NoError(err)
	var pooled map[common.Hash]struct{}
	err = retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		status, err := el.txPoolStatus()
		if err != nil {
			el.log.Warn("Pool status lookup failed; will retry", "node", el, "err", err)
			return err
		}
		if status.Total() < uint64(count) {
			el.log.Info("Pool holds too few transactions yet", "node", el, "count", count, "status", status)
			return fmt.Errorf("the pool of %s holds %s, expecting at least %d of the transactions", el, status, count)
		}
		holdings, err := el.pooledTransactions(lookup)
		if err != nil {
			el.log.Warn("Pool contents lookup failed; will retry", "node", el, "err", err)
			return err
		}
		pooled = holdings.all()
		if len(pooled) < count {
			el.log.Info("Not enough transactions pooled yet", "node", el, "pooled", len(pooled), "count", count)
			return fmt.Errorf("the pool of %s holds %d of the transactions, expecting %d", el, len(pooled), count)
		}
		return nil
	})
	el.require.NoError(err, "the pool of %s did not take %d of the transactions", el, count)
	el.log.Info("Transactions pooled", "node", el, "pooled", len(pooled), "count", count)
	return pooled
}

// WaitForPendingTransactions waits, polling once a second up to attempts times, until the node's
// pool holds every one of txs as pending: ready for the next block, not queued behind anything.
// It reads the pool contents only once the pool holds that many pending transactions in all.
func (el *L2ELNode) WaitForPendingTransactions(txs []*types.Transaction, attempts int) {
	el.log.Info("Waiting for transactions to be pending", "node", el, "count", len(txs), "attempts", attempts)
	lookup, err := newTxPoolLookup(txs)
	el.require.NoError(err)
	wanted := len(lookup.hashes)
	err = retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		status, err := el.txPoolStatus()
		if err != nil {
			el.log.Warn("Pool status lookup failed; will retry", "node", el, "err", err)
			return err
		}
		if status.Pending < uint64(wanted) {
			el.log.Info("Pool holds too few pending transactions yet", "node", el, "count", wanted, "status", status)
			return fmt.Errorf("the pool of %s holds %s, expecting at least %d pending", el, status, wanted)
		}
		holdings, err := el.pooledTransactions(lookup)
		if err != nil {
			el.log.Warn("Pool contents lookup failed; will retry", "node", el, "err", err)
			return err
		}
		if len(holdings.pending) < wanted {
			el.log.Info("Not every transaction is pending yet", "node", el, "pending", len(holdings.pending), "queued", len(holdings.queued), "count", wanted)
			return fmt.Errorf("the pool of %s holds %d of %d transactions as pending", el, len(holdings.pending), wanted)
		}
		return nil
	})
	el.require.NoError(err, "the pool of %s did not hold all %d transactions as pending", el, wanted)
	el.log.Info("Transactions pending", "node", el, "count", wanted)
}

// WaitForTransactionsOutOfPool waits until the node's pool holds none of txs. A node removes the
// transactions of a block it imported from its pool asynchronously to the import. Each poll reads
// only the senders that still held one of txs on the previous poll; once none does, one more
// poll reads every sender, since a transaction can return to the pool.
func (el *L2ELNode) WaitForTransactionsOutOfPool(txs []*types.Transaction, attempts int) {
	el.log.Info("Waiting for transactions to leave the pool", "node", el, "count", len(txs), "attempts", attempts)
	everySender, err := newTxPoolLookup(txs)
	el.require.NoError(err)
	lookup, readEverySender := everySender, true
	err = retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: txPoolPollInterval}, func() error {
		holdings, err := el.pooledTransactions(lookup)
		if err != nil {
			el.log.Warn("Pool contents lookup failed; will retry", "node", el, "err", err)
			return err
		}
		pooled := len(holdings.pending) + len(holdings.queued)
		switch {
		case pooled > 0:
			el.log.Info("Transactions still pooled", "node", el, "count", len(txs), "pooled", pooled)
			lookup, readEverySender = everySender.withSenders(holdings.holders), false
			return fmt.Errorf("the pool of %s still holds %d of %d transactions", el, pooled, len(txs))
		case !readEverySender:
			el.log.Info("The remaining senders let go of the transactions; reading every sender", "node", el, "count", len(txs))
			lookup, readEverySender = everySender, true
			return fmt.Errorf("the pool of %s has not been read for every sender of the transactions yet", el)
		default:
			return nil
		}
	})
	el.require.NoError(err, "the pool of %s did not let go of the transactions", el)
	el.log.Info("Transactions left the pool", "node", el, "count", len(txs))
}
