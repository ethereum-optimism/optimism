package resources

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	SafeL2Head(chainId *big.Int) eth.L2BlockRef
}

// SupernodeAuthority is the supernode's implementation of SuperAuthority.
// It provides coordination and authority functions to op-node instances
// running within the supernode.
type SupernodeAuthority struct {
	chainContainers map[*big.Int]chain_container.ChainContainer
}

// NewSupernodeAuthority creates a new SupernodeAuthority instance.
func NewSupernodeAuthority() *SupernodeAuthority {
	return &SupernodeAuthority{
		chainContainers: chainContainers,
	}
}

func (s *SupernodeAuthority) SafeL2Head(chainId *big.Int) eth.L2BlockRef {
	panic("TODO")
}

// Interface conformance assertion
var _ SuperAuthority = (*SupernodeAuthority)(nil)
