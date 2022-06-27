package db

import "github.com/ethereum/go-ethereum/common"

// ETHL1Token is a placeholder token for differentiating ETH transactions from
// ERC20 transactions on L1.
var ETHL1Token = &Token{
	Address:  "0x0000000000000000000000000000000000000000",
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}

// ETHL2Address is a placeholder address for differentiating ETH transactions
// from ERC20 transactions on L2.
var ETHL2Address = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")

// ETHL2Token is a placeholder token for differentiating ETH transactions from
// ERC20 transactions on L2.
var ETHL2Token = &Token{
	Address:  "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
	Name:     "Ethereum",
	Symbol:   "ETH",
	Decimals: 18,
}
