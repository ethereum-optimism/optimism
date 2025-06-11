package geth

import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
)

type GethInstance struct {
	Backend *eth.Ethereum
	Node    *node.Node
}

var _ services.EthInstance = (*GethInstance)(nil)

func (gi *GethInstance) UserRPC() endpoint.RPC {
	fallback := endpoint.WsOrHttpRPC{
		WsURL:   gi.Node.WSEndpoint(),
		HttpURL: gi.Node.HTTPEndpoint(),
	}
	srv, err := gi.Node.RPCHandler()
	if err != nil {
		return fallback
	}
	return &endpoint.ServerRPC{
		Fallback: fallback,
		Server:   srv,
	}
}

func (gi *GethInstance) AuthRPC() endpoint.RPC {
	// TODO: can we rely on the in-process RPC server to support the auth namespaces?
	return endpoint.WsOrHttpRPC{
		WsURL:   gi.Node.WSAuthEndpoint(),
		HttpURL: gi.Node.HTTPAuthEndpoint(),
	}
}

func (gi *GethInstance) Close() error {
	return gi.Node.Close()
}
