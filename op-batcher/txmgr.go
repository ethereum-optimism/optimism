package op_batcher

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const networkTimeout = 2 * time.Second // How long a single network request can take. TODO: put in a config somewhere

// TransactionManager wraps the simple txmgr package to make it easy to send & wait for transactions
type TransactionManager struct {
	// Config
	batchInboxAddress common.Address
	senderAddress     common.Address
	chainID           *big.Int
	// Outside world
	txMgr    txmgr.TxManager
	l1Client *ethclient.Client
	signerFn func(types.TxData) (*types.Transaction, error)
	log      log.Logger
}

func NewTransactionManager(log log.Logger, txMgrConfg txmgr.Config, batchInboxAddress common.Address, chainID *big.Int, privKey *ecdsa.PrivateKey, l1Client *ethclient.Client) *TransactionManager {
	signerFn := func(rawTx types.TxData) (*types.Transaction, error) {
		return types.SignNewTx(privKey, types.LatestSignerForChainID(chainID), rawTx)
	}

	t := &TransactionManager{
		batchInboxAddress: batchInboxAddress,
		senderAddress:     crypto.PubkeyToAddress(privKey.PublicKey),
		chainID:           chainID,
		txMgr:             txmgr.NewSimpleTxManager("batcher", txMgrConfg, l1Client),
		l1Client:          l1Client,
		signerFn:          signerFn,
		log:               log,
	}
	return t
}

// SendTransaction creates & submits a transaction to the batch inbox address with the given `data`.
// It currently uses the underlying `txmgr` to handle transaction sending & price management.
// This is a blocking method. It should not be called concurrently.
// TODO: where to put concurrent transaction handling logic.
func (t *TransactionManager) SendTransaction(ctx context.Context, data []byte) (*types.Receipt, error) {
	tx, err := t.CraftTx(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx: %w", err)
	}
	// Construct a closure that will update the txn with the current gas prices.
	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		return t.UpdateGasPrice(ctx, tx)
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Second) // TODO: Select a timeout that makes sense here.
	defer cancel()
	if receipt, err := t.txMgr.Send(ctx, updateGasPrice, t.l1Client.SendTransaction); err != nil {
		t.log.Warn("unable to publish tx", "err", err)
		return nil, err
	} else {
		t.log.Info("tx successfully published", "tx_hash", receipt.TxHash)
		return receipt, nil
	}
}

// calcGasTipAndFeeCap queries L1 to determine what a suitable miner tip & basefee limit would be for timely inclusion
func (t *TransactionManager) calcGasTipAndFeeCap(ctx context.Context) (gasTipCap *big.Int, gasFeeCap *big.Int, err error) {
	childCtx, cancel := context.WithTimeout(ctx, networkTimeout)
	gasTipCap, err = t.l1Client.SuggestGasTipCap(childCtx)
	cancel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get suggested gas tip cap: %w", err)
	}

	childCtx, cancel = context.WithTimeout(ctx, networkTimeout)
	head, err := t.l1Client.HeaderByNumber(childCtx, nil)
	cancel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L1 head block for fee cap: %w", err)
	}
	gasFeeCap = txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	return gasTipCap, gasFeeCap, nil
}

// CraftTx creates the signed transaction to the batchInboxAddress.
// It queries L1 for the current fee market conditions as well as for the nonce.
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (t *TransactionManager) CraftTx(ctx context.Context, data []byte) (*types.Transaction, error) {
	gasTipCap, gasFeeCap, err := t.calcGasTipAndFeeCap(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, networkTimeout)
	nonce, err := t.l1Client.NonceAt(ctx, t.senderAddress, nil)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	rawTx := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     nonce,
		To:        &t.batchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      data,
	}
	t.log.Info("creating tx", "to", rawTx.To, "from", t.senderAddress)

	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate intrinsic gas: %w", err)
	}
	rawTx.Gas = gas

	return t.signerFn(rawTx)
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (t *TransactionManager) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	gasTipCap, gasFeeCap, err := t.calcGasTipAndFeeCap(ctx)
	if err != nil {
		return nil, err
	}

	rawTx := &types.DynamicFeeTx{
		ChainID:   t.chainID,
		Nonce:     tx.Nonce(),
		To:        tx.To(),
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       tx.Gas(),
		Data:      tx.Data(),
	}
	// Only log the new tip/fee cap because the updateGasPrice closure reuses the same initial transaction
	t.log.Trace("updating gas price", "tip_cap", gasTipCap, "fee_cap", gasFeeCap)

	return t.signerFn(rawTx)
}
