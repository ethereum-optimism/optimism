package utils

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Contract struct {
	address   common.Address
	parsedABI abi.ABI
}

func NewContract(address common.Address, parsedABI abi.ABI) *Contract {
	return &Contract{
		address:   address,
		parsedABI: parsedABI,
	}
}

func (c *Contract) Address() common.Address {
	return c.address
}

func (c *Contract) ABI() abi.ABI {
	return c.parsedABI
}
