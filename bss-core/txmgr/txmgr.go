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

// ErrReverted signals that a mined transaction reverted.
var ErrReverted = errors.New("transaction reverted")

// UpdateGasPriceSendTxFunc defines a function signature for publishing a
// desired tx with a specific gas price. Implementations of this signature
// should also return promptly when the context is canceled.
type UpdateGasPriceFunc = func(ctx context.Context) (*types.Transaction, error)

type SendTransactionFunc = func(ctx context.Context, tx *types.Transaction) error

// Config houses parameters for altering the behavior of a SimpleTxManager.
type Config struct {
	// Name the name of the driver to appear in log lines.
	Name string

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

	// SafeAbortNonceTooLowCount specifies how many ErrNonceTooLow observations
	// are required to give up on a tx at a particular nonce without receiving
	// confirmation.
	SafeAbortNonceTooLowCount uint64
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
	Send(
		ctx context.Context,
		updateGasPrice UpdateGasPriceFunc,
		sendTxn SendTransactionFunc,
	) (*types.Receipt, error)
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
	ctx context.Context,
	updateGasPrice UpdateGasPriceFunc,
	sendTx SendTransactionFunc,
) (*types.Receipt, error) {

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

	sendState := NewSendState(m.cfg.SafeAbortNonceTooLowCount)

	// Create a closure that will block on passed sendTx function in the
	// background, returning the first successfully mined receipt back to
	// the main event loop via receiptChan.
	receiptChan := make(chan *types.Receipt, 1)
	sendTxAsync := func() {
		defer wg.Done()

		tx, err := updateGasPrice(ctxc)
		if err != nil {
			if err == context.Canceled ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			log.Error(name+" unable to update txn gas price", "err", err)
			return
		}

		txHash := tx.Hash()
		nonce := tx.Nonce()
		gasTipCap := tx.GasTipCap()
		gasFeeCap := tx.GasFeeCap()
		log.Info(name+" publishing transaction", "txHash", txHash,
			"nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)

		// Sign and publish transaction with current gas price.
		err = sendTx(ctxc, tx)
		sendState.ProcessSendError(err)
		if err != nil {
			if err == context.Canceled ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			log.Error(name+" unable to publish transaction", "err", err)
			if sendState.ShouldAbortImmediately() {
				cancel()
			}
			// TODO(conner): add retry?
			return
		}

		log.Info(name+" transaction published successfully", "hash", txHash,
			"nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)

		// Wait for the transaction to be mined, reporting the receipt
		// back to the main event loop if found.
		receipt, err := waitMined(
			ctxc, m.backend, tx, m.cfg.ReceiptQueryInterval,
			m.cfg.NumConfirmations, sendState,
		)
		if err != nil {
			log.Debug(name+" send tx failed", "hash", txHash,
				"nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap,
				"err", err)
		}
		if receipt != nil {
			// Use non-blocking select to ensure function can exit
			// if more than one receipt is discovered.
			select {
			case receiptChan <- receipt:
				log.Trace(name+" send tx succeeded", "hash", txHash,
					"nonce", nonce, "gasTipCap", gasTipCap,
					"gasFeeCap", gasFeeCap)
			default:
			}
		}
	}

	// Submit and wait for the receipt at our first gas price in the
	// background, before entering the event loop and waiting out the
	// resubmission timeout.
	wg.Add(1)
	go sendTxAsync()

	tick := time.NewTicker(m.cfg.ResubmissionTimeout)

	for {
		select {

		// Whenever a resubmission timeout has elapsed, bump the gas
		// price and publish a new transaction.
		case <-tick.C:
			// Avoid republishing if we are waiting for confirmation on an
			// existing tx. This is primarily an optimization to reduce the
			// number of API calls we make, but also reduces the chances of
			// getting a false positive reading for ShouldAbortImmediately.
			if sendState.IsWaitingForConfirmation() {
				continue
			}

			// Submit and wait for the bumped traction to confirm.
			wg.Add(1)
			go sendTxAsync()

		// The passed context has been canceled, i.e. in the event of a
		// shutdown.
		case <-ctxc.Done():
			return nil, ctxc.Err()

		// The transaction has confirmed.
		case receipt := <-receiptChan:
			if receipt.Status == types.ReceiptStatusFailed {
				return receipt, ErrReverted
			}
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
	return waitMined(ctx, backend, tx, queryInterval, numConfirmations, nil)
}

// waitMined implements the core functionality of WaitMined, with the option to
// pass in a SendState to record whether or not the transaction is mined.
func waitMined(
	ctx context.Context,
	backend ReceiptSource,
	tx *types.Transaction,
	queryInterval time.Duration,
	numConfirmations uint64,
	sendState *SendState,
) (*types.Receipt, error) {

	queryTicker := time.NewTicker(queryInterval)
	defer queryTicker.Stop()

	txHash := tx.Hash()

	for {
		receipt, err := backend.TransactionReceipt(ctx, txHash)
		switch {
		case receipt != nil:
			if sendState != nil {
				sendState.TxMined(txHash)
			}

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
				reverted := receipt.Status == types.ReceiptStatusFailed
				log.Info("Transaction confirmed",
					"txHash", txHash,
					"reverted", reverted)
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
			if sendState != nil {
				sendState.TxNotMined(txHash)
			}
			log.Trace("Transaction not yet mined", "hash", txHash)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

// CalcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to:
//
//	gasTipCap + 2*baseFee.
func CalcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
}
