package deployer

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// Primitive ABI types primarily used with `abi.Arguments` to pack/unpack values when calling contract methods.
var (
	Uint256Type = opcm.MustType("uint256")
	BytesType   = opcm.MustType("bytes")
	AddressType = opcm.MustType("address")
	Bytes32Type = opcm.MustType("bytes32")
)
