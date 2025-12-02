package dsl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
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

func (n *L1Network) WaitForBlock() eth.BlockRef {
	return NewL1ELNode(n.inner.L1ELNode(match.FirstL1EL)).WaitForBlock()
}

// PrintChain is used for testing/debugging, it prints the blockchain hashes and parent hashes to logs, which is useful when developing reorg tests
func (n *L1Network) PrintChain() {
	l1_el := n.inner.L1ELNode(match.FirstL1EL)

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
	return NewL1ELNode(n.inner.L1ELNode(match.FirstL1EL)).WaitForFinalization()
}

func (n *L1Network) WaitForOnline() {
	NewL1ELNode(n.inner.L1ELNode(match.FirstL1EL)).WaitForOnline()
}
