package p2p

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	c sources.P2PClient
}

// Legacy for supporting backwards compatibility
func NewClient(c *rpc.Client) *Client {
	return &Client{c: *sources.NewP2PClient(client.NewBaseRPCClient(c))}
}
