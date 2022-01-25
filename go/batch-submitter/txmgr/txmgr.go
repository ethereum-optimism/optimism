package txmgr

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// ErrPublishTimeout signals that the tx manager did not receive a confirmation
// for a given tx after publishing with the maximum gas price and waiting out a
// resubmission timeout.
var ErrPublishTimeout = errors.New("failed to publish tx with max gas price")

// SendTxFunc defines a function signature for publishing a desired tx with a
// specific gas price. Implementations of this signature should also return
// promptly when the context is canceled.
type SendTxFunc = func(
	ctx context.Context, gasPrice *big.Int) (*types.Transaction, error)

// Config houses parameters for altering the behavior of a SimpleTxManager.
type Config struct {
	Name string

	// MinGasPrice is the minimum gas price (in gwei). This is used as the
	// initial publication attempt.
	MinGasPrice *big.Int

	// MaxGasPrice is the maximum gas price (in gwei). This is used to clamp
	// the upper end of the range that the TxManager will ever publish when
	// attempting to confirm a transaction.
	MaxGasPrice *big.Int

	// GasRetryIncrement is the additive gas price (in gwei) that will be
	// used to bump each successive tx after a ResubmissionTimeout has
	// elapsed.
	GasRetryIncrement *big.Int

	// ResubmissionTimeout is the interval at which, if no previously
	// published transaction has been mined, the new tx with a bumped gas
	// price will be published. Only one publication at MaxGasPrice will be
	// attempted.
	ResubmissionTimeout time.Duration

	// RequireQueryInterval is the interval at which the tx manager will
	// query the backend to check for confirmations after a tx at a
	// specific gas price has been published.
	ReceiptQueryInterval time.Duration

	// NumConfirmations specifies how many blocks are need to consider a
	// transaction confirmed.
	NumConfirmations uint64
}

// TxManager is an interface that allows callers to reliably publish txs,
// bumping the gas price if needed, and obtain the receipt of the resulting tx.
type TxManager interface {
	// Send is used to publish a transaction with incrementally higher gas
	// prices until the transaction eventually confirms. This method blocks
	// until an invocation of sendTx returns (called with differing gas
	// prices). The method may be canceled using the passed context.
	//
	// NOTE: Send should be called by AT MOST one caller at a time.
	Send(ctx context.Context, sendTx SendTxFunc) (*types.Receipt, error)
}

