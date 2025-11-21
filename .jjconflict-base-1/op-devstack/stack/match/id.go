package match

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ByID matches element with the given component ID.
func MatchComponentID[E stack.ComponentIdentifiable](id stack.ComponentID) stack.ComponentIDMatcher[E] {
	return &matchComponentID[E]{id: id}
}

type matchComponentID[E stack.ComponentIdentifiable] struct {
	id stack.ComponentID
}

func (m *matchComponentID[E]) Match(elems []E) []E {
	for _, elem := range elems {
		if elem.ID() == m.id {
			return []E{elem}
		}
	}
	return nil
}

func (m *matchComponentID[E]) String() string {
	return fmt.Sprintf("MatchID(%s)", m.id)
}

// ByID matches element with the given component ID.
func MatchChainID[E stack.ChainIdentifiable](id eth.ChainID) stack.ChainIDMatcher[E] {
	return &matchChainID[E]{id: id}
}

type matchChainID[E stack.ChainIdentifiable] struct {
	id eth.ChainID
}

func (m *matchChainID[E]) Match(elems []E) []E {
	for _, elem := range elems {
		if elem.ID() == m.id {
			return []E{elem}
		}
	}
	return nil
}

func (m *matchChainID[E]) String() string {
	return fmt.Sprintf("MatchID(%s)", m.id)
}

var MatchIDL2EL = MatchComponentID[stack.L2ELNode]
var MatchIDL2CL = MatchComponentID[stack.L2CLNode]
var MatchIDL2Batcher = MatchComponentID[stack.L2Batcher]
var MatchIDL2Proposer = MatchComponentID[stack.L2Proposer]
var MatchIDL2Challenger = MatchComponentID[stack.L2Challenger]

var MatchIDTestSequencer = MatchComponentID[stack.TestSequencer]
var MatchIDSupervisor = MatchComponentID[stack.Supervisor]

var MatchIDL1EL = MatchComponentID[stack.L1ELNode]
var MatchIDL1CL = MatchComponentID[stack.L1CLNode]

var MatchIDL1Network = MatchChainID[stack.L1Network]
var MatchIDL2Network = MatchChainID[stack.L2Network]
var MatchIDSuperchain = MatchComponentID[stack.Superchain]

var MatchIDFaucet = MatchComponentID[stack.Faucet]
var MatchIDSyncTester = MatchComponentID[stack.SyncTester]

var MatchIDOPRBuilderNode = MatchComponentID[stack.OPRBuilderNode]
var MatchIDRollupBoostNode = MatchComponentID[stack.RollupBoostNode]
