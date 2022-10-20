package ether

import (
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// BytesBacked is a re-export of the same interface in Geth,
// which is unfortunately private.
type BytesBacked interface {
	Bytes() []byte
}

// CalcAllowanceStorageKey calculates the storage key of an allowance in OVM ETH.
func CalcAllowanceStorageKey(owner common.Address, spender common.Address) common.Hash {
	inner := CalcStorageKey(owner, common.Big1)
	return CalcStorageKey(spender, inner)
}

// CalcOVMETHStorageKey calculates the storage key of an OVM ETH balance.
func CalcOVMETHStorageKey(addr common.Address) common.Hash {
	return CalcStorageKey(addr, common.Big0)
}

// CalcStorageKey is a helper method to calculate storage keys.
func CalcStorageKey(a, b BytesBacked) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(common.LeftPadBytes(a.Bytes(), 32))
	hasher.Write(common.LeftPadBytes(b.Bytes(), 32))
	digest := hasher.Sum(nil)
	return common.BytesToHash(digest)
}