// ReceiptSource is a minimal function signature used to detect the confirmation
// of published txs.
//
// NOTE: This is a subset of bind.DeployBackend.
type ReceiptSource interface {
	// BlockNumber returns the most recent block number.
	BlockNumber(ctx context.Context) (uint64, error)

	// TransactionReceipt queries the backend for a receipt associated with
	// txHash. If lookup does not fail, but the transaction is not found,
	// nil should be returned for both values.
	TransactionReceipt(
		ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// SimpleTxManager is a implementation of TxManager that performs linear fee
// bumping of a tx until it confirms.
type SimpleTxManager struct {
	name    string
	cfg     Config
	backend ReceiptSource
}

// NewSimpleTxManager initializes a new SimpleTxManager with the passed Config.
func NewSimpleTxManager(
	name string, cfg Config, backend ReceiptSource) *SimpleTxManager {

	if cfg.NumConfirmations == 0 {
		panic("txmgr: NumConfirmations cannot be zero")
	}

	return &SimpleTxManager{
		name:    name,
		cfg:     cfg,
		backend: backend,
	}
}

// Send is used to publish a transaction with incrementally higher gas prices
// until the transaction eventually confirms. This method blocks until an
// invocation of sendTx returns (called with differing gas prices). The method
// may be canceled using the passed context.
//
// NOTE: Send should be called by AT MOST one caller at a time.
func (m *SimpleTxManager) Send(
	ctx context.Context, sendTx SendTxFunc) (*types.Receipt, error) {

	name := m.name

	// Initialize a wait group to track any spawned goroutines, and ensure
	// we properly clean up any dangling resources this method generates.
	// We assert that this is the case thoroughly in our unit tests.
	var wg sync.WaitGroup
	defer wg.Wait()

	// Initialize a subcontext for the goroutines spawned in this process.
	// The defer to cancel is done here (in reverse order of Wait) so that
	// the goroutines can exit before blocking on the wait group.
	ctxc, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a closure that will block on passed sendTx function in the
	// background, returning the first successfully mined receipt back to
	// the main event loop via receiptChan.
	receiptChan := make(chan *types.Receipt, 1)
	sendTxAsync := func(gasPrice *big.Int) {
		defer wg.Done()

		// Sign and publish transaction with current gas price.
		tx, err := sendTx(ctxc, gasPrice)
		if err != nil {
			if err == context.Canceled ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			log.Error(name+" unable to publish transaction",
				"gas_price", gasPrice, "err", err)
			// TODO(conner): add retry?
			return
		}

		txHash := tx.Hash()
		log.Info(name+" transaction published successfully", "hash", txHash,
			"gas_price", gasPrice)

		// Wait for the transaction to be mined, reporting the receipt
		// back to the main event loop if found.
		receipt, err := WaitMined(
			ctxc, m.backend, tx, m.cfg.ReceiptQueryInterval,
			m.cfg.NumConfirmations,
		)
		if err != nil {
			log.Debug(name+" send tx failed", "hash", txHash,
				"gas_price", gasPrice, "err", err)
		}
		if receipt != nil {
			// Use non-blocking select to ensure function can exit
			// if more than one receipt is discovered.
			select {
			case receiptChan <- receipt:
				log.Trace(name+" send tx succeeded", "hash", txHash,
					"gas_price", gasPrice)
			default:
			}
		}
	}

	// Initialize our initial gas price to the configured minimum.
	curGasPrice := new(big.Int).Set(m.cfg.MinGasPrice)

	// Submit and wait for the receipt at our first gas price in the
	// background, before entering the event loop and waiting out the
	// resubmission timeout.
	wg.Add(1)
	go sendTxAsync(curGasPrice)

	for {
		select {

		// Whenever a resubmission timeout has elapsed, bump the gas
		// price and publish a new transaction.
		case <-time.After(m.cfg.ResubmissionTimeout):
			// If our last attempt published at the max gas price,
			// return an error as we are unlikely to succeed in
			// publishing. This also indicates that the max gas
			// price should likely be adjusted higher for the
			// daemon.
			if curGasPrice.Cmp(m.cfg.MaxGasPrice) >= 0 {
				return nil, ErrPublishTimeout
			}

			// Bump the gas price using linear gas price increments.
			curGasPrice = NextGasPrice(
				curGasPrice, m.cfg.GasRetryIncrement,
				m.cfg.MaxGasPrice,
			)

			// Submit and wait for the bumped traction to confirm.
			wg.Add(1)
			go sendTxAsync(curGasPrice)

		// The passed context has been canceled, i.e. in the event of a
		// shutdown.
		case <-ctxc.Done():
			return nil, ctxc.Err()

		// The transaction has confirmed.
		case receipt := <-receiptChan:
			return receipt, nil
		}
	}
}

// WaitMined blocks until the backend indicates confirmation of tx and returns
// the tx receipt. Queries are made every queryInterval, regardless of whether
// the backend returns an error. This method can be canceled using the passed
// context.
func WaitMined(
	ctx context.Context,
	backend ReceiptSource,
	tx *types.Transaction,
	queryInterval time.Duration,
	numConfirmations uint64,
) (*types.Receipt, error) {

	queryTicker := time.NewTicker(queryInterval)
	defer queryTicker.Stop()

	txHash := tx.Hash()

	for {
		receipt, err := backend.TransactionReceipt(ctx, txHash)
		switch {
		case receipt != nil:
			txHeight := receipt.BlockNumber.Uint64()
			tipHeight, err := backend.BlockNumber(ctx)
			if err != nil {
				log.Error("Unable to fetch block number", "err", err)
				break
			}

			log.Trace("Transaction mined, checking confirmations",
				"txHash", txHash, "txHeight", txHeight,
				"tipHeight", tipHeight,
				"numConfirmations", numConfirmations)

			// The transaction is considered confirmed when
			// txHeight+numConfirmations-1 <= tipHeight. Note that the -1 is
			// needed to account for the fact that confirmations have an
			// inherent off-by-one, i.e. when using 1 confirmation the
			// transaction should be confirmed when txHeight is equal to
			// tipHeight. The equation is rewritten in this form to avoid
			// underflows.
			if txHeight+numConfirmations <= tipHeight+1 {
				log.Info("Transaction confirmed", "txHash", txHash)
				return receipt, nil
			}

			// Safe to subtract since we know the LHS above is greater.
			confsRemaining := (txHeight + numConfirmations) - (tipHeight + 1)
			log.Info("Transaction not yet confirmed", "txHash", txHash,
				"confsRemaining", confsRemaining)

		case err != nil:
			log.Trace("Receipt retrievel failed", "hash", txHash,
				"err", err)

		default:
			log.Trace("Transaction not yet mined", "hash", txHash)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

// NextGasPrice bumps the current gas price using an additive gasRetryIncrement,
// clamping the resulting value to maxGasPrice.
//
// NOTE: This method does not mutate curGasPrice, but instead returns a copy.
// This removes the possiblity of races occuring from goroutines sharing access
// to the same underlying big.Int.
func NextGasPrice(curGasPrice, gasRetryIncrement, maxGasPrice *big.Int) *big.Int {
	nextGasPrice := new(big.Int).Set(curGasPrice)
	nextGasPrice.Add(nextGasPrice, gasRetryIncrement)
	if nextGasPrice.Cmp(maxGasPrice) == 1 {
		nextGasPrice.Set(maxGasPrice)
	}
	return nextGasPrice
}
