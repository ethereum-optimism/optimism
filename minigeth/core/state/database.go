package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/oracle"
)

// TODO: add oracle calls here
// wrapper for the oracle

type Database struct {
	BlockNumber *big.Int
}

// ContractCode retrieves a particular contract's code.
func (db *Database) ContractCode(addr common.Address, codeHash common.Hash) ([]byte, error) {
	code := oracle.GetProvedCodeBytes(db.BlockNumber, addr, codeHash)
	return code, nil
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *Database) ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	code := oracle.GetProvedCodeBytes(db.BlockNumber, addr, codeHash)
	return len(code), nil
}
