package state

import "github.com/ethereum/go-ethereum/common"

// TODO: add oracle calls here
// wrapper for the oracle

type Database struct {
}

// ContractCode retrieves a particular contract's code.
func (db *Database) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	return nil, nil
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *Database) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	return 0, nil
}
