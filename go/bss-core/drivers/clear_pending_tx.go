package drivers

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/go/bss-core/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// ErrClearPendingRetry signals that a transaction from a previous running
// instance confirmed rather than our clearing transaction on startup. In this
// case the caller should retry.
var ErrClearPendingRetry = errors.New("retry clear pending txn")

// ClearPendingTx publishes a NOOP transaction at the wallet's next unused
// nonce. This is used on restarts in order to clear the mempool of any prior
// publications and ensure the batch submitter starts submitting from a clean
// slate.
func ClearPendingTx(
	name string,
	ctx context.Context,
	txMgr txmgr.TxManager,
	l1Client L1Client,
	walletAddr common.Address,
	privKey *ecdsa.PrivateKey,
	chainID *big.Int,
) error {

	// Query for the submitter's current nonce.
	nonce, err := l1Client.NonceAt(ctx, walletAddr, nil)
	if err != nil {
		log.Error(name+" unable to get current nonce",
			"err", err)
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Construct the clearing transaction submission clousure that will attempt
	// to send the a clearing transaction transaction at the given nonce and gas
	// price.
	sendTx := func(
		ctx context.Context,
	) (*types.Transaction, error) {
		log.Info(name+" clearing pending tx", "nonce", nonce)

		signedTx, err := SignClearingTx(
			name, ctx, walletAddr, nonce, l1Client, privKey, chainID,
		)
		if err != nil {
			log.Error(name+" unable to sign clearing tx", "nonce", nonce,
				"err", err)
			return nil, err
		}
		txHash := signedTx.Hash()
		gasTipCap := signedTx.GasTipCap()
		gasFeeCap := signedTx.GasFeeCap()

		err = l1Client.SendTransaction(ctx, signedTx)
		switch {

		// Clearing transaction successfully confirmed.
		case err == nil:
			log.Info(name+" submitted clearing tx", "nonce", nonce,
				"gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap,
				"txHash", txHash)

			return signedTx, nil

		// Getting a nonce too low error implies that a previous transaction in
		// the mempool has confirmed and we should abort trying to publish at
		// this nonce.
		case strings.Contains(err.Error(), core.ErrNonceTooLow.Error()):
			log.Info(name + " transaction from previous restart confirmed, " +
				"aborting mempool clearing")
			cancel()
			return nil, context.Canceled

		// An unexpected error occurred. This also handles the case where the
		// clearing transaction has not yet bested the gas price a prior
		// transaction in the mempool at this nonce. In such a case we will
		// continue until our ratchetting strategy overtakes the old
		// transaction, or abort if the old one confirms.
		default:
			log.Error(name+" unable to submit clearing tx",
				"nonce", nonce, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap,
				"txHash", txHash, "err", err)
			return nil, err
		}
	}

	receipt, err := txMgr.Send(ctx, sendTx)
	switch {

	// If the current context is canceled, a prior transaction in the mempool
	// confirmed. The caller should retry, which will use the next nonce, before
	// proceeding.
	case err == context.Canceled:
		log.Info(name + " transaction from previous restart confirmed, " +
			"proceeding to startup")
		return ErrClearPendingRetry

	// Otherwise we were unable to confirm our transaction, this method should
	// be retried by the caller.
	case err != nil:
		log.Warn(name+" unable to send clearing tx", "nonce", nonce,
			"err", err)
		return err

	// We succeeded in confirming a clearing transaction. Proceed to startup as
	// normal.
	default:
		log.Info(name+" cleared pending tx", "nonce", nonce,
			"txHash", receipt.TxHash)
		return nil
	}
}

// SignClearingTx creates a signed clearing tranaction which sends 0 ETH back to
// the sender's address. EstimateGas is used to set an appropriate gas limit.
func SignClearingTx(
	name string,
	ctx context.Context,
	walletAddr common.Address,
	nonce uint64,
	l1Client L1Client,
	privKey *ecdsa.PrivateKey,
	chainID *big.Int,
) (*types.Transaction, error) {

	gasTipCap, err := l1Client.SuggestGasTipCap(ctx)
	if err != nil {
		if !IsMaxPriorityFeePerGasNotFoundError(err) {
			return nil, err
		}

		// If the transaction failed because the backend does not support
		// eth_maxPriorityFeePerGas, fallback to using the default constant.
		// Currently Alchemy is the only backend provider that exposes this
		// method, so in the event their API is unreachable we can fallback to a
		// degraded mode of operation. This also applies to our test
		// environments, as hardhat doesn't support the query either.
		log.Warn(name + " eth_maxPriorityFeePerGas is unsupported " +
			"by current backend, using fallback gasTipCap")
		gasTipCap = FallbackGasTipCap
	}

	head, err := l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	gasLimit, err := l1Client.EstimateGas(ctx, ethereum.CallMsg{
		From:      walletAddr,
		To:        &walletAddr,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     nil,
		Data:      nil,
	})
	if err != nil {
		return nil, err
	}

	tx := CraftClearingTx(walletAddr, nonce, gasFeeCap, gasTipCap, gasLimit)

	return types.SignTx(
		tx, types.LatestSignerForChainID(chainID), privKey,
	)
}

// CraftClearingTx creates an unsigned clearing transaction which sends 0 ETH
// back to the sender's address.
func CraftClearingTx(
	walletAddr common.Address,
	nonce uint64,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	gasLimit uint64,
) *types.Transaction {

	return types.NewTx(&types.DynamicFeeTx{
		To:        &walletAddr,
		Nonce:     nonce,
		Gas:       gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     nil,
		Data:      nil,
	})
}
