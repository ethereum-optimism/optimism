package system2

import (
	"crypto/ecdsa"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
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
	// Other addresses will be added here later
}

type L2Keys interface {
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
	Keys() L2Keys

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

type L2NetworkConfig struct {
	NetworkConfig
	ID           L2NetworkID
	RollupConfig *rollup.Config
	Deployment   L2Deployment
	Keys         L2Keys

	Superchain Superchain
	L1         L1Network
	Cluster    Cluster
}

type presetL2Network struct {
	presetNetwork
	id L2NetworkID

	rollupCfg  *rollup.Config
	deployment L2Deployment
	keys       L2Keys

	superchain Superchain
	l1         L1Network
	cluster    Cluster

	batchers    locks.RWMap[L2BatcherID, L2Batcher]
	proposers   locks.RWMap[L2ProposerID, L2Proposer]
	challengers locks.RWMap[L2ChallengerID, L2Challenger]

	els locks.RWMap[L2ELNodeID, L2ELNode]
	cls locks.RWMap[L2CLNodeID, L2CLNode]
}

var _ L2Network = (*presetL2Network)(nil)

func NewL2Network(cfg L2NetworkConfig) ExtensibleL2Network {
	// sanity-check the configs match the expected chains
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.NetworkConfig.ChainConfig.ChainID), "chain config must match expected chain")
	require.Equal(cfg.T, cfg.L1.ChainID(), eth.ChainIDFromBig(cfg.RollupConfig.L1ChainID), "rollup config must match expected L1 chain")
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.RollupConfig.L2ChainID), "rollup config must match expected L2 chain")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetL2Network{
		id:            cfg.ID,
		presetNetwork: newNetwork(cfg.NetworkConfig),
		rollupCfg:     cfg.RollupConfig,
		deployment:    cfg.Deployment,
		keys:          cfg.Keys,
		superchain:    cfg.Superchain,
		l1:            cfg.L1,
		cluster:       cfg.Cluster,
	}
}

func (p *presetL2Network) ID() L2NetworkID {
	return p.id
}

func (p *presetL2Network) RollupConfig() *rollup.Config {
	p.require().NotNil(p.rollupCfg, "l2 chain %s must have a rollup config", p.ID())
	return p.rollupCfg
}

func (p *presetL2Network) Deployment() L2Deployment {
	p.require().NotNil(p.deployment, "l2 chain %s must have a deployment", p.ID())
	return p.deployment
}

func (p *presetL2Network) Keys() L2Keys {
	p.require().NotNil(p.keys, "l2 chain %s must have keys", p.ID())
	return p.keys
}

func (p *presetL2Network) Superchain() Superchain {
	p.require().NotNil(p.superchain, "l2 chain %s must have a superchain", p.ID())
	return p.superchain
}

func (p *presetL2Network) L1() L1Network {
	p.require().NotNil(p.l1, "l2 chain %s must have an L1 chain", p.ID())
	return p.l1
}

func (p *presetL2Network) Cluster() Cluster {
	p.require().NotNil(p.cluster, "l2 chain %s must have a cluster", p.ID())
	return p.cluster
}

func (p *presetL2Network) L2Batcher(id L2BatcherID) L2Batcher {
	v, ok := p.batchers.Get(id)
	p.require().True(ok, "l2 batcher %s must exist", id)
	return v
}

func (p *presetL2Network) AddL2Batcher(v L2Batcher) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 batcher %s must be on chain %s", id, p.chainID)
	p.require().True(p.batchers.SetIfMissing(id, v), "l2 batcher %s must not already exist", id)
}

func (p *presetL2Network) L2Proposer(id L2ProposerID) L2Proposer {
	v, ok := p.proposers.Get(id)
	p.require().True(ok, "l2 proposer %s must exist", id)
	return v
}

func (p *presetL2Network) AddL2Proposer(v L2Proposer) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 proposer %s must be on chain %s", id, p.chainID)
	p.require().True(p.proposers.SetIfMissing(id, v), "l2 proposer %s must not already exist", id)
}

func (p *presetL2Network) L2Challenger(id L2ChallengerID) L2Challenger {
	v, ok := p.challengers.Get(id)
	p.require().True(ok, "l2 challenger %s must exist", id)
	return v
}

func (p *presetL2Network) AddL2Challenger(v L2Challenger) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 challenger %s must be on chain %s", id, p.chainID)
	p.require().True(p.challengers.SetIfMissing(id, v), "l2 challenger %s must not already exist", id)
}

func (p *presetL2Network) L2CLNode(id L2CLNodeID) L2CLNode {
	v, ok := p.cls.Get(id)
	p.require().True(ok, "l2 CL node %s must exist", id)
	return v
}

func (p *presetL2Network) AddL2CLNode(v L2CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l2 CL node %s must not already exist", id)
}

func (p *presetL2Network) L2ELNode(id L2ELNodeID) L2ELNode {
	v, ok := p.els.Get(id)
	p.require().True(ok, "l2 EL node %s must exist", id)
	return v
}

func (p *presetL2Network) AddL2ELNode(v L2ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l2 EL node %s must not already exist", id)
}

func (p *presetL2Network) L2Batchers() []L2BatcherID {
	return SortL2BatcherIDs(p.batchers.Keys())
}

func (p *presetL2Network) L2Proposers() []L2ProposerID {
	return SortL2ProposerIDs(p.proposers.Keys())
}

func (p *presetL2Network) L2Challengers() []L2ChallengerID {
	return SortL2ChallengerIDs(p.challengers.Keys())
}

func (p *presetL2Network) L2CLNodes() []L2CLNodeID {
	return SortL2CLNodeIDs(p.cls.Keys())
}

func (p *presetL2Network) L2ELNodes() []L2ELNodeID {
	return SortL2ELNodeIDs(p.els.Keys())
}
