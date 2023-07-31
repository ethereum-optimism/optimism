package ether

import (
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/common"
	"golang.org/x/crypto/sha3"
)

var (
	// Boba proxy legacy slots
	BobaLegacyProxyOwnerSlot          = CalcLegacyProxyKey("proxyOwner", libcommon.Big0)
	BobaLegacyProxyImplementationSlot = CalcLegacyProxyKey("proxyTarget", libcommon.Big0)
)

// BytesBacked is a re-export of the same interface in Geth,
// which is unfortunately private.
type BytesBacked interface {
	Bytes() []byte
}

// CalcAllowanceStorageKey calculates the storage key of an allowance in OVM ETH.
func CalcAllowanceStorageKey(owner libcommon.Address, spender libcommon.Address) libcommon.Hash {
	inner := CalcStorageKey(owner, libcommon.Big1)
	return CalcStorageKey(spender, inner)
}

// CalcOVMETHStorageKey calculates the storage key of an OVM ETH balance.
func CalcOVMETHStorageKey(addr libcommon.Address) libcommon.Hash {
	return CalcStorageKey(addr, libcommon.Big0)
}

func CalcOVMETHTotalSupplyKey() libcommon.Hash {
	position := libcommon.Big2
	key := libcommon.BytesToHash(common.LeftPadBytes(position.Bytes(), 32))
	return key
}

// CalcStorageKey is a helper method to calculate storage keys.
func CalcStorageKey(a, b BytesBacked) libcommon.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(common.LeftPadBytes(a.Bytes(), 32))
	hasher.Write(common.LeftPadBytes(b.Bytes(), 32))
	digest := hasher.Sum(nil)
	return libcommon.BytesToHash(digest)
}

func CalcLegacyProxyKey(a string, b BytesBacked) libcommon.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(a))
	hasher.Write(common.LeftPadBytes(b.Bytes(), 32))
	digest := hasher.Sum(nil)
	return libcommon.BytesToHash(digest)
}
