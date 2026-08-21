package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

// ProxyAdmin binds the subset of the ProxyAdmin (L1 contract and L2 predeploy at 0x42...18)
// needed to identify the account authorized to run proxy-admin-owner-gated calls.
type ProxyAdmin struct {
	// Read-only functions
	Owner func() TypedCall[common.Address] `sol:"owner"`
}
