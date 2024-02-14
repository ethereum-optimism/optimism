package ethclient

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/common/hexutil"
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
	if err := c.Client.CallContext(ctx, &result, "eth_getCode", contract, toBlockNumArg(blockNumber)); err != nil {
		return nil, fmt.Errorf("error calling eth_getCode: %w", err)
	}
	return result, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	if number.IsInt64() {
		return rpc.BlockNumber(number.Int64()).String()
	}
	// It's negative and large, which is invalid.
	return fmt.Sprintf("<invalid %d>", number)
}
