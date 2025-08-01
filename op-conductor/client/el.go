package client

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ElP2PClient interface {
	PeerCount(ctx context.Context) (int, error)
}

type elP2PClientNet struct {
	client *ethclient.Client
}

var _ ElP2PClient = (*elP2PClientNet)(nil)

func NewElP2PClientNet(client *ethclient.Client) ElP2PClient {
	return &elP2PClientNet{
		client: client,
	}
}

func (c *elP2PClientNet) PeerCount(ctx context.Context) (int, error) {
	var peerCount hexutil.Uint64
	if err := c.client.Client().Call(&peerCount, "net_peerCount"); err != nil {
		return 0, err
	}

	return int(peerCount), nil
}

type elP2PClientAdmin struct {
	client *ethclient.Client
}

var _ ElP2PClient = (*elP2PClientAdmin)(nil)

func NewElP2PClientAdmin(client *ethclient.Client) ElP2PClient {
	return &elP2PClientAdmin{
		client: client,
	}
}

func (c *elP2PClientAdmin) PeerCount(ctx context.Context) (int, error) {
	var peerCount []*p2p.PeerInfo
	if err := c.client.Client().Call(&peerCount, "admin_peers"); err != nil {
		return 0, err
	}

	return len(peerCount), nil
}
