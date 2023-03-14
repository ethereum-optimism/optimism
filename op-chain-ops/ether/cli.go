package ether

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/state"
)

// getOVMETHTotalSupply returns OVM ETH's total supply by reading
// the appropriate storage slot.
func getOVMETHTotalSupply(db *state.StateDB) *big.Int {
	key := getOVMETHTotalSupplySlot()
	return db.GetState(OVMETHAddress, key).Big()
}

func getOVMETHTotalSupplySlot() common.Hash {
	position := common.Big2
	key := common.BytesToHash(common.LeftPadBytes(position.Bytes(), 32))
	return key
}
