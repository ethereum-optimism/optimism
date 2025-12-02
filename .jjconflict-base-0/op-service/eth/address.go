package eth

import "github.com/ethereum/go-ethereum/common"

// AddressAsLeftPaddedHash converts an address to a hash by left-padding it with zeros.
// No hashing is performed.
// This was previously known as Address.Hash(),
// but removed from go-ethereum in PR 28228, because the naming was not clear.
func AddressAsLeftPaddedHash(addr common.Address) (out common.Hash) {
	copy(out[32-20:], addr[:])
	return
}
