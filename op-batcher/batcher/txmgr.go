package batcher

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
)

const networkTimeout = 2 * time.Second // How long a single network request can take. TODO: put in a config somewhere

// ExternalTxManager is a minimal interface for the [TransactionManager]
// that is a subset of [txmgr.SimpleTxManager] functionality
//
//go:generate mockery --name ExternalTxManager --output ./mocks
type ExternalTxManager interface {
	Send(ctx context.Context, tx *types.Transaction) (*types.Receipt, error)
}

// NewExternalTxManager creates a new tx manager with the given config.
func NewExternalTxManager(log log.Logger, txMgrConfg txmgr.Config, l1Client txmgr.ETHBackend) ExternalTxManager {
	return txmgr.NewSimpleTxManager("batcher", log, txMgrConfg, l1Client)
}

// TxManagerProvider is a minimal interface for the [TransactionManager]
// to fetch data from a fully-featured [ethclient.Client].
//
//go:generate mockery --name TxManagerProvider --output ./mocks
type TxManagerProvider interface {
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

// TransactionManager wraps the simple txmgr package to make it easy to send & wait for transactions
type TransactionManager struct {
	// Config
	batchInboxAddress common.Address
	senderAddress     common.Address
	chainID           *big.Int
	// Outside world
	txMgr    ExternalTxManager
	l1Client TxManagerProvider
	signerFn opcrypto.SignerFn
	log      log.Logger
}

func NewTransactionManager(log log.Logger, txMgrConfg txmgr.Config, batchInboxAddress common.Address, chainID *big.Int, senderAddress common.Address, l1Client TxManagerProvider, externalTxMgr ExternalTxManager) *TransactionManager {
	t := &TransactionManager{
		batchInboxAddress: batchInboxAddress,
		senderAddress:     senderAddress,
		chainID:           chainID,
		txMgr:             externalTxMgr,
		l1Client:          l1Client,
		signerFn:          txMgrConfg.Signer,
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

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute) // TODO: Select a timeout that makes sense here.
	defer cancel()
	if receipt, err := t.txMgr.Send(ctx, tx); err != nil {
		t.log.Warn("unable to publish tx", "err", err, "data_size", len(data))
		return nil, err
	} else {
		t.log.Info("tx successfully published", "tx_hash", receipt.TxHash, "data_size", len(data))
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

	if gasTipCap == nil {
		t.log.Warn("unexpected unset gasTipCap, using default 2 gwei")
		gasTipCap = new(big.Int).SetUint64(params.GWei * 2)
	}

	childCtx, cancel = context.WithTimeout(ctx, networkTimeout)
	head, err := t.l1Client.HeaderByNumber(childCtx, nil)
	cancel()
	if err != nil || head == nil {
		return nil, nil, fmt.Errorf("failed to get L1 head block for fee cap: %w", err)
	}
	if head.BaseFee == nil {
		return nil, nil, fmt.Errorf("failed to get L1 basefee in block %d for fee cap", head.Number)
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

	childCtx, cancel := context.WithTimeout(ctx, networkTimeout)
	nonce, err := t.l1Client.NonceAt(childCtx, t.senderAddress, nil)
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

	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate intrinsic gas: %w", err)
	}
	rawTx.Gas = gas

	ctx, cancel = context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	tx := types.NewTx(rawTx)

	return t.signerFn(ctx, t.senderAddress, tx)
}
