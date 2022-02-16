package l1

import (
	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1erc20"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func QueryERC20(address common.Address, contract *l1erc20.L1ERC20) (*db.Token, error) {
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
