package filter

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// EthClient defines the block and receipt fetching dependency used by the
// logsDB ingester. Tests can inject an in-memory implementation.
type EthClient interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethTypes.Receipts, error)
	Close()
}

type rpcEthClient struct {
	rpc client.RPC
	eth *sources.EthClient
}

// NewRPCEthClient creates the production RPC-backed EthClient.
func NewRPCEthClient(ctx context.Context, logger log.Logger, rpcURL string) (EthClient, error) {
	rpcClient, err := client.NewRPC(ctx, logger, rpcURL)
	if err != nil {
		return nil, err
	}

	ethClient, err := sources.NewEthClient(
		rpcClient,
		logger,
		nil,
		&sources.EthClientConfig{
			ReceiptsCacheSize:     1000,
			TransactionsCacheSize: 1000,
			HeadersCacheSize:      1000,
			PayloadsCacheSize:     100,
			MaxRequestsPerBatch:   20,
			MaxConcurrentRequests: 10,
			TrustRPC:              false,
			MustBePostMerge:       true,
			RPCProviderKind:       sources.RPCKindStandard,
		},
	)
	if err != nil {
		rpcClient.Close()
		return nil, err
	}

	return &rpcEthClient{
		rpc: rpcClient,
		eth: ethClient,
	}, nil
}

func (c *rpcEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	return c.eth.InfoByLabel(ctx, label)
}

func (c *rpcEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	return c.eth.InfoByNumber(ctx, number)
}

func (c *rpcEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, gethTypes.Receipts, error) {
	return c.eth.FetchReceipts(ctx, blockHash)
}

func (c *rpcEthClient) Close() {
	c.eth.Close()
	c.rpc.Close()
}

var _ EthClient = (*rpcEthClient)(nil)
