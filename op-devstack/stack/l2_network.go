package stack

import (
	"crypto/ecdsa"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// L2NetworkID identifies a L2Network by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2NetworkID idOnlyChainID

var _ IDOnlyChainID = (*L2NetworkID)(nil)

const L2NetworkKind Kind = "L2Network"

func (id L2NetworkID) ChainID() eth.ChainID {
	return eth.ChainID(id)
}

func (id L2NetworkID) Kind() Kind {
	return L2NetworkKind
}

func (id L2NetworkID) String() string {
	return idOnlyChainID(id).string(L2NetworkKind)
}

func (id L2NetworkID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

func (id L2NetworkID) MarshalText() ([]byte, error) {
	return idOnlyChainID(id).marshalText(L2NetworkKind)
}

func (id *L2NetworkID) UnmarshalText(data []byte) error {
	return (*idOnlyChainID)(id).unmarshalText(L2NetworkKind, data)
}

func SortL2NetworkIDs(ids []L2NetworkID) []L2NetworkID {
	return copyAndSort(ids, func(a, b L2NetworkID) bool {
		return lessIDOnlyChainID(idOnlyChainID(a), idOnlyChainID(b))
	})
}

func SortL2Networks(elems []L2Network) []L2Network {
	return copyAndSort(elems, func(a, b L2Network) bool {
		return lessIDOnlyChainID(idOnlyChainID(a.ID()), idOnlyChainID(b.ID()))
	})
}

var _ L2NetworkMatcher = L2NetworkID{}

func (id L2NetworkID) Match(elems []L2Network) []L2Network {
	return findByID(id, elems)
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

	L2Batcher(m L2BatcherMatcher) L2Batcher
	L2Proposer(m L2ProposerMatcher) L2Proposer
	L2Challenger(m L2ChallengerMatcher) L2Challenger
	L2CLNode(m L2CLMatcher) L2CLNode
	L2ELNode(m L2ELMatcher) L2ELNode
	Conductor(m ConductorMatcher) Conductor

	L2BatcherIDs() []L2BatcherID
	L2ProposerIDs() []L2ProposerID
	L2ChallengerIDs() []L2ChallengerID
	L2CLNodeIDs() []L2CLNodeID
	L2ELNodeIDs() []L2ELNodeID

	L2Batchers() []L2Batcher
	L2Proposers() []L2Proposer
	L2Challengers() []L2Challenger
	L2CLNodes() []L2CLNode
	L2ELNodes() []L2ELNode
	Conductors() []Conductor
	FlashblocksBuilders() []FlashblocksBuilderNode
	AddFlashblocksBuilder(v FlashblocksBuilderNode)
	FlashblocksWebsocketProxies() []FlashblocksWebsocketProxy
	AddFlashblocksWebsocketProxy(v FlashblocksWebsocketProxy)
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
}
