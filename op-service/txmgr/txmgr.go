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
	"github.com/ethereum/go-ethereum/params"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
)

// Geth defaults the priceBump to 10
// Set it to 15% to be more aggressive about including transactions
const priceBump int64 = 15

// new = old * (100 + priceBump) / 100
var priceBumpPercent = big.NewInt(100 + priceBump)
var oneHundred = big.NewInt(100)

// UpdateGasPriceSendTxFunc defines a function signature for publishing a
// desired tx with a specific gas price. Implementations of this signature
// should also return promptly when the context is canceled.
type UpdateGasPriceFunc = func(ctx context.Context) (*types.Transaction, error)

type SendTransactionFunc = func(ctx context.Context, tx *types.Transaction) error

// Config houses parameters for altering the behavior of a SimpleTxManager.
type Config struct {
	// ResubmissionTimeout is the interval at which, if no previously
	// published transaction has been mined, the new tx with a bumped gas
	// price will be published. Only one publication at MaxGasPrice will be
	// attempted.
	ResubmissionTimeout time.Duration

	// ChainID is the chain ID of the L1 chain.
	ChainID *big.Int

	// NetworkTimeout is the allowed duration for a single network request.
	// This is intended to be used for network requests that can be replayed.
	//
	// If not set, this will default to 2 seconds.
	NetworkTimeout time.Duration

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

	// Signer is used to sign transactions when the gas price is increased.
	Signer opcrypto.SignerFn
	From   common.Address
}

// TxManager is an interface that allows callers to reliably publish txs,
// bumping the gas price if needed, and obtain the receipt of the resulting tx.
//
//go:generate mockery --name TxManager --output ./mocks
type TxManager interface {
	// Send is used to publish a transaction with incrementally higher gas
	// prices until the transaction eventually confirms. This method blocks
	// until an invocation of sendTx returns (called with differing gas
	// prices). The method may be canceled using the passed context.
	//
	// The initial transaction MUST be signed & ready to submit.
	//
	// NOTE: Send should be called by AT MOST one caller at a time.
	Send(ctx context.Context, tx *types.Transaction) (*types.Receipt, error)

	// CraftTx is used to craft a transaction using a [TxCandidate].
	CraftTx(ctx context.Context, candidate TxCandidate) (*types.Transaction, error)
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
	/// EstimateGas returns an estimate of the amount of gas needed to execute the given
	/// transaction against the current pending block.
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
}

// SimpleTxManager is a implementation of TxManager that performs linear fee
// bumping of a tx until it confirms.
type SimpleTxManager struct {
	Config  // embed the config directly
	name    string
	chainID *big.Int

	backend ETHBackend
	l       log.Logger
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

// calcGasTipAndFeeCap queries L1 to determine what a suitable miner tip & basefee limit would be for timely inclusion
func (m *SimpleTxManager) calcGasTipAndFeeCap(ctx context.Context) (gasTipCap *big.Int, gasFeeCap *big.Int, err error) {
	childCtx, cancel := context.WithTimeout(ctx, m.Config.NetworkTimeout)
	gasTipCap, err = m.backend.SuggestGasTipCap(childCtx)
	cancel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get suggested gas tip cap: %w", err)
	}

	if gasTipCap == nil {
		m.l.Warn("unexpected unset gasTipCap, using default 2 gwei")
		gasTipCap = new(big.Int).SetUint64(params.GWei * 2)
	}

	childCtx, cancel = context.WithTimeout(ctx, m.Config.NetworkTimeout)
	head, err := m.backend.HeaderByNumber(childCtx, nil)
	cancel()
	if err != nil || head == nil {
		return nil, nil, fmt.Errorf("failed to get L1 head block for fee cap: %w", err)
	}
	if head.BaseFee == nil {
		return nil, nil, fmt.Errorf("failed to get L1 basefee in block %d for fee cap", head.Number)
	}
	gasFeeCap = CalcGasFeeCap(head.BaseFee, gasTipCap)

	return gasTipCap, gasFeeCap, nil
}

