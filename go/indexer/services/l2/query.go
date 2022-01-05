package l2

import (
	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2erc20"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
)

func QueryERC20(address l2common.Address, client *l2ethclient.Client) (*db.Token, error) {
	contract, err := l2erc20.NewL2ERC20(address, client)
	if err != nil {
		return nil, err
	}

	name, err := contract.Name(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	symbol, err := contract.Symbol(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	decimals, err := contract.Decimals(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	return &db.Token{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}
