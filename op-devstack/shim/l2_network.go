package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type L2NetworkConfig struct {
	NetworkConfig
	ID           stack.L2NetworkID
	RollupConfig *rollup.Config
	Deployment   stack.L2Deployment
	Keys         stack.Keys

	Superchain stack.Superchain
	L1         stack.L1Network
	Cluster    stack.Cluster
}

type presetL2Network struct {
	presetNetwork
	id stack.L2NetworkID

	rollupCfg  *rollup.Config
	deployment stack.L2Deployment
	keys       stack.Keys

	superchain stack.Superchain
	l1         stack.L1Network
	cluster    stack.Cluster

	batchers    locks.RWMap[stack.L2BatcherID, stack.L2Batcher]
	proposers   locks.RWMap[stack.L2ProposerID, stack.L2Proposer]
	challengers locks.RWMap[stack.L2ChallengerID, stack.L2Challenger]

	els locks.RWMap[stack.L2ELNodeID, stack.L2ELNode]
	cls locks.RWMap[stack.L2CLNodeID, stack.L2CLNode]

	conductors locks.RWMap[stack.ConductorID, stack.Conductor]
	fbBuilders locks.RWMap[stack.FlashblocksBuilderID, stack.FlashblocksBuilderNode]
}

var _ stack.L2Network = (*presetL2Network)(nil)

func NewL2Network(cfg L2NetworkConfig) stack.ExtensibleL2Network {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	// sanity-check the configs match the expected chains
	require.Equal(cfg.T, cfg.ID.ChainID(), eth.ChainIDFromBig(cfg.NetworkConfig.ChainConfig.ChainID), "chain config must match expected chain")
	require.Equal(cfg.T, cfg.L1.ChainID(), eth.ChainIDFromBig(cfg.RollupConfig.L1ChainID), "rollup config must match expected L1 chain")
	require.Equal(cfg.T, cfg.ID.ChainID(), eth.ChainIDFromBig(cfg.RollupConfig.L2ChainID), "rollup config must match expected L2 chain")
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

func (p *presetL2Network) ID() stack.L2NetworkID {
	return p.id
}

func (p *presetL2Network) RollupConfig() *rollup.Config {
	p.require().NotNil(p.rollupCfg, "l2 chain %s must have a rollup config", p.ID())
	return p.rollupCfg
}

func (p *presetL2Network) Deployment() stack.L2Deployment {
	p.require().NotNil(p.deployment, "l2 chain %s must have a deployment", p.ID())
	return p.deployment
}

func (p *presetL2Network) Keys() stack.Keys {
	p.require().NotNil(p.keys, "l2 chain %s must have keys", p.ID())
	return p.keys
}

func (p *presetL2Network) Superchain() stack.Superchain {
	p.require().NotNil(p.superchain, "l2 chain %s must have a superchain", p.ID())
	return p.superchain
}

func (p *presetL2Network) L1() stack.L1Network {
	p.require().NotNil(p.l1, "l2 chain %s must have an L1 chain", p.ID())
	return p.l1
}

func (p *presetL2Network) Cluster() stack.Cluster {
	p.require().NotNil(p.cluster, "l2 chain %s must have a cluster", p.ID())
	return p.cluster
}

func (p *presetL2Network) L2Batcher(m stack.L2BatcherMatcher) stack.L2Batcher {
	v, ok := findMatch(m, p.batchers.Get, p.L2Batchers)
	p.require().True(ok, "must find L2 batcher %s", m)
	return v
}

func (p *presetL2Network) AddL2Batcher(v stack.L2Batcher) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 batcher %s must be on chain %s", id, p.chainID)
	p.require().True(p.batchers.SetIfMissing(id, v), "l2 batcher %s must not already exist", id)
}

func (p *presetL2Network) Conductor(m stack.ConductorMatcher) stack.Conductor {
	v, ok := findMatch(m, p.conductors.Get, p.Conductors)
	p.require().True(ok, "must find L2 conductor %s", m)
	return v
}

func (p *presetL2Network) AddConductor(v stack.Conductor) {
	id := v.ID()
	p.require().True(p.conductors.SetIfMissing(id, v), "conductor %s must not already exist", id)
}

