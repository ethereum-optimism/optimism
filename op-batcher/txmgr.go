package op_batcher

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func (l *BatchSubmitter) submitTransaction(data []byte) error {
	// Query for the submitter's current nonce.
	ctx, cancel := context.WithTimeout(l.ctx, time.Second*10)
	nonce, err := l.cfg.L1Client.NonceAt(ctx, l.addr, nil)
	cancel()
	if err != nil {
		l.log.Error("unable to get current nonce", "err", err)
		return err
	}

	// Create the transaction
	ctx, cancel = context.WithTimeout(l.ctx, time.Second*10)
	tx, err := l.CraftTx(ctx, data, nonce)
	cancel()
	if err != nil {
		l.log.Error("unable to craft tx", "err", err)
		return err
	}

	// Construct the a closure that will update the txn with the current gas prices.
	updateGasPrice := func(ctx context.Context) (*types.Transaction, error) {
		l.log.Debug("updating batch tx gas price")
		return l.UpdateGasPrice(ctx, tx)
	}

	// Wait until one of our submitted transactions confirms. If no
	// receipt is received it's likely our gas price was too low.
	// TODO: does the tx manager nicely replace the tx?
	//  (submit a new one, that's within the channel timeout, but higher fee than previously submitted tx? Or use a cheap cancel tx?)
	ctx, cancel = context.WithTimeout(l.ctx, time.Second*time.Duration(l.cfg.ChannelTimeout))
	receipt, err := l.txMgr.Send(ctx, updateGasPrice, l.cfg.L1Client.SendTransaction)
	cancel()
	if err != nil {
		l.log.Warn("unable to publish tx", "err", err)
		return err
	}

	// The transaction was successfully submitted.
	l.log.Info("tx successfully published", "tx_hash", receipt.TxHash)
	return nil
}

// NOTE: This method SHOULD NOT publish the resulting transaction.
func (l *BatchSubmitter) CraftTx(ctx context.Context, data []byte, nonce uint64) (*types.Transaction, error) {
	gasTipCap, err := l.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	head, err := l.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   l.cfg.ChainID,
		Nonce:     nonce,
		To:        &l.cfg.BatchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      data,
	}
	l.log.Debug("creating tx", "to", rawTx.To, "from", crypto.PubkeyToAddress(l.cfg.PrivKey.PublicKey))

	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true)
	if err != nil {
		return nil, err
	}
	rawTx.Gas = gas

	return types.SignNewTx(l.cfg.PrivKey, types.LatestSignerForChainID(l.cfg.ChainID), rawTx)
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: Thie method SHOULD NOT publish the resulting transaction.
func (l *BatchSubmitter) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	gasTipCap, err := l.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	head, err := l.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   l.cfg.ChainID,
		Nonce:     tx.Nonce(),
		To:        tx.To(),
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       tx.Gas(),
		Data:      tx.Data(),
	}

	return types.SignNewTx(l.cfg.PrivKey, types.LatestSignerForChainID(l.cfg.ChainID), rawTx)
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (l *BatchSubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return l.cfg.L1Client.SendTransaction(ctx, tx)
}
