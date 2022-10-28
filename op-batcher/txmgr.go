package op_batcher

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

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
