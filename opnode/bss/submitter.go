package bss

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BatchSubmitter struct {
	Client    *ethclient.Client
	ToAddress common.Address
	ChainID   *big.Int
	PrivKey   *ecdsa.PrivateKey
}

// Submit creates & submits batches to L1. Blocks until the transaction is included.
// Return the tx hash as well as a possible error.
func (b *BatchSubmitter) Submit(config *rollup.Config, batches []*derive.BatchData) (common.Hash, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var buf bytes.Buffer
	if err := derive.EncodeBatches(config, batches, &buf); err != nil {
		return common.Hash{}, err
	}

	tip, err := b.Client.SuggestGasTipCap(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	fee, err := b.Client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	// Note: If the BSS acts up, look into the pending nonce.
	addr := crypto.PubkeyToAddress(b.PrivKey.PublicKey)
	nonce, err := b.Client.PendingNonceAt(ctx, addr)
	if err != nil {
		return common.Hash{}, err
	}

	rawTx := &types.DynamicFeeTx{
		ChainID:   b.ChainID,
		Nonce:     nonce,
		To:        &b.ToAddress,
		GasTipCap: tip,
		GasFeeCap: fee,
		Data:      buf.Bytes(),
	}

	// No contract execution so we just pay intrinsic gas.
	// If we add contract execution, making it gas usage deterministic is very helpful.
	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true)
	if err != nil {
		return common.Hash{}, err
	}
	rawTx.Gas = gas

	tx, err := types.SignNewTx(b.PrivKey, types.LatestSignerForChainID(b.ChainID), rawTx)
	if err != nil {
		return common.Hash{}, err
	}

	err = b.Client.SendTransaction(ctx, tx)
	if err != nil {
		return common.Hash{}, err
	}

	timeout := time.After(30 * time.Second)

	for {
		receipt, err := b.Client.TransactionReceipt(context.Background(), tx.Hash())
		if receipt != nil {
			return tx.Hash(), nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return common.Hash{}, err
		}
		<-time.After(150 * time.Millisecond)

		select {
		case <-timeout:
			return common.Hash{}, errors.New("timeout")
		default:
		}
	}

}
