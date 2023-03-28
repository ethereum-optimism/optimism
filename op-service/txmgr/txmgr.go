package txmgr

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Geth defaults the priceBump to 10
// Set it to 15% to be more aggressive about including transactions
const priceBump int64 = 15

// new = old * (100 + priceBump) / 100
var priceBumpPercent = big.NewInt(100 + priceBump)
var oneHundred = big.NewInt(100)

// TxManager is an interface that allows callers to reliably publish txs,
// bumping the gas price if needed, and obtain the receipt of the resulting tx.
//
//go:generate mockery --name TxManager --output ./mocks
type TxManager interface {
	// Send is used to create & send a transaction. It will handle increasing
	// the gas price & ensuring that the transaction remains in the transaction pool.
	// It can be stopped by cancelling the provided context; however, the transaction
	// may be included on L1 even if the context is cancelled.
	//
	// NOTE: Send should be called by AT MOST one caller at a time.
	Send(ctx context.Context, candidate TxCandidate) (*types.Receipt, error)

	// From returns the sending address associated with the instance of the transaction manager.
	// It is static for a single instance of a TxManager.
	From() common.Address
}

// ETHBackend is the set of methods that the transaction manager uses to resubmit gas & determine
// when transactions are included on L1.
type ETHBackend interface {
	// BlockNumber returns the most recent block number.
	BlockNumber(ctx context.Context) (uint64, error)

	// TransactionReceipt queries the backend for a receipt associated with
	// txHash. If lookup does not fail, but the transaction is not found,
	// nil should be returned for both values.
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)

	// SendTransaction submits a signed transaction to L1.
	SendTransaction(ctx context.Context, tx *types.Transaction) error

	// These functions are used to estimate what the basefee & priority fee should be set to.
	// TODO(CLI-3318): Maybe need a generic interface to support different RPC providers
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	// NonceAt returns the account nonce of the given account.
	// The block number can be nil, in which case the nonce is taken from the latest known block.
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	// PendingNonce returns the pending nonce.
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	/// EstimateGas returns an estimate of the amount of gas needed to execute the given
	/// transaction against the current pending block.
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
}

// SimpleTxManager is a implementation of TxManager that performs linear fee
// bumping of a tx until it confirms.
type SimpleTxManager struct {
	cfg     Config // embed the config directly
	name    string
	chainID *big.Int

	backend ETHBackend
	l       log.Logger
}

// NewSimpleTxManager initializes a new SimpleTxManager with the passed Config.
func NewSimpleTxManager(name string, l log.Logger, cfg Config) *SimpleTxManager {
	if cfg.NumConfirmations == 0 {
		panic("txmgr: NumConfirmations cannot be zero")
	}
	if cfg.NetworkTimeout == 0 {
		cfg.NetworkTimeout = 2 * time.Second
	}

	return &SimpleTxManager{
		chainID: cfg.ChainID,
		name:    name,
		cfg:     cfg,
		backend: cfg.Backend,
		l:       l.New("service", name),
	}
}

func (m *SimpleTxManager) From() common.Address {
	return m.cfg.From
}

// TxCandidate is a transaction candidate that can be submitted to ask the
// [TxManager] to construct a transaction with gas price bounds.
type TxCandidate struct {
	// TxData is the transaction data to be used in the constructed tx.
	TxData []byte
	// To is the recipient of the constructed tx.
	To common.Address
	// GasLimit is the gas limit to be used in the constructed tx.
	GasLimit uint64
	// From is the sender (or `from`) of the constructed tx.
	From common.Address
}

// Send is used to publish a transaction with incrementally higher gas prices
// until the transaction eventually confirms. This method blocks until an
// invocation of sendTx returns (called with differing gas prices). The method
// may be canceled using the passed context.
//
// The initially supplied transaction must be signed, have gas estimation done, and have a reasonable gas fee.
// When the transaction is resubmitted the tx manager will re-sign the transaction at a different gas pricing
// but retain the gas used, the nonce, and the data.
//
// NOTE: Send should be called by AT MOST one caller at a time.
func (m *SimpleTxManager) Send(ctx context.Context, candidate TxCandidate) (*types.Receipt, error) {
	tx, err := m.craftTx(ctx, candidate)
	if err != nil {
		m.l.Error("Failed to create the transaction", "err", err)
		return nil, err
	}
	return m.send(ctx, tx)
}

