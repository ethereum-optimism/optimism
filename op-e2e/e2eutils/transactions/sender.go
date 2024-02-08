package transactions

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	crypto2 "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const basefeeWiggleMultiplier = 2

func SendTx(ctx context.Context, privKey *ecdsa.PrivateKey, candidate txmgr.TxCandidate, client *ethclient.Client) (*types.Transaction, *types.Receipt, error) {
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain ID: %w", err)
	}
	signer := crypto.PrivateKeySignerFn(privKey, chainID)
	from := crypto2.PubkeyToAddress(privKey.PublicKey)
	// Query for basefee if gasPrice not specified
	head, errHead := client.HeaderByNumber(ctx, nil)
	if errHead != nil {
		return nil, nil, errHead
	} else if head.BaseFee == nil {
		return nil, nil, errors.New("pre-london chains not supported")
	}
	rawTx, err := createDynamicTx(ctx, from, client, candidate, head)
	if err != nil {
		return nil, nil, err
	}
	// Sign the transaction and schedule it for execution
	signedTx, err := signer(from, rawTx)
	if err != nil {
		return nil, nil, err
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return nil, nil, err
	}
	rcpt, err := wait.ForReceiptOK(ctx, client, signedTx.Hash())
	return signedTx, rcpt, err
}

func createDynamicTx(ctx context.Context, from common.Address, client *ethclient.Client, candidate txmgr.TxCandidate, head *types.Header) (*types.Transaction, error) {
	// Normalize value
	value := candidate.Value
	if value == nil {
		value = new(big.Int)
	}
	// Estimate TipCap
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}

	// Estimate FeeCap
	gasFeeCap := new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(head.BaseFee, big.NewInt(basefeeWiggleMultiplier)),
	)
	if gasFeeCap.Cmp(gasTipCap) < 0 {
		return nil, fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", gasFeeCap, gasTipCap)
	}
	// Estimate GasLimit
	gasLimit := candidate.GasLimit
	if candidate.GasLimit == 0 {
		var err error
		gasLimit, err = estimateGasLimit(ctx, from, client, candidate, gasTipCap, gasFeeCap, value)
		if err != nil {
			return nil, err
		}
	}
	// create the transaction
	nonce, err := getNonce(ctx, client, from)
	if err != nil {
		return nil, err
	}
	baseTx := &types.DynamicFeeTx{
		To:        candidate.To,
		Nonce:     nonce,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Gas:       gasLimit,
		Value:     value,
		Data:      candidate.TxData,
	}
	return types.NewTx(baseTx), nil
}

func estimateGasLimit(ctx context.Context, from common.Address, client *ethclient.Client, candidate txmgr.TxCandidate, gasTipCap, gasFeeCap, value *big.Int) (uint64, error) {
	msg := ethereum.CallMsg{
		From:      from,
		To:        candidate.To,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Value:     value,
		Data:      candidate.TxData,
	}
	return client.EstimateGas(ctx, msg)
}

func getNonce(ctx context.Context, client *ethclient.Client, from common.Address) (uint64, error) {
	return client.PendingNonceAt(ctx, from)
}
