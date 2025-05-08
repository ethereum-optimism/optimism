package dsl

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Network wraps a stack.L1Network interface for DSL operations
type L1Network struct {
	commonImpl
	inner stack.L1Network
}

// NewL1Network creates a new L1Network DSL wrapper
func NewL1Network(inner stack.L1Network) *L1Network {
	return &L1Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (n *L1Network) String() string {
	return n.inner.ID().String()
}

func (n *L1Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L1Network
func (n *L1Network) Escape() stack.L1Network {
	return n.inner
}

func (n *L1Network) WaitForBlock() {
	l1_el := n.inner.L1ELNode(match.FirstL1EL)

	initial, err := l1_el.EthClient().InfoByLabel(n.ctx, "latest")
	n.require.NoError(err, "Expected to get latest block from L1 execution client")

	err = wait.For(n.ctx, 500*time.Millisecond, func() (bool, error) {
		newBlock, err := l1_el.EthClient().InfoByLabel(n.ctx, "latest")
		if err != nil {
			return false, err
		}

		if initial.Hash().Cmp(newBlock.Hash()) == 0 {
			n.log.Info("Still same L1 block detected as initial", "block", eth.InfoToL1BlockRef(newBlock))

			return false, nil
		}

		n.log.Info("New L1 block detected", "new_block", eth.InfoToL1BlockRef(newBlock), "prev_block", eth.InfoToL1BlockRef(initial))
		return true, nil
	})
	n.require.NoError(err, "Expected to get latest block from L1 execution client for comparison")
}
