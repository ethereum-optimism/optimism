package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

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

var _ stack.ELNode = (*rpcELNode)(nil)

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

func (r *rpcELNode) EthClient() apis.EthClient {
	return r.ethClient
}
