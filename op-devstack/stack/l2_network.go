package stack

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type L2Deployment interface {
	SystemConfigProxyAddr() common.Address
	DisputeGameFactoryProxyAddr() common.Address
	L1StandardBridgeProxyAddr() common.Address
	// Other addresses will be added here later
}

type Keys interface {
	Secret(key devkeys.Key) *ecdsa.PrivateKey
	Address(key devkeys.Key) common.Address
}

// L2Network represents a L2 chain, a collection of configuration and node resources.
// There is an extension-interface ExtensibleL2Network for adding new components to the chain.
type L2Network interface {
	Network
	ID() ComponentID
	RollupConfig() *rollup.Config
	Deployment() L2Deployment
	Keys() Keys

	Superchain() Superchain
	L1() L1Network
	Cluster() Cluster

	L2Batcher(m L2BatcherMatcher) L2Batcher
	L2Proposer(m L2ProposerMatcher) L2Proposer
	L2Challenger(m L2ChallengerMatcher) L2Challenger
	L2CLNode(m L2CLMatcher) L2CLNode
	L2ELNode(m L2ELMatcher) L2ELNode
	Conductor(m ConductorMatcher) Conductor
	RollupBoostNode(m RollupBoostNodeMatcher) RollupBoostNode
	OPRBuilderNode(m OPRBuilderNodeMatcher) OPRBuilderNode

	L2BatcherIDs() []ComponentID
	L2ProposerIDs() []ComponentID
	L2ChallengerIDs() []ComponentID
	L2CLNodeIDs() []ComponentID
	L2ELNodeIDs() []ComponentID

	L2Batchers() []L2Batcher
	L2Proposers() []L2Proposer
	L2Challengers() []L2Challenger
	L2CLNodes() []L2CLNode
	L2ELNodes() []L2ELNode
	Conductors() []Conductor
	RollupBoostNodes() []RollupBoostNode
	OPRBuilderNodes() []OPRBuilderNode
}

// ExtensibleL2Network is an optional extension interface for L2Network,
// for adding new components to the chain. Used during test-setup, not generally during test execution.
type ExtensibleL2Network interface {
	ExtensibleNetwork
	L2Network
	AddL2Batcher(v L2Batcher)
	AddL2Proposer(v L2Proposer)
	AddL2Challenger(v L2Challenger)
	AddL2CLNode(v L2CLNode)
	AddL2ELNode(v L2ELNode)
	AddConductor(v Conductor)
	AddRollupBoostNode(v RollupBoostNode)
	AddOPRBuilderNode(v OPRBuilderNode)
}
