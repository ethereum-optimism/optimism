package dsl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L1Network wraps a stack.L1Network interface for DSL operations
type L1Network struct {
	commonImpl
	inner     stack.L1Network
	primaryEL *L1ELNode
	primaryCL *L1CLNode
}

// NewL1Network creates a new L1Network DSL wrapper
func NewL1Network(inner stack.L1Network, primaryEL *L1ELNode, primaryCL *L1CLNode) *L1Network {
	return &L1Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		primaryEL:  primaryEL,
		primaryCL:  primaryCL,
	}
}

func (n *L1Network) String() string {
	return n.inner.Name()
}

func (n *L1Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L1Network
func (n *L1Network) Escape() stack.L1Network {
	return n.inner
}

func (n *L1Network) PrimaryEL() *L1ELNode {
	n.require.NotNil(n.primaryEL, "l1 network %s is missing a primary EL node", n.String())
	return n.primaryEL
}

func (n *L1Network) PrimaryCL() *L1CLNode {
	n.require.NotNil(n.primaryCL, "l1 network %s is missing a primary CL node", n.String())
	return n.primaryCL
}

func (n *L1Network) WaitForBlock() eth.BlockRef {
	return n.PrimaryEL().WaitForBlock()
}

// PrintChain is used for testing/debugging, it prints the blockchain hashes and parent hashes to logs, which is useful when developing reorg tests
func (n *L1Network) PrintChain() {
	l1_el := n.PrimaryEL().Escape()

	unsafeHeadRef, err := l1_el.EthClient().InfoByLabel(n.ctx, "latest")
	n.require.NoError(err, "Expected to get latest block from L1 execution client")

	var entries []string
	for i := unsafeHeadRef.NumberU64(); i > 0; i-- {
		ref, txs, err := l1_el.EthClient().InfoAndTxsByNumber(n.ctx, i)
		n.require.NoError(err, "Expected to get block ref by number")

		entries = append(entries, fmt.Sprintf("Time: %d Block: %s Txs: %d Parent: %s", ref.Time(), eth.InfoToL1BlockRef(ref), len(txs), ref.ParentHash()))
	}

	n.log.Info("Printing block hashes and parent hashes", "network", n.String(), "chain", n.ChainID())
	spew.Dump(entries)
}

func (n *L1Network) WaitForFinalization() eth.BlockRef {
	return n.PrimaryEL().WaitForFinalization()
}

func (n *L1Network) WaitForOnline() {
	n.PrimaryEL().WaitForOnline()
}
