package sources

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type FaucetClient struct {
	client client.RPC
}

var _ apis.Faucet = (*FaucetClient)(nil)

func NewFaucetClient(client client.RPC) *FaucetClient {
	return &FaucetClient{
		client: client,
	}
}

func (cl *FaucetClient) ChainID(ctx context.Context) (eth.ChainID, error) {
	var result eth.ChainID
	err := cl.client.CallContext(ctx, &result, "faucet_chainID")
	return result, err
}

func (cl *FaucetClient) RequestETH(ctx context.Context, addr common.Address, amount eth.ETH) error {
	return cl.client.CallContext(ctx, nil, "faucet_requestETH", addr, amount)
}
