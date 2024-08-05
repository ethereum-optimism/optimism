package txmgr

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type TestTxManager struct {
	*SimpleTxManager
	ss *SendState
	tx *types.Transaction
}

// JamTxPool sends a transaction intended to get stuck in the txpool, and should be used ONLY for testing.
// It is non-blocking. See WaitOnJammingTx if you wish to wait on the transaction to clear.
func (m *TestTxManager) JamTxPool(ctx context.Context, candidate TxCandidate) error {
	var err error
	m.tx, err = m.makeStuckTx(ctx, candidate)
	if err != nil {
		return err
	}
	m.ss = NewSendState(m.cfg.SafeAbortNonceTooLowCount, m.cfg.TxNotInMempoolTimeout)
	if err := m.backend.SendTransaction(ctx, m.tx); err != nil {
		return err
	}
	return nil
}

// WaitOnJammingTx can be called after JamTxPool in order to wait on the jam transaction clearing.
func (m *TestTxManager) WaitOnJammingTx(ctx context.Context) error {
	if m.ss == nil {
		return errors.New("WaitOnJammingTx called without first calling JamTxPool")
	}
	_, err := m.waitMined(ctx, m.tx, m.ss)
	return err
}

func (m *TestTxManager) makeStuckTx(ctx context.Context, candidate TxCandidate) (*types.Transaction, error) {
	gasTipCap, _, blobBaseFee, err := m.SuggestGasPriceCaps(ctx)
	if err != nil {
		return nil, err
	}

	// override with minimal fees to make sure tx gets stuck in the pool
	gasFeeCap := big.NewInt(2)
	gasTipCap.SetUint64(1)

	var sidecar *types.BlobTxSidecar
	var blobHashes []common.Hash
	if len(candidate.Blobs) > 0 {
		if sidecar, blobHashes, err = MakeSidecar(candidate.Blobs); err != nil {
			return nil, err
		}
	}

	nonce, err := m.backend.NonceAt(ctx, m.cfg.From, nil)
	if err != nil {
		return nil, err
	}

	var txMessage types.TxData
	if sidecar != nil {
		blobFeeCap := m.calcBlobFeeCap(blobBaseFee)
		message := &types.BlobTx{
			To:         *candidate.To,
			Data:       candidate.TxData,
			Gas:        candidate.GasLimit,
			BlobHashes: blobHashes,
			Sidecar:    sidecar,
			Nonce:      nonce,
		}
		if err := finishBlobTx(message, m.chainID, gasTipCap, gasFeeCap, blobFeeCap, candidate.Value); err != nil {
			return nil, err
		}
		txMessage = message
	} else {
		txMessage = &types.DynamicFeeTx{
			ChainID:   m.chainID,
			To:        candidate.To,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Value:     candidate.Value,
			Data:      candidate.TxData,
			Gas:       candidate.GasLimit,
			Nonce:     nonce,
		}
	}

	return m.cfg.Signer(ctx, m.cfg.From, types.NewTx(txMessage))
}
