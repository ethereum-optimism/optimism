package crossdomain

import (
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
)

var (
	uint160Max, _ = uint256.FromHex("0xffffffffffffffffffffffffffffffffffffffff")
	offset        = new(uint256.Int).SetBytes(common.HexToAddress("0x1111000000000000000000000000000000001111").Bytes())
)

// ApplyL1ToL2Alias will apply the alias applied to L1 to L2 messages when it
// originates from a contract address
func ApplyL1ToL2Alias(address *common.Address) *common.Address {
	input := new(uint256.Int).SetBytes(address.Bytes())
	output := new(uint256.Int).AddMod(input, offset, uint160Max)
	if output.Cmp(input) < 0 {
		output = output.Sub(output, new(uint256.Int).SetUint64(1))
	}
	addr := common.BigToAddress(output.ToBig())
	return &addr
}

// UndoL1ToL2Alias will remove the alias applied to L1 to L2 messages when it
// originates from a contract address
func UndoL1ToL2Alias(address *common.Address) *common.Address {
	input := new(uint256.Int).SetBytes(address.Bytes())
	output := new(uint256.Int).Sub(input, offset)
	addr := common.BigToAddress(output.ToBig())
	return &addr
}
