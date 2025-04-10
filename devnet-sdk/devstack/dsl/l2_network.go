package dsl

import (
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2Network wraps a stack.L2Network interface for DSL operations
type L2Network struct {
	commonImpl
	inner stack.L2Network
}

// NewL2Network creates a new L2Network DSL wrapper
func NewL2Network(inner stack.L2Network) *L2Network {
	return &L2Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (n *L2Network) String() string {
	return n.inner.ID().String()
}

func (n *L2Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L2Network
func (n *L2Network) Escape() stack.L2Network {
	return n.inner
}

func (n *L2Network) WaitForBlock() {
	l2_el := n.inner.L2ELNode(match.FirstL2EL)

	initial, err := l2_el.EthClient().InfoByLabel(n.ctx, "latest")
	n.require.NoError(err, "Expected to get latest block from L2 execution client")

	initialHash := initial.Hash()

	err = wait.For(n.ctx, 500*time.Millisecond, func() (bool, error) {
		latest, err := l2_el.EthClient().InfoByLabel(n.ctx, "latest")
		if err != nil {
			return false, err
		}

		newHash := latest.Hash()

		if initialHash.Cmp(newHash) == 0 {
			n.log.Info("Still same block detected", "initial_block_hash", initialHash, "new_block_hash", newHash)

			return false, nil
		}

		n.log.Info("New block detected", "prev_block_hash", initialHash, "new_block_hash", newHash)
		return true, nil
	})
	n.require.NoError(err, "Expected to get latest block from L2 execution client for comparison")
}

// PrintChain is used for testing/debugging, it prints the blockchain hashes and parent hashes to logs, which is useful when developing reorg tests
func (n *L2Network) PrintChain() {
	l2_el := n.inner.L2ELNode(match.FirstL2EL)
	l2_cl := n.inner.L2CLNode(match.FirstL2CL)

	unsafeHeadRef := n.UnsafeHeadRef()

	var entries []string
	for i := unsafeHeadRef.Number; i > 0; i-- {
		ref, err := l2_el.EthClient().BlockRefByNumber(n.ctx, i)
		n.require.NoError(err, "Expected to get block ref by number")

		entries = append(entries, fmt.Sprintln("Number: ", ref.Number, "Hash: ", ref.Hash.Hex(), "Parent: ", ref.ParentID().Hash.Hex()))
	}

	syncStatus, err := l2_cl.RollupAPI().SyncStatus(n.ctx)
	n.require.NoError(err, "Expected to get sync status")

	entries = append(entries, spew.Sdump(syncStatus))

	n.log.Info("Printing block hashes and parent hashes")
	spew.Dump(entries)
}

func (n *L2Network) UnsafeHeadRef() eth.BlockRef {
	l2_el := n.inner.L2ELNode(match.FirstL2EL)

	unsafeHead, err := l2_el.EthClient().InfoByLabel(n.ctx, "latest")
	n.require.NoError(err, "Expected to get latest block from L2 execution client")

	unsafeHeadRef, err := l2_el.EthClient().BlockRefByHash(n.ctx, unsafeHead.Hash())
	n.require.NoError(err, "Expected to get block ref by hash")

	return unsafeHeadRef
}
