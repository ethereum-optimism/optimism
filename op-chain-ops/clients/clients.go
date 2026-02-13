package clients

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

// Clients represents a set of initialized RPC clients
type Clients struct {
	L1Client     *ethclient.Client
	L2Client     *ethclient.Client
	L1RpcClient  *rpc.Client
	L2RpcClient  *rpc.Client
	L1GethClient *gethclient.Client
	L2GethClient *gethclient.Client
}

// NewClientsFromContext will create new RPC clients from a CLI context
func NewClientsFromContext(ctx *cli.Context) (*Clients, error) {
	return NewClients(ctx.String("l1-rpc-url"), ctx.String("l2-rpc-url"))
}

// NewClients will create new RPC clients from given URLs.
func NewClients(l1RpcURL, l2RpcURL string) (*Clients, error) {
	clients := Clients{}

	if l1RpcURL != "" {
		l1Client, l1RpcClient, l1GethClient, err := newClients(l1RpcURL)
		if err != nil {
			return nil, err
		}
		clients.L1Client = l1Client
		clients.L1RpcClient = l1RpcClient
		clients.L1GethClient = l1GethClient
	}

	if l2RpcURL != "" {
		l2Client, l2RpcClient, l2GethClient, err := newClients(l2RpcURL)
		if err != nil {
			return nil, err
		}
		clients.L2Client = l2Client
		clients.L2RpcClient = l2RpcClient
		clients.L2GethClient = l2GethClient
	}

	return &clients, nil
}

// newClients will create new clients from a given URL
func newClients(url string) (*ethclient.Client, *rpc.Client, *gethclient.Client, error) {
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot dial ethclient: %w", err)
	}
	rpcClient, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot dial rpc client: %w", err)
	}
	return ethClient, rpcClient, gethclient.New(rpcClient), nil
}
