package stack

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// L2NetworkID identifies a L2Network by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2NetworkID idWithChain

const L2NetworkKind Kind = "L2Network"

func (id L2NetworkID) String() string {
	return idWithChain(id).string(L2NetworkKind)
}

func (id L2NetworkID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L2NetworkKind)
}

func (id *L2NetworkID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L2NetworkKind, data)
}

func SortL2NetworkIDs(ids []L2NetworkID) []L2NetworkID {
	return copyAndSort(ids, func(a, b L2NetworkID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

type L2Deployment interface {
	SystemConfigProxyAddr() common.Address
	DisputeGameFactoryProxyAddr() common.Address
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
	ID() L2NetworkID
	RollupConfig() *rollup.Config
	Deployment() L2Deployment
	Keys() Keys

	Superchain() Superchain
	L1() L1Network
	Cluster() Cluster

	L2Batcher(id L2BatcherID) L2Batcher
	L2Proposer(id L2ProposerID) L2Proposer
	L2Challenger(id L2ChallengerID) L2Challenger
	L2CLNode(id L2CLNodeID) L2CLNode
	L2ELNode(id L2ELNodeID) L2ELNode

	L2Batchers() []L2BatcherID
	L2Proposers() []L2ProposerID
	L2Challengers() []L2ChallengerID
	L2CLNodes() []L2CLNodeID
	L2ELNodes() []L2ELNodeID
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
}