// CraftTx creates the signed transaction to the batchInboxAddress.
// It queries L1 for the current fee market conditions as well as for the nonce.
// NOTE: This method SHOULD NOT publish the resulting transaction.
// NOTE: If the [TxCandidate.GasLimit] is non-zero, it will be used as the transaction's gas.
// NOTE: Otherwise, the [SimpleTxManager] will query the specified backend for an estimate.
func (m *SimpleTxManager) CraftTx(ctx context.Context, candidate TxCandidate) (*types.Transaction, error) {
	gasTipCap, gasFeeCap, err := m.calcGasTipAndFeeCap(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch the sender's nonce from the latest known block (nil `blockNumber`)
	childCtx, cancel := context.WithTimeout(ctx, m.Config.NetworkTimeout)
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

	ctx, cancel = context.WithTimeout(ctx, m.Config.NetworkTimeout)
	defer cancel()
	return m.Signer(ctx, candidate.From, types.NewTx(rawTx))
}

// IncreaseGasPrice takes the previous transaction & potentially clones then signs it with a higher tip.
// If the tip + basefee suggested by the network are not greater than the previous values, the same transaction
// will be returned. If they are greater, this function will ensure that they are at least greater by 15% than
// the previous transaction's value to ensure that the price bump is large enough.
//
// We do not re-estimate the amount of gas used because for some stateful transactions (like output proposals) the
// act of including the transaction renders the repeat of the transaction invalid.
func (m *SimpleTxManager) IncreaseGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var gasTipCap, gasFeeCap *big.Int

	if tip, err := m.backend.SuggestGasTipCap(ctx); err != nil {
		return nil, err
	} else if tip == nil {
		return nil, errors.New("the suggested tip was nil")
	} else {
		gasTipCap = tip
	}

	// Return the same transaction if we don't update any fields.
	// We do this because ethereum signatures are not deterministic and therefore the transaction hash will change
	// when we re-sign the tx. We don't want to do that because we want to see ErrAlreadyKnown instead of ErrReplacementUnderpriced
	var reusedTip, reusedFeeCap bool

	// new = old * (100 + priceBump) / 100
	// Enforce a min priceBump on the tip. Do this before the feeCap is calculated
	thresholdTip := new(big.Int).Mul(priceBumpPercent, tx.GasTipCap())
	thresholdTip = thresholdTip.Div(thresholdTip, oneHundred)
	if tx.GasTipCapIntCmp(gasTipCap) >= 0 {
		m.l.Debug("Reusing the previous tip", "previous", tx.GasTipCap(), "suggested", gasTipCap)
		gasTipCap = tx.GasTipCap()
		reusedTip = true
	} else if thresholdTip.Cmp(gasTipCap) > 0 {
		m.l.Debug("Overriding the tip to enforce a price bump", "previous", tx.GasTipCap(), "suggested", gasTipCap, "new", thresholdTip)
		gasTipCap = thresholdTip
	}

	if head, err := m.backend.HeaderByNumber(ctx, nil); err != nil {
		return nil, err
	} else if head.BaseFee == nil {
		return nil, errors.New("txmgr does not support pre-london blocks that do not have a basefee")
	} else {
		// CalcGasFeeCap ensure that the fee cap is large enough for the tip.
		gasFeeCap = CalcGasFeeCap(head.BaseFee, gasTipCap)
	}

	// new = old * (100 + priceBump) / 100
	// Enforce a min priceBump on the feeCap
	thresholdFeeCap := new(big.Int).Mul(priceBumpPercent, tx.GasFeeCap())
	thresholdFeeCap = thresholdFeeCap.Div(thresholdFeeCap, oneHundred)
	if tx.GasFeeCapIntCmp(gasFeeCap) >= 0 {
		if reusedTip {
			m.l.Debug("Reusing the previous fee cap", "previous", tx.GasFeeCap(), "suggested", gasFeeCap)
			gasFeeCap = tx.GasFeeCap()
			reusedFeeCap = true
		} else {
			m.l.Debug("Overriding the fee cap to enforce a price bump because we increased the tip", "previous", tx.GasFeeCap(), "suggested", gasFeeCap, "new", thresholdFeeCap)
			gasFeeCap = thresholdFeeCap
		}
	} else if thresholdFeeCap.Cmp(gasFeeCap) > 0 {
		if reusedTip {
			// TODO (CLI-3620): Increase the basefee then recompute the feecap
			m.l.Warn("Overriding the fee cap to enforce a price bump without increasing the tip. Will likely result in ErrReplacementUnderpriced",
				"previous", tx.GasFeeCap(), "suggested", gasFeeCap, "new", thresholdFeeCap)
		} else {
			m.l.Debug("Overriding the fee cap to enforce a price bump", "previous", tx.GasFeeCap(), "suggested", gasFeeCap, "new", thresholdFeeCap)
		}
		gasFeeCap = thresholdFeeCap
	}

	if reusedTip && reusedFeeCap {
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
	return m.Signer(ctx, m.From, types.NewTx(rawTx))
}

// NewSimpleTxManager initializes a new SimpleTxManager with the passed Config.
func NewSimpleTxManager(name string, l log.Logger, cfg Config, backend ETHBackend) *SimpleTxManager {
	if cfg.NumConfirmations == 0 {
		panic("txmgr: NumConfirmations cannot be zero")
	}
	if cfg.NetworkTimeout == 0 {
		cfg.NetworkTimeout = 2 * time.Second
	}

	return &SimpleTxManager{
		chainID: cfg.ChainID,
		name:    name,
		Config:  cfg,
		backend: backend,
		l:       l.New("service", name),
	}
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
func (m *SimpleTxManager) Send(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {

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

	sendState := NewSendState(m.SafeAbortNonceTooLowCount)

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

	ticker := time.NewTicker(m.ResubmissionTimeout)
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
	queryTicker := time.NewTicker(m.ReceiptQueryInterval)
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
				"tipHeight", tipHeight, "numConfirmations", m.NumConfirmations)

			// The transaction is considered confirmed when
			// txHeight+numConfirmations-1 <= tipHeight. Note that the -1 is
			// needed to account for the fact that confirmations have an
			// inherent off-by-one, i.e. when using 1 confirmation the
			// transaction should be confirmed when txHeight is equal to
			// tipHeight. The equation is rewritten in this form to avoid
			// underflows.
			if txHeight+m.NumConfirmations <= tipHeight+1 {
				m.l.Info("Transaction confirmed", "txHash", txHash)
				return receipt, nil
			}

			// Safe to subtract since we know the LHS above is greater.
			confsRemaining := (txHeight + m.NumConfirmations) - (tipHeight + 1)
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
