package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2NetworkConfig struct {
	NetworkConfig
	ID           stack.ComponentID
	RollupConfig *rollup.Config
	Deployment   stack.L2Deployment
	Keys         stack.Keys

	Superchain stack.Superchain
	L1         stack.L1Network
	Cluster    stack.Cluster
}

type presetL2Network struct {
	presetNetwork
	id stack.ComponentID

	rollupCfg  *rollup.Config
	deployment stack.L2Deployment
	keys       stack.Keys

	superchain stack.Superchain
	l1         stack.L1Network
	cluster    stack.Cluster
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

func (p *presetL2Network) ID() stack.ComponentID {
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
	getter := func(id stack.ComponentID) (stack.L2Batcher, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2Batcher), true
	}
	v, ok := findMatch(m, getter, p.L2Batchers)
	p.require().True(ok, "must find L2 batcher %s", m)
	return v
}

func (p *presetL2Network) AddL2Batcher(v stack.L2Batcher) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 batcher %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l2 batcher %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) Conductor(m stack.ConductorMatcher) stack.Conductor {
	getter := func(id stack.ComponentID) (stack.Conductor, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.Conductor), true
	}
	v, ok := findMatch(m, getter, p.Conductors)
	p.require().True(ok, "must find L2 conductor %s", m)
	return v
}

func (p *presetL2Network) AddConductor(v stack.Conductor) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "conductor %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) L2Proposer(m stack.L2ProposerMatcher) stack.L2Proposer {
	getter := func(id stack.ComponentID) (stack.L2Proposer, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2Proposer), true
	}
	v, ok := findMatch(m, getter, p.L2Proposers)
	p.require().True(ok, "must find L2 proposer %s", m)
	return v
}

func (p *presetL2Network) AddL2Proposer(v stack.L2Proposer) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 proposer %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l2 proposer %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) L2Challenger(m stack.L2ChallengerMatcher) stack.L2Challenger {
	getter := func(id stack.ComponentID) (stack.L2Challenger, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2Challenger), true
	}
	v, ok := findMatch(m, getter, p.L2Challengers)
	p.require().True(ok, "must find L2 challenger %s", m)
	return v
}

func (p *presetL2Network) AddL2Challenger(v stack.L2Challenger) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l2 challenger %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) L2CLNode(m stack.L2CLMatcher) stack.L2CLNode {
	getter := func(id stack.ComponentID) (stack.L2CLNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2CLNode), true
	}
	v, ok := findMatch(m, getter, p.L2CLNodes)
	p.require().True(ok, "must find L2 CL %s", m)
	return v
}

func (p *presetL2Network) AddL2CLNode(v stack.L2CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 CL node %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l2 CL node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) L2ELNode(m stack.L2ELMatcher) stack.L2ELNode {
	getter := func(id stack.ComponentID) (stack.L2ELNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L2ELNode), true
	}
	v, ok := findMatch(m, getter, p.L2ELNodes)
	p.require().True(ok, "must find L2 EL %s", m)
	return v
}

func (p *presetL2Network) AddL2ELNode(v stack.L2ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l2 EL node %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l2 EL node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) L2BatcherIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2Batcher))
}

func (p *presetL2Network) L2Batchers() []stack.L2Batcher {
	ids := p.registry.IDsByKind(stack.KindL2Batcher)
	result := make([]stack.L2Batcher, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2Batcher))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) L2ProposerIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2Proposer))
}

func (p *presetL2Network) L2Proposers() []stack.L2Proposer {
	ids := p.registry.IDsByKind(stack.KindL2Proposer)
	result := make([]stack.L2Proposer, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2Proposer))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) L2ChallengerIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2Challenger))
}

func (p *presetL2Network) L2Challengers() []stack.L2Challenger {
	ids := p.registry.IDsByKind(stack.KindL2Challenger)
	result := make([]stack.L2Challenger, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2Challenger))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) Conductors() []stack.Conductor {
	ids := p.registry.IDsByKind(stack.KindConductor)
	result := make([]stack.Conductor, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.Conductor))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) L2CLNodeIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2CLNode))
}

func (p *presetL2Network) L2CLNodes() []stack.L2CLNode {
	ids := p.registry.IDsByKind(stack.KindL2CLNode)
	result := make([]stack.L2CLNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2CLNode))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) L2ELNodeIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL2ELNode))
}

func (p *presetL2Network) L2ELNodes() []stack.L2ELNode {
	ids := p.registry.IDsByKind(stack.KindL2ELNode)
	result := make([]stack.L2ELNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L2ELNode))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) RollupBoostNodes() []stack.RollupBoostNode {
	ids := p.registry.IDsByKind(stack.KindRollupBoostNode)
	result := make([]stack.RollupBoostNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.RollupBoostNode))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) OPRBuilderNodes() []stack.OPRBuilderNode {
	ids := p.registry.IDsByKind(stack.KindOPRBuilderNode)
	result := make([]stack.OPRBuilderNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.OPRBuilderNode))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL2Network) AddRollupBoostNode(v stack.RollupBoostNode) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "rollup boost node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) AddOPRBuilderNode(v stack.OPRBuilderNode) {
	id := v.ID()
	_, exists := p.registry.Get(id)
	p.require().False(exists, "OPR builder node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL2Network) OPRBuilderNode(m stack.OPRBuilderNodeMatcher) stack.OPRBuilderNode {
	getter := func(id stack.ComponentID) (stack.OPRBuilderNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.OPRBuilderNode), true
	}
	v, ok := findMatch(m, getter, p.OPRBuilderNodes)
	p.require().True(ok, "must find OPR builder node %s", m)
	return v
}

func (p *presetL2Network) RollupBoostNode(m stack.RollupBoostNodeMatcher) stack.RollupBoostNode {
	getter := func(id stack.ComponentID) (stack.RollupBoostNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.RollupBoostNode), true
	}
	v, ok := findMatch(m, getter, p.RollupBoostNodes)
	p.require().True(ok, "must find rollup boost node %s", m)
	return v
}
