package p2p

import (
	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MultiChainFilter struct {
	Allowed []eth.ChainID
	Log     log.Logger
}

var _ p2p.NodeFilter = (*MultiChainFilter)(nil)

// Allow filters the node record. If it returns false then the node is not accepted.
func (f *MultiChainFilter) Allow(node *enode.Node) bool {
	var dat p2p.OpStackENRData
	err := node.Load(&dat)
	// if the entry does not exist, or if it is invalid, then ignore the node
	if err != nil {
		f.Log.Trace("discovered node record has no opstack info", "node", node.ID(), "err", err)
		return false
	}
	chainID := eth.ChainIDFromUInt64(dat.ChainID)
	// check chain ID is allowed
	if !slices.Contains(f.Allowed, chainID) {
		f.Log.Trace("discovered node record has no allowed chain ID", "node", node.ID(), "got", dat.ChainID)
		return false
	}
	// check version matches
	if dat.Version != 0 {
		f.Log.Trace("discovered node record has no matching version", "node", node.ID(), "got", dat.Version, "expected", 0)
		return false
	}
	return true
}
