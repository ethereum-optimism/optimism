package db

import (
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
)

var ETHL1Address common.Address

// ETHL1Token is a placeholder token for differentiating ETH transactions from
// ERC20 transactions on L1.
var ETHL1Token = &Token{
	Address:  ETHL1Address.String(),
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}

// ETHL2Token is a placeholder token for differentiating ETH transactions from
// ERC20 transactions on L2.
var ETHL2Token = &Token{
	Address:  predeploys.LegacyERC20ETH,
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}