// craftTx creates the signed transaction
// It queries L1 for the current fee market conditions as well as for the nonce.
// NOTE: This method SHOULD NOT publish the resulting transaction.
// NOTE: If the [TxCandidate.GasLimit] is non-zero, it will be used as the transaction's gas.
// NOTE: Otherwise, the [SimpleTxManager] will query the specified backend for an estimate.
func (m *SimpleTxManager) craftTx(ctx context.Context, candidate TxCandidate) (*types.Transaction, error) {
	gasTipCap, basefee, err := m.suggestGasPriceCaps(ctx)
	if err != nil {
		return nil, err
	}
	gasFeeCap := calcGasFeeCap(basefee, gasTipCap)

	// Fetch the sender's nonce from the latest known block (nil `blockNumber`)
	childCtx, cancel := context.WithTimeout(ctx, m.cfg.NetworkTimeout)
	defer cancel()
	nonce, err := m.backend.NonceAt(childCtx, candidate.From, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	rawTx := &types.DynamicFeeTx{
		ChainID:   m.chainID,
		Nonce:     nonce,
		To:        &candidate.To,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      candidate.TxData,
	}

	m.l.Info("creating tx", "to", rawTx.To, "from", candidate.From)

	// If the gas limit is set, we can use that as the gas
	if candidate.GasLimit != 0 {
		rawTx.Gas = candidate.GasLimit
	} else {
		// Calculate the intrinsic gas for the transaction
		gas, err := m.backend.EstimateGas(ctx, ethereum.CallMsg{
			From:      candidate.From,
			To:        &candidate.To,
			GasFeeCap: gasFeeCap,
			GasTipCap: gasTipCap,
			Data:      rawTx.Data,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}
		rawTx.Gas = gas
	}

	ctx, cancel = context.WithTimeout(ctx, m.cfg.NetworkTimeout)
	defer cancel()
	return m.cfg.Signer(ctx, candidate.From, types.NewTx(rawTx))
}

func (m *SimpleTxManager) send(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	// Initialize a wait group to track any spawned goroutines, and ensure
	// we properly clean up any dangling resources this method generates.
	// We assert that this is the case thoroughly in our unit tests.
	var wg sync.WaitGroup
	defer wg.Wait()

	// Initialize a subcontext for the goroutines spawned in this process.
	// The defer to cancel is done here (in reverse order of Wait) so that
	// the goroutines can exit before blocking on the wait group.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sendState := NewSendState(m.cfg.SafeAbortNonceTooLowCount)

	// Create a closure that will block on submitting the tx in the
	// background, returning the first successfully mined receipt back to
	// the main event loop via receiptChan.
	receiptChan := make(chan *types.Receipt, 1)
	sendTxAsync := func(tx *types.Transaction) {
		defer wg.Done()

		txHash := tx.Hash()
		nonce := tx.Nonce()
		gasTipCap := tx.GasTipCap()
		gasFeeCap := tx.GasFeeCap()
		log := m.l.New("txHash", txHash, "nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)
		log.Info("publishing transaction")

		err := m.backend.SendTransaction(ctx, tx)
		sendState.ProcessSendError(err)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			if errors.Is(err, txpool.ErrAlreadyKnown) {
				log.Info("resubmitted already known transaction")
				return
			}
			log.Error("unable to publish transaction", "err", err)
			if sendState.ShouldAbortImmediately() {
				log.Warn("Aborting transaction submission")
				cancel()
			}
			// TODO(conner): add retry?
			return
		}

		log.Info("transaction published successfully")

		// Wait for the transaction to be mined, reporting the receipt
		// back to the main event loop if found.
		receipt, err := m.waitMined(ctx, tx, sendState)
		if err != nil {
			log.Debug("send tx failed", "err", err)
		}
		if receipt != nil {
			// Use non-blocking select to ensure function can exit
			// if more than one receipt is discovered.
			select {
			case receiptChan <- receipt:
				log.Trace("send tx succeeded")
			default:
			}
		}
	}

	// Submit and wait for the receipt at our first gas price in the
	// background, before entering the event loop and waiting out the
	// resubmission timeout.
	wg.Add(1)
	go sendTxAsync(tx)

	ticker := time.NewTicker(m.cfg.ResubmissionTimeout)
	defer ticker.Stop()

	for {
		select {

		// Whenever a resubmission timeout has elapsed, bump the gas
		// price and publish a new transaction.
		case <-ticker.C:
			// Avoid republishing if we are waiting for confirmation on an
			// existing tx. This is primarily an optimization to reduce the
			// number of API calls we make, but also reduces the chances of
			// getting a false positive reading for ShouldAbortImmediately.
			if sendState.IsWaitingForConfirmation() {
				continue
			}

			// Increase the gas price & submit the new transaction
			newTx, err := m.IncreaseGasPrice(ctx, tx)
			if err != nil {
				m.l.Error("Failed to increase the gas price for the tx", "err", err)
				// Don't `continue` here so we resubmit the transaction with the same gas price.
			} else {
				// Save the tx so we know it's gas price.
				tx = newTx
			}
			wg.Add(1)
			go sendTxAsync(tx)

		// The passed context has been canceled, i.e. in the event of a
		// shutdown.
		case <-ctx.Done():
			return nil, ctx.Err()

		// The transaction has confirmed.
		case receipt := <-receiptChan:
			return receipt, nil
		}
	}
}

// waitMined implements the core functionality of WaitMined, with the option to
// pass in a SendState to record whether or not the transaction is mined.
func (m *SimpleTxManager) waitMined(ctx context.Context, tx *types.Transaction, sendState *SendState) (*types.Receipt, error) {
	queryTicker := time.NewTicker(m.cfg.ReceiptQueryInterval)
	defer queryTicker.Stop()

	txHash := tx.Hash()

	for {
		receipt, err := m.backend.TransactionReceipt(ctx, txHash)
		switch {
		case receipt != nil:
			if sendState != nil {
				sendState.TxMined(txHash)
			}

			txHeight := receipt.BlockNumber.Uint64()
			tipHeight, err := m.backend.BlockNumber(ctx)
			if err != nil {
				m.l.Error("Unable to fetch block number", "err", err)
				break
			}

			m.l.Debug("Transaction mined, checking confirmations", "txHash", txHash, "txHeight", txHeight,
				"tipHeight", tipHeight, "numConfirmations", m.cfg.NumConfirmations)

			// The transaction is considered confirmed when
			// txHeight+numConfirmations-1 <= tipHeight. Note that the -1 is
			// needed to account for the fact that confirmations have an
			// inherent off-by-one, i.e. when using 1 confirmation the
			// transaction should be confirmed when txHeight is equal to
			// tipHeight. The equation is rewritten in this form to avoid
			// underflows.
			if txHeight+m.cfg.NumConfirmations <= tipHeight+1 {
				m.l.Info("Transaction confirmed", "txHash", txHash)
				return receipt, nil
			}

			// Safe to subtract since we know the LHS above is greater.
			confsRemaining := (txHeight + m.cfg.NumConfirmations) - (tipHeight + 1)
			m.l.Debug("Transaction not yet confirmed", "txHash", txHash, "confsRemaining", confsRemaining)

		case err != nil:
			m.l.Trace("Receipt retrievel failed", "hash", txHash, "err", err)

		default:
			if sendState != nil {
				sendState.TxNotMined(txHash)
			}
			m.l.Trace("Transaction not yet mined", "hash", txHash)
		}

		select {
		case <-ctx.Done():
			m.l.Warn("context cancelled in waitMined")
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

// suggestGasPriceCaps suggests what the new tip & new basefee should be based on the current L1 conditions
func (m *SimpleTxManager) suggestGasPriceCaps(ctx context.Context) (*big.Int, *big.Int, error) {
	cCtx, cancel := context.WithTimeout(ctx, m.cfg.NetworkTimeout)
	defer cancel()
	tip, err := m.backend.SuggestGasTipCap(cCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch the suggested gas tip cap: %w", err)
	} else if tip == nil {
		return nil, nil, errors.New("the suggested tip was nil")
	}
	cCtx, cancel = context.WithTimeout(ctx, m.cfg.NetworkTimeout)
	defer cancel()
	head, err := m.backend.HeaderByNumber(cCtx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch the suggested basefee: %w", err)
	} else if head.BaseFee == nil {
		return nil, nil, errors.New("txmgr does not support pre-london blocks that do not have a basefee")
	}
	return tip, head.BaseFee, nil
}

// IncreaseGasPrice takes the previous transaction & potentially clones then signs it with a higher tip.
// If the tip + basefee suggested by the network are not greater than the previous values, the same transaction
// will be returned. If they are greater, this function will ensure that they are at least greater by 15% than
// the previous transaction's value to ensure that the price bump is large enough.
//
// We do not re-estimate the amount of gas used because for some stateful transactions (like output proposals) the
// act of including the transaction renders the repeat of the transaction invalid.
func (m *SimpleTxManager) IncreaseGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	tip, basefee, err := m.suggestGasPriceCaps(ctx)
	if err != nil {
		return nil, err
	}
	gasTipCap, gasFeeCap := updateFees(tx.GasTipCap(), tx.GasFeeCap(), tip, basefee, m.l)

	if tx.GasTipCapIntCmp(gasTipCap) == 0 && tx.GasFeeCapIntCmp(gasFeeCap) == 0 {
		return tx, nil
	}

	rawTx := &types.DynamicFeeTx{
		ChainID:    tx.ChainId(),
		Nonce:      tx.Nonce(),
		GasTipCap:  gasTipCap,
		GasFeeCap:  gasFeeCap,
		Gas:        tx.Gas(),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
	}
	return m.cfg.Signer(ctx, m.cfg.From, types.NewTx(rawTx))
}

// calcThresholdValue returns x * priceBumpPercent / 100
func calcThresholdValue(x *big.Int) *big.Int {
	threshold := new(big.Int).Mul(priceBumpPercent, x)
	threshold = threshold.Div(threshold, oneHundred)
	return threshold
}

// updateFees takes the old tip/basefee & the new tip/basefee and then suggests
// a gasTipCap and gasFeeCap that satisfies geth's required fee bumps
// Geth: FC and Tip must be bumped if any increase
func updateFees(oldTip, oldFeeCap, newTip, newBaseFee *big.Int, lgr log.Logger) (*big.Int, *big.Int) {
	newFeeCap := calcGasFeeCap(newBaseFee, newTip)
	lgr = lgr.New("old_tip", oldTip, "old_feecap", oldFeeCap, "new_tip", newTip, "new_feecap", newFeeCap)
	// If the new prices are less than the old price, reuse the old prices
	if oldTip.Cmp(newTip) >= 0 && oldFeeCap.Cmp(newFeeCap) >= 0 {
		lgr.Debug("Reusing old tip and feecap")
		return oldTip, oldFeeCap
	}
	// Determine if we need to increase the suggested values
	thresholdTip := calcThresholdValue(oldTip)
	thresholdFeeCap := calcThresholdValue(oldFeeCap)
	if newTip.Cmp(thresholdTip) >= 0 && newFeeCap.Cmp(thresholdFeeCap) >= 0 {
		lgr.Debug("Using new tip and feecap")
		return newTip, newFeeCap
	} else if newTip.Cmp(thresholdTip) >= 0 && newFeeCap.Cmp(thresholdFeeCap) < 0 {
		// Tip has gone up, but basefee is flat or down.
		// TODO(CLI-3714): Do we need to recalculate the FC here?
		lgr.Debug("Using new tip and threshold feecap")
		return newTip, thresholdFeeCap
	} else if newTip.Cmp(thresholdTip) < 0 && newFeeCap.Cmp(thresholdFeeCap) >= 0 {
		// Basefee has gone up, but the tip hasn't. Recalculate the feecap because if the tip went up a lot
		// not enough of the feecap may be dedicated to paying the basefee.
		lgr.Debug("Using threshold tip and recalculated feecap")
		return thresholdTip, calcGasFeeCap(newBaseFee, thresholdTip)

	} else {
		// TODO(CLI-3713): Should we skip the bump in this case?
		lgr.Debug("Using threshold tip and threshold feecap")
		return thresholdTip, thresholdFeeCap
	}
}

// calcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to:
//
//	gasTipCap + 2*baseFee.
func calcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
}
