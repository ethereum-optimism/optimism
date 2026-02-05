import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	SafeL2Head(chainId *big.Int) eth.L2BlockRef
}

func (c *simpleChainContainer) SafeL2Head() eth.L2BlockRef {
	panic("TODO")
}

// Interface satisfaction static check
var _ SuperAuthority = (*simpleChainContainer)(nil)
