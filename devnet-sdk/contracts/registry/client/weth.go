package client

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type WETHBinding struct {
	contractAddress types.Address
	client          *ethclient.Client
	binding         *bindings.SuperchainWETH
}

var _ interfaces.WETH = (*WETHBinding)(nil)

func (b *WETHBinding) BalanceOf(addr types.Address) types.ReadInvocation[types.Balance] {
	return &WETHBalanceOfImpl{
		contract: b,
		addr:     addr,
	}
}

type WETHBalanceOfImpl struct {
	contract *WETHBinding
	addr     types.Address
}

func (i *WETHBalanceOfImpl) Call(ctx context.Context) (types.Balance, error) {
	balance, err := i.contract.binding.BalanceOf(nil, i.addr)
	if err != nil {
		return types.Balance{}, err
	}
	return types.NewBalance(balance), nil
}
