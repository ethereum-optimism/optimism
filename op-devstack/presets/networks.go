package presets

import (
	"slices"
	"sort"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

type presetCommon struct {
	log    log.Logger
	t      devtest.T
	req    *testreq.Assertions
	labels *locks.RWMap[string, string]
	name   string
}

func newPresetCommon(t devtest.T, name string) presetCommon {
	return presetCommon{
		log:    t.Logger(),
		t:      t,
		req:    t.Require(),
		labels: new(locks.RWMap[string, string]),
		name:   name,
	}
}

func (c *presetCommon) T() devtest.T {
	return c.t
}

func (c *presetCommon) Logger() log.Logger {
	return c.log
}

func (c *presetCommon) Name() string {
	return c.name
}

func (c *presetCommon) Label(key string) string {
	out, _ := c.labels.Get(key)
	return out
}

func (c *presetCommon) SetLabel(key, value string) {
	c.labels.Set(key, value)
}

func (c *presetCommon) require() *testreq.Assertions {
	return c.req
}

type presetNetworkBase struct {
	presetCommon
	chainCfg *params.ChainConfig
	chainID  eth.ChainID

	faucets     []*faucetFrontend
	syncTesters []*syncTesterFrontend
}

func (n *presetNetworkBase) ChainID() eth.ChainID {
	return n.chainID
}

func (n *presetNetworkBase) ChainConfig() *params.ChainConfig {
	return n.chainCfg
}

func (n *presetNetworkBase) Faucets() []stack.Faucet {
	return mapSlice(sortByNameFunc(n.faucets), func(v *faucetFrontend) stack.Faucet { return v })
}

func (n *presetNetworkBase) AddFaucet(v *faucetFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "faucet %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.faucets, v.Name())
	n.require().False(exists, "faucet %s must not already exist", v.Name())
	n.faucets = append(n.faucets, v)
}

func (n *presetNetworkBase) SyncTesters() []stack.SyncTester {
	return mapSlice(sortByNameFunc(n.syncTesters), func(v *syncTesterFrontend) stack.SyncTester { return v })
}

func (n *presetNetworkBase) AddSyncTester(v *syncTesterFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "sync tester %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.syncTesters, v.Name())
	n.require().False(exists, "sync tester %s must not already exist", v.Name())
	n.syncTesters = append(n.syncTesters, v)
}

type presetL1Network struct {
	presetNetworkBase

	l1ELNodes []*l1ELFrontend
	l1CLNodes []*l1CLFrontend
}

var _ stack.L1Network = (*presetL1Network)(nil)

func newPresetL1Network(t devtest.T, name string, chainCfg *params.ChainConfig) *presetL1Network {
	chainID := eth.ChainIDFromBig(chainCfg.ChainID)
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	t.Require().NotEmpty(name, "l1 network name must not be empty")
	return &presetL1Network{
		presetNetworkBase: presetNetworkBase{
			presetCommon: newPresetCommon(t, name),
			chainCfg:     chainCfg,
			chainID:      chainID,
		},
	}
}

func (n *presetL1Network) AddL1ELNode(v *l1ELFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l1 EL node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l1ELNodes, v.Name())
	n.require().False(exists, "l1 EL node %s must not already exist", v.Name())
	n.l1ELNodes = append(n.l1ELNodes, v)
}

func (n *presetL1Network) AddL1CLNode(v *l1CLFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l1 CL node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l1CLNodes, v.Name())
	n.require().False(exists, "l1 CL node %s must not already exist", v.Name())
	n.l1CLNodes = append(n.l1CLNodes, v)
}

func (n *presetL1Network) L1ELNodes() []stack.L1ELNode {
	return mapSlice(sortByNameFunc(n.l1ELNodes), func(v *l1ELFrontend) stack.L1ELNode { return v })
}

func (n *presetL1Network) L1CLNodes() []stack.L1CLNode {
	return mapSlice(sortByNameFunc(n.l1CLNodes), func(v *l1CLFrontend) stack.L1CLNode { return v })
}

type presetL2Network struct {
	presetNetworkBase

	rollupCfg  *rollup.Config
	deployment stack.L2Deployment
	keys       *keyringImpl

	l1 *presetL1Network

	l2Batchers       []*l2BatcherFrontend
	l2Proposers      []*l2ProposerFrontend
	l2Challengers    []*l2ChallengerFrontend
	l2CLNodes        []*l2CLFrontend
	l2ELNodes        []*l2ELFrontend
	conductors       []*conductorFrontend
	rollupBoostNodes []*rollupBoostFrontend
	oprBuilderNodes  []*oprBuilderFrontend
}

var _ stack.L2Network = (*presetL2Network)(nil)

func newPresetL2Network(
	t devtest.T,
	name string,
	chainCfg *params.ChainConfig,
	rollupCfg *rollup.Config,
	deployment stack.L2Deployment,
	keys *keyringImpl,
	l1 *presetL1Network,
) *presetL2Network {
	chainID := eth.ChainIDFromBig(chainCfg.ChainID)
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	t.Require().NotEmpty(name, "l2 network name must not be empty")
	t.Require().Equal(l1.ChainID(), eth.ChainIDFromBig(rollupCfg.L1ChainID), "rollup config must match expected L1 chain")
	t.Require().Equal(chainID, eth.ChainIDFromBig(rollupCfg.L2ChainID), "rollup config must match expected L2 chain")
	return &presetL2Network{
		presetNetworkBase: presetNetworkBase{
			presetCommon: newPresetCommon(t, name),
			chainCfg:     chainCfg,
			chainID:      chainID,
		},
		rollupCfg:  rollupCfg,
		deployment: deployment,
		keys:       keys,
		l1:         l1,
	}
}

