package ethclient

import (
	"context"
	"math/big"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/log/v3"
)

type Client struct {
	Client *rpc.Client
}

func NewEthClient(url string) (*Client, error) {
	c, err := rpc.DialContext(context.Background(), url, log.New())
	if err != nil {
		return nil, err
	}
	return &Client{Client: c}, nil
}

func (c *Client) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutility.Bytes
	if err := c.Client.CallContext(ctx, &result, "eth_getCode", contract); err != nil {
		return nil, err
	}
	return result, nil
}
