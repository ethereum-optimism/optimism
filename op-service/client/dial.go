package client

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// DefaultDialTimeout is a default timeout for dialing a client.
const DefaultDialTimeout = 1 * time.Minute
const defaultRetryCount = 30
const defaultRetryTime = 2 * time.Second

// DialEthClientWithTimeout attempts to dial the L1 provider using the provided
// URL. If the dial doesn't complete within defaultDialTimeout seconds, this
// method will return an error.
func DialEthClientWithTimeout(timeout time.Duration, log log.Logger, url string) (*ethclient.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c, err := dialRPCClientWithBackoff(ctx, log, url)
	if err != nil {
		return nil, err
	}

	return ethclient.NewClient(c), nil
}

// DialRollupClientWithTimeout attempts to dial the RPC provider using the provided URL.
// If the dial doesn't complete within timeout seconds, this method will return an error.
func DialRollupClientWithTimeout(timeout time.Duration, log log.Logger, url string) (*sources.RollupClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rpcCl, err := dialRPCClientWithBackoff(ctx, log, url)
	if err != nil {
		return nil, err
	}

	return sources.NewRollupClient(client.NewBaseRPCClient(rpcCl)), nil
}

// Dials a JSON-RPC endpoint repeatedly, with a backoff, until a client connection is established. Auth is optional.
func dialRPCClientWithBackoff(ctx context.Context, log log.Logger, addr string) (*rpc.Client, error) {
	bOff := retry.Fixed(defaultRetryTime)
	return retry.Do(ctx, defaultRetryCount, bOff, func() (*rpc.Client, error) {
		if !client.IsURLAvailable(addr) {
			log.Warn("failed to dial address, but may connect later", "addr", addr)
			return nil, fmt.Errorf("address unavailable (%s)", addr)
		}
		client, err := rpc.DialOptions(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address (%s): %w", addr, err)
		}
		return client, nil
	})
}

const BatcherFallbackThreshold int64 = 10
const ProposerFallbackThreshold int64 = 3
const TxmgrFallbackThreshold int64 = 3

// DialEthClientWithTimeoutAndFallback will try to dial within the timeout period and create an EthClient.
// If the URL is a multi URL, then a fallbackClient will be created to add the fallback capability to the client
func DialEthClientWithTimeoutAndFallback(ctx context.Context, url []string, timeout time.Duration, l log.Logger, fallbackThreshold int64, m FallbackClientMetricer) (EthClient, error) {
	ctxt, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if len(url) > 1 {
		firstEthClient, err := dialRPCClientWithBackoff(ctxt, l, url[0])
		if err != nil {
			return nil, err
		}
		fallbackClient := NewFallbackClient(ethclient.NewClient(firstEthClient), url, l, fallbackThreshold, m, func(url string) (EthClient, error) {
			ctxtIn, cancelIn := context.WithTimeout(ctx, timeout)
			defer cancelIn()
			ethClientNew, err := ethclient.DialContext(ctxtIn, url)
			if err != nil {
				return nil, err
			}
			return ethClientNew, nil
		})
		return fallbackClient, nil
	}

	return DialEthClientWithTimeout(timeout, l, url[0])
}

type EthClient interface {
	ChainID(ctx context.Context) (*big.Int, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	BlockNumber(ctx context.Context) (uint64, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	Close()
}
