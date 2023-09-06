package clients

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-ufm/pkg/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type InstrumentedEthClient struct {
	c            *ethclient.Client
	providerName string
}

func Dial(providerName string, url string) (*InstrumentedEthClient, error) {
	start := time.Now()
	c, err := ethclient.Dial(url)
	if err != nil {
		metrics.RecordErrorDetails(providerName, "ethclient.Dial", err)
		return nil, err
	}
	metrics.RecordRPCLatency(providerName, "ethclient", "Dial", time.Since(start))
	return &InstrumentedEthClient{c: c, providerName: providerName}, nil
}

func (i *InstrumentedEthClient) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	start := time.Now()
	tx, isPending, err := i.c.TransactionByHash(ctx, hash)
	if err != nil {
		if !i.ignorableErrors(err) {
			metrics.RecordErrorDetails(i.providerName, "ethclient.TransactionByHash", err)
		}
		return nil, false, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "TransactionByHash", time.Since(start))
	return tx, isPending, err
}

func (i *InstrumentedEthClient) PendingNonceAt(ctx context.Context, address string) (uint64, error) {
	start := time.Now()
	nonce, err := i.c.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		metrics.RecordErrorDetails(i.providerName, "ethclient.PendingNonceAt", err)
		return 0, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "PendingNonceAt", time.Since(start))
	return nonce, err
}

func (i *InstrumentedEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	start := time.Now()
	receipt, err := i.c.TransactionReceipt(ctx, txHash)
	if err != nil {
		if !i.ignorableErrors(err) {
			metrics.RecordErrorDetails(i.providerName, "ethclient.TransactionReceipt", err)
		}
		return nil, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "TransactionReceipt", time.Since(start))
	return receipt, err
}

func (i *InstrumentedEthClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	start := time.Now()
	err := i.c.SendTransaction(ctx, tx)
	if err != nil {
		if !i.ignorableErrors(err) {
			metrics.RecordErrorDetails(i.providerName, "ethclient.SendTransaction", err)
		}
		return err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "SendTransaction", time.Since(start))
	return err
}

func (i *InstrumentedEthClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	start := time.Now()
	gas, err := i.c.EstimateGas(ctx, msg)
	if err != nil {
		metrics.RecordErrorDetails(i.providerName, "ethclient.EstimateGas", err)
		return 0, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "EstimateGas", time.Since(start))
	return gas, err
}

func (i *InstrumentedEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	start := time.Now()
	gasTipCap, err := i.c.SuggestGasTipCap(ctx)
	if err != nil {
		metrics.RecordErrorDetails(i.providerName, "ethclient.SuggestGasTipCap", err)
		return nil, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "SuggestGasTipCap", time.Since(start))
	return gasTipCap, err
}

func (i *InstrumentedEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	start := time.Now()
	header, err := i.c.HeaderByNumber(ctx, number)
	if err != nil {
		metrics.RecordErrorDetails(i.providerName, "ethclient.HeaderByNumber", err)
		return nil, err
	}
	metrics.RecordRPCLatency(i.providerName, "ethclient", "HeaderByNumber", time.Since(start))
	return header, err
}

func (i *InstrumentedEthClient) ignorableErrors(err error) bool {
	msg := err.Error()
	// we dont use errors.Is because eth client actually uses errors.New,
	// therefore creating an incomparable instance :(
	return msg == ethereum.NotFound.Error() ||
		msg == txpool.ErrAlreadyKnown.Error() ||
		msg == core.ErrNonceTooLow.Error()
}
