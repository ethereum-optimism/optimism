package transactions

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type SendTxOpt func(cfg *sendTxCfg)

type sendTxCfg struct {
	receiptStatus       uint64
	ignoreReceiptStatus bool
}

func makeSendTxCfg(opts ...SendTxOpt) *sendTxCfg {
	cfg := &sendTxCfg{
		receiptStatus: types.ReceiptStatusSuccessful,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func WithReceiptFail() SendTxOpt {
	return func(cfg *sendTxCfg) {
		cfg.receiptStatus = types.ReceiptStatusFailed
	}
}

func WithReceiptStatusIgnore() SendTxOpt {
	return func(cfg *sendTxCfg) {
		cfg.ignoreReceiptStatus = true
	}
}

func RequireSendTx(t *testing.T, ctx context.Context, client *ethclient.Client, candidate txmgr.TxCandidate, privKey *ecdsa.PrivateKey, opts ...SendTxOpt) (*types.Transaction, *types.Receipt) {
	tx, rcpt, err := SendTx(ctx, client, candidate, privKey, opts...)
	require.NoError(t, err, "Failed to send transaction")
	return tx, rcpt
}

func SendTx(ctx context.Context, client *ethclient.Client, candidate txmgr.TxCandidate, privKey *ecdsa.PrivateKey, opts ...SendTxOpt) (*types.Transaction, *types.Receipt, error) {
	cfg := makeSendTxCfg(opts...)
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain ID: %w", err)
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get next nonce: %w", err)
	}

	latestBlock, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block: %w", err)
	}
	gasFeeCap := new(big.Int).Mul(latestBlock.BaseFee, big.NewInt(3))
	gasTipCap := big.NewInt(1 * params.GWei)
	if gasFeeCap.Cmp(gasTipCap) < 0 {
		// gasTipCap can't be higher than gasFeeCap
		// Since there's a minimum gasTipCap to be accepted, increase the gasFeeCap. Extra will be refunded anyway.
		gasFeeCap = gasTipCap
	}
	msg := ethereum.CallMsg{
		From:      from,
		To:        candidate.To,
		Value:     candidate.Value,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      candidate.TxData,
	}
	gas, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to estimate gas: %w", errutil.TryAddRevertReason(err))
	}

	tx := types.MustSignNewTx(privKey, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        candidate.To,
		Value:     candidate.Value,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      candidate.TxData,
		Gas:       gas,
	})
	err = client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send transaction (tx: %s): %w", tx.Hash(), errutil.TryAddRevertReason(err))
	}
	receipt, err := wait.ForReceiptMaybe(ctx, client, tx.Hash(), cfg.receiptStatus, cfg.ignoreReceiptStatus)
	if err != nil {
		return tx, receipt, fmt.Errorf("failed to find OK receipt (tx: %s): %w", tx.Hash(), err)
	}
	return tx, receipt, nil
}
