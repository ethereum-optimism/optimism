package crossdomain

import (
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
)

var (
	offsetAddr = common.HexToAddress("0x1111000000000000000000000000000000001111")
	offsetU256 = new(uint256.Int).SetBytes20(offsetAddr[:])
)

// ApplyL1ToL2Alias will apply the alias applied to L1 to L2 messages when it
// originates from a contract address
func ApplyL1ToL2Alias(address common.Address) common.Address {
	var input uint256.Int
	input.SetBytes20(address[:])
	input.Add(&input, offsetU256)
	// clipping to bytes20 is the same as modulo 160 here, since the modulo is a multiple of 8 bits
	return input.Bytes20()
}

// UndoL1ToL2Alias will remove the alias applied to L1 to L2 messages when it
// originates from a contract address
func UndoL1ToL2Alias(address common.Address) common.Address {
	var input uint256.Int
	input.SetBytes20(address[:])
	input.Sub(&input, offsetU256)
	// clipping to bytes20 is the same as modulo 160 here, since the modulo is a multiple of 8 bits.
	// and underflows are not affected either.
	return input.Bytes20()
}
