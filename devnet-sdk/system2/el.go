package system2

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type EthClient interface {
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	// more methods may be added
}

type ELNode interface {
	Common
	ChainID() eth.ChainID
	EthClient() EthClient
}

type ELNodeConfig struct {
	CommonConfig
	Client  client.RPC
	ChainID eth.ChainID
}

type rpcELNode struct {
	commonImpl

	client    client.RPC
	ethClient *sources.EthClient
	chainID   eth.ChainID
}

var _ ELNode = (*rpcELNode)(nil)

// newRpcELNode creates a generic ELNode, safe to embed in other structs
func newRpcELNode(cfg ELNodeConfig) rpcELNode {
	ethCl, err := sources.NewEthClient(cfg.Client, cfg.Log, nil, sources.DefaultEthClientConfig(10))
	require.NoError(cfg.T, err)

	return rpcELNode{
		commonImpl: newCommon(cfg.CommonConfig),
		client:     cfg.Client,
		ethClient:  ethCl,
		chainID:    cfg.ChainID,
	}
}

func (r *rpcELNode) ChainID() eth.ChainID {
	return r.chainID
}

func (r *rpcELNode) EthClient() EthClient {
	return r.ethClient
}

func (r *rpcELNode) Close() {
	r.client.Close()
}