func (n *presetL2Network) RollupConfig() *rollup.Config {
	n.require().NotNil(n.rollupCfg, "l2 chain %s must have a rollup config", n.Name())
	return n.rollupCfg
}

func (n *presetL2Network) Deployment() stack.L2Deployment {
	n.require().NotNil(n.deployment, "l2 chain %s must have a deployment", n.Name())
	return n.deployment
}

func (n *presetL2Network) Keys() stack.Keys {
	n.require().NotNil(n.keys, "l2 chain %s must have keys", n.Name())
	return n.keys
}

func (n *presetL2Network) L1() stack.L1Network {
	n.require().NotNil(n.l1, "l2 chain %s must have an L1 chain", n.Name())
	return n.l1
}

func (n *presetL2Network) AddL2Batcher(v *l2BatcherFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l2 batcher %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l2Batchers, v.Name())
	n.require().False(exists, "l2 batcher %s must not already exist", v.Name())
	n.l2Batchers = append(n.l2Batchers, v)
}

func (n *presetL2Network) AddL2Proposer(v *l2ProposerFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l2 proposer %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l2Proposers, v.Name())
	n.require().False(exists, "l2 proposer %s must not already exist", v.Name())
	n.l2Proposers = append(n.l2Proposers, v)
}

func (n *presetL2Network) AddL2Challenger(v *l2ChallengerFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l2 challenger %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l2Challengers, v.Name())
	n.require().False(exists, "l2 challenger %s must not already exist", v.Name())
	n.l2Challengers = append(n.l2Challengers, v)
}

func (n *presetL2Network) AddL2CLNode(v *l2CLFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l2 CL node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l2CLNodes, v.Name())
	n.require().False(exists, "l2 CL node %s must not already exist", v.Name())
	n.l2CLNodes = append(n.l2CLNodes, v)
}

func (n *presetL2Network) AddL2ELNode(v *l2ELFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "l2 EL node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.l2ELNodes, v.Name())
	n.require().False(exists, "l2 EL node %s must not already exist", v.Name())
	n.l2ELNodes = append(n.l2ELNodes, v)
}

func (n *presetL2Network) AddConductor(v *conductorFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "conductor %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.conductors, v.Name())
	n.require().False(exists, "conductor %s must not already exist", v.Name())
	n.conductors = append(n.conductors, v)
}

func (n *presetL2Network) AddRollupBoostNode(v *rollupBoostFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "rollup boost node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.rollupBoostNodes, v.Name())
	n.require().False(exists, "rollup boost node %s must not already exist", v.Name())
	n.rollupBoostNodes = append(n.rollupBoostNodes, v)
}

func (n *presetL2Network) AddOPRBuilderNode(v *oprBuilderFrontend) {
	n.require().Equal(n.chainID, v.ChainID(), "OPR builder node %s must be on chain %s", v.Name(), n.chainID)
	_, exists := componentByName(n.oprBuilderNodes, v.Name())
	n.require().False(exists, "OPR builder node %s must not already exist", v.Name())
	n.oprBuilderNodes = append(n.oprBuilderNodes, v)
}

func (n *presetL2Network) L2Batchers() []stack.L2Batcher {
	return mapSlice(sortByNameFunc(n.l2Batchers), func(v *l2BatcherFrontend) stack.L2Batcher { return v })
}

func (n *presetL2Network) L2Proposers() []stack.L2Proposer {
	return mapSlice(sortByNameFunc(n.l2Proposers), func(v *l2ProposerFrontend) stack.L2Proposer { return v })
}

func (n *presetL2Network) L2Challengers() []stack.L2Challenger {
	return mapSlice(sortByNameFunc(n.l2Challengers), func(v *l2ChallengerFrontend) stack.L2Challenger { return v })
}

func (n *presetL2Network) L2CLNodes() []stack.L2CLNode {
	return mapSlice(sortByNameFunc(n.l2CLNodes), func(v *l2CLFrontend) stack.L2CLNode { return v })
}

func (n *presetL2Network) L2ELNodes() []stack.L2ELNode {
	return mapSlice(sortByNameFunc(n.l2ELNodes), func(v *l2ELFrontend) stack.L2ELNode { return v })
}

func (n *presetL2Network) Conductors() []stack.Conductor {
	return mapSlice(sortByNameFunc(n.conductors), func(v *conductorFrontend) stack.Conductor { return v })
}

func (n *presetL2Network) RollupBoostNodes() []stack.RollupBoostNode {
	return mapSlice(sortByNameFunc(n.rollupBoostNodes), func(v *rollupBoostFrontend) stack.RollupBoostNode { return v })
}

func (n *presetL2Network) OPRBuilderNodes() []stack.OPRBuilderNode {
	return mapSlice(sortByNameFunc(n.oprBuilderNodes), func(v *oprBuilderFrontend) stack.OPRBuilderNode { return v })
}

type named interface {
	Name() string
}

func componentByName[T named](components []T, name string) (T, bool) {
	for _, component := range components {
		if component.Name() == name {
			return component, true
		}
	}
	var zero T
	return zero, false
}

func sortByNameFunc[T named](components []T) []T {
	out := slices.Clone(components)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

func mapSlice[T any, U any](items []T, mapFn func(T) U) []U {
	out := make([]U, len(items))
	for i, item := range items {
		out[i] = mapFn(item)
	}
	return out
}