func (p *presetL2Network) FlashblocksBuilder(m stack.FlashblocksBuilderMatcher) stack.FlashblocksBuilderNode {
	v, ok := findMatch(m, p.fbBuilders.Get, p.FlashblocksBuilders)
	p.require().True(ok, "must find flashblocks builder %s", m)
	return v
}

func (p *presetL2Network) AddFlashblocksBuilder(v stack.FlashblocksBuilderNode) {
	id := v.ID()
	p.require().True(p.fbBuilders.SetIfMissing(id, v), "flashblocks builder %s must not already exist", id)
}

func (p *presetL2Network) L2Proposer(m stack.L2ProposerMatcher) stack.L2Proposer {
	v, ok := findMatch(m, p.proposers.Get, p.L2Proposers)
	p.require().True(ok, "must find L2 proposer %s", m)
	return v
}

func (p *presetL2Network) AddL2Proposer(v stack.L2Proposer) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 proposer %s must be on chain %s", id, p.chainID)
	p.require().True(p.proposers.SetIfMissing(id, v), "l2 proposer %s must not already exist", id)
}

func (p *presetL2Network) L2Challenger(m stack.L2ChallengerMatcher) stack.L2Challenger {
	v, ok := findMatch(m, p.challengers.Get, p.L2Challengers)
	p.require().True(ok, "must find L2 challenger %s", m)
	return v
}

func (p *presetL2Network) AddL2Challenger(v stack.L2Challenger) {
	id := v.ID()

	p.require().True(p.challengers.SetIfMissing(id, v), "l2 challenger %s must not already exist", id)
}

func (p *presetL2Network) L2CLNode(m stack.L2CLMatcher) stack.L2CLNode {
	v, ok := findMatch(m, p.cls.Get, p.L2CLNodes)
	p.require().True(ok, "must find L2 CL %s", m)
	return v
}

func (p *presetL2Network) AddL2CLNode(v stack.L2CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l2 CL node %s must not already exist", id)
}

func (p *presetL2Network) L2ELNode(m stack.L2ELMatcher) stack.L2ELNode {
	v, ok := findMatch(m, p.els.Get, p.L2ELNodes)
	p.require().True(ok, "must find L2 EL %s", m)
	return v
}

func (p *presetL2Network) AddL2ELNode(v stack.L2ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l2 EL node %s must not already exist", id)
}

func (p *presetL2Network) L2BatcherIDs() []stack.L2BatcherID {
	return stack.SortL2BatcherIDs(p.batchers.Keys())
}

func (p *presetL2Network) L2Batchers() []stack.L2Batcher {
	return stack.SortL2Batchers(p.batchers.Values())
}

func (p *presetL2Network) L2ProposerIDs() []stack.L2ProposerID {
	return stack.SortL2ProposerIDs(p.proposers.Keys())
}

func (p *presetL2Network) L2Proposers() []stack.L2Proposer {
	return stack.SortL2Proposers(p.proposers.Values())
}

func (p *presetL2Network) L2ChallengerIDs() []stack.L2ChallengerID {
	return stack.SortL2ChallengerIDs(p.challengers.Keys())
}

func (p *presetL2Network) L2Challengers() []stack.L2Challenger {
	return stack.SortL2Challengers(p.challengers.Values())
}

func (p *presetL2Network) Conductors() []stack.Conductor {
	output := stack.SortConductors(p.conductors.Values())
	p.require().NotEmpty(output, "l2 chain %s must have at least one conductor", p.ID())
	return output
}

func (p *presetL2Network) FlashblocksBuilders() []stack.FlashblocksBuilderNode {
	return stack.SortFlashblocksBuilders(p.fbBuilders.Values())
}

func (p *presetL2Network) L2CLNodeIDs() []stack.L2CLNodeID {
	return stack.SortL2CLNodeIDs(p.cls.Keys())
}

func (p *presetL2Network) L2CLNodes() []stack.L2CLNode {
	return stack.SortL2CLNodes(p.cls.Values())
}

func (p *presetL2Network) L2ELNodeIDs() []stack.L2ELNodeID {
	return stack.SortL2ELNodeIDs(p.els.Keys())
}

func (p *presetL2Network) L2ELNodes() []stack.L2ELNode {
	return stack.SortL2ELNodes(p.els.Values())
}
