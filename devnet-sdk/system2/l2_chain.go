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

// L2ChainID identifies a L2Chain by name and chainID, is type-safe, and can be value-copied and used as map key.
type L2ChainID idWithChain

func (id L2ChainID) String() string {
	return idWithChain(id).string("L2Chain")
}

func (id L2ChainID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText("L2Chain")
}

func (id *L2ChainID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText("L2Chain", data)
}

func SortL2ChainIDs(ids []L2ChainID) []L2ChainID {
	return copyAndSort(ids, func(a, b L2ChainID) bool {
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

// L2Chain represents a L2 chain, a collection of configuration and node resources.
// There is an extension-interface ExtensibleL2Chain for adding new components to the chain.
type L2Chain interface {
	Chain
	ID() L2ChainID
	RollupConfig() *rollup.Config
	Deployment() L2Deployment
	Keys() L2Keys

	Superchain() Superchain
	L1() L1Chain
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

// ExtensibleL2Chain is an optional extension interface for L2Chain,
// for adding new components to the chain. Used during test-setup, not generally during test execution.
type ExtensibleL2Chain interface {
	L2Chain
	AddL2Batcher(v L2Batcher)
	AddL2Proposer(v L2Proposer)
	AddL2Challenger(v L2Challenger)
	AddL2CLNode(v L2CLNode)
	AddL2ELNode(v L2ELNode)
}

type L2ChainConfig struct {
	ChainConfig
	ID           L2ChainID
	RollupConfig *rollup.Config
	Deployment   L2Deployment
	Keys         L2Keys

	Superchain Superchain
	L1         L1Chain
	Cluster    Cluster
}

type presetL2Chain struct {
	presetChain
	id L2ChainID

	rollupCfg  *rollup.Config
	deployment L2Deployment
	keys       L2Keys

	superchain Superchain
	l1         L1Chain
	cluster    Cluster

	batchers    locks.RWMap[L2BatcherID, L2Batcher]
	proposers   locks.RWMap[L2ProposerID, L2Proposer]
	challengers locks.RWMap[L2ChallengerID, L2Challenger]

	els locks.RWMap[L2ELNodeID, L2ELNode]
	cls locks.RWMap[L2CLNodeID, L2CLNode]
}

var _ L2Chain = (*presetL2Chain)(nil)

func NewL2Chain(cfg L2ChainConfig) ExtensibleL2Chain {
	// sanity-check the configs match the expected chains
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.ChainConfig.ChainCfg.ChainID), "chain config must match expected chain")
	require.Equal(cfg.T, cfg.L1.ChainID(), eth.ChainIDFromBig(cfg.RollupConfig.L1ChainID), "rollup config must match expected L1 chain")
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.RollupConfig.L2ChainID), "rollup config must match expected L2 chain")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetL2Chain{
		id:          cfg.ID,
		presetChain: newChain(cfg.ChainConfig),
		rollupCfg:   cfg.RollupConfig,
		deployment:  cfg.Deployment,
		keys:        cfg.Keys,
		superchain:  cfg.Superchain,
		l1:          cfg.L1,
		cluster:     cfg.Cluster,
	}
}

func (p *presetL2Chain) ID() L2ChainID {
	return p.id
}

func (p *presetL2Chain) RollupConfig() *rollup.Config {
	p.require().NotNil(p.rollupCfg, "l2 chain %s must have a rollup config", p.ID())
	return p.rollupCfg
}

func (p *presetL2Chain) Deployment() L2Deployment {
	p.require().NotNil(p.deployment, "l2 chain %s must have a deployment", p.ID())
	return p.deployment
}

func (p *presetL2Chain) Keys() L2Keys {
	p.require().NotNil(p.keys, "l2 chain %s must have keys", p.ID())
	return p.keys
}

func (p *presetL2Chain) Superchain() Superchain {
	p.require().NotNil(p.superchain, "l2 chain %s must have a superchain", p.ID())
	return p.superchain
}

func (p *presetL2Chain) L1() L1Chain {
	p.require().NotNil(p.l1, "l2 chain %s must have an L1 chain", p.ID())
	return p.l1
}

func (p *presetL2Chain) Cluster() Cluster {
	p.require().NotNil(p.cluster, "l2 chain %s must have a cluster", p.ID())
	return p.cluster
}

func (p *presetL2Chain) L2Batcher(id L2BatcherID) L2Batcher {
	v, ok := p.batchers.Get(id)
	p.require().True(ok, "l2 batcher %s must exist", id)
	return v
}

func (p *presetL2Chain) AddL2Batcher(v L2Batcher) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 batcher %s must be on chain %s", id, p.chainID)
	p.require().True(p.batchers.SetIfMissing(id, v), "l2 batcher %s must not already exist", id)
}

func (p *presetL2Chain) L2Proposer(id L2ProposerID) L2Proposer {
	v, ok := p.proposers.Get(id)
	p.require().True(ok, "l2 proposer %s must exist", id)
	return v
}

func (p *presetL2Chain) AddL2Proposer(v L2Proposer) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 proposer %s must be on chain %s", id, p.chainID)
	p.require().True(p.proposers.SetIfMissing(id, v), "l2 proposer %s must not already exist", id)
}

func (p *presetL2Chain) L2Challenger(id L2ChallengerID) L2Challenger {
	v, ok := p.challengers.Get(id)
	p.require().True(ok, "l2 challenger %s must exist", id)
	return v
}

func (p *presetL2Chain) AddL2Challenger(v L2Challenger) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 challenger %s must be on chain %s", id, p.chainID)
	p.require().True(p.challengers.SetIfMissing(id, v), "l2 challenger %s must not already exist", id)
}

func (p *presetL2Chain) L2CLNode(id L2CLNodeID) L2CLNode {
	v, ok := p.cls.Get(id)
	p.require().True(ok, "l2 CL node %s must exist", id)
	return v
}

func (p *presetL2Chain) AddL2CLNode(v L2CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l2 CL node %s must not already exist", id)
}

func (p *presetL2Chain) L2ELNode(id L2ELNodeID) L2ELNode {
	v, ok := p.els.Get(id)
	p.require().True(ok, "l2 EL node %s must exist", id)
	return v
}

func (p *presetL2Chain) AddL2ELNode(v L2ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l2 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l2 EL node %s must not already exist", id)
}

func (p *presetL2Chain) L2Batchers() []L2BatcherID {
	return SortL2BatcherIDs(p.batchers.Keys())
}

func (p *presetL2Chain) L2Proposers() []L2ProposerID {
	return SortL2ProposerIDs(p.proposers.Keys())
}

func (p *presetL2Chain) L2Challengers() []L2ChallengerID {
	return SortL2ChallengerIDs(p.challengers.Keys())
}

func (p *presetL2Chain) L2CLNodes() []L2CLNodeID {
	return SortL2CLNodeIDs(p.cls.Keys())
}

func (p *presetL2Chain) L2ELNodes() []L2ELNodeID {
	return SortL2ELNodeIDs(p.els.Keys())
}
