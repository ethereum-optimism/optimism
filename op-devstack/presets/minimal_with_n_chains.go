package presets

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MinimalWithNChains struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2Chains   []*dsl.L2Network
	L2Batchers []*dsl.L2Batcher
	L2ELs      []*dsl.L2ELNode
	L2CLs      []*dsl.L2CLNode

	TestSequencer *dsl.TestSequencer

	Wallet *dsl.HDWallet

	FaucetL1 *dsl.Faucet
	FaucetL2 []*dsl.Faucet
	FunderL1 *dsl.Funder
	FunderL2 []*dsl.Funder
}

func (m *MinimalWithNChains) L2Networks() []*dsl.L2Network {
	return m.L2Chains
}

func (m *MinimalWithNChains) StandardBridge() *dsl.StandardBridge {
	if len(m.L2Chains) == 0 {
		return nil
	}
	return dsl.NewStandardBridge(m.T, m.L2Chains[0], nil, m.L1EL)
}

func WithMinimalWithNChains(n int) stack.CommonOption {
	return stack.MakeCommon(createMinimalWithNChainsSystem(n))
}

func createMinimalWithNChainsSystem(n int) stack.Option[*sysgo.Orchestrator] {
	l1ID := eth.ChainIDFromUInt64(900)

	// Create chain IDs for N L2 chains starting from 901
	l2ChainIDs := make([]eth.ChainID, n)
	for i := 0; i < n; i++ {
		l2ChainIDs[i] = eth.ChainIDFromUInt64(901 + uint64(i))
	}

	opt := stack.Combine[*sysgo.Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *sysgo.Orchestrator) {
		o.P().Logger().Info("Setting up minimal system with N chains", "n", n)
	}))

	opt.Add(sysgo.WithMnemonicKeys(devkeys.TestMnemonic))

	// Add deployer with commons and prefunded L2 chains
	deployerOpts := []sysgo.DeployerOption{
		sysgo.WithLocalContractSources(),
		sysgo.WithCommons(l1ID),
	}

	// Add prefunded L2 for each chain
	for _, l2ID := range l2ChainIDs {
		deployerOpts = append(deployerOpts, sysgo.WithPrefundedL2(l1ID, l2ID))
	}

	opt.Add(sysgo.WithDeployer(), sysgo.WithDeployerOptions(deployerOpts...))

	// Add L1 nodes
	l1ELID := stack.NewL1ELNodeID("l1", l1ID)
	l1CLID := stack.NewL1CLNodeID("l1", l1ID)
	opt.Add(sysgo.WithL1Nodes(l1ELID, l1CLID))

	// Add L2 nodes for each chain
	for _, l2ID := range l2ChainIDs {
		l2ELID := stack.NewL2ELNodeID("sequencer", l2ID)
		l2CLID := stack.NewL2CLNodeID("sequencer", l2ID)
		l2BatcherID := stack.NewL2BatcherID("main", l2ID)
		l2ProposerID := stack.NewL2ProposerID("main", l2ID)
		l2ChallengerID := stack.NewL2ChallengerID("main", l2ID)

		opt.Add(sysgo.WithL2ELNode(l2ELID, nil))
		opt.Add(sysgo.WithL2CLNode(l2CLID, true, false, l1CLID, l1ELID, l2ELID))
		opt.Add(sysgo.WithBatcher(l2BatcherID, l1ELID, l2CLID, l2ELID))
		opt.Add(sysgo.WithProposer(l2ProposerID, l1ELID, &l2CLID, nil))
		opt.Add(sysgo.WithL2Challenger(l2ChallengerID, l1ELID, l1CLID, nil, nil, &l2CLID, []stack.L2ELNodeID{l2ELID}))
	}

	// Add test sequencer (using first L2 chain)
	if len(l2ChainIDs) > 0 {
		testSequencerID := stack.TestSequencerID("test-sequencer")
		firstL2CLID := stack.NewL2CLNodeID("sequencer", l2ChainIDs[0])
		firstL2ELID := stack.NewL2ELNodeID("sequencer", l2ChainIDs[0])
		opt.Add(sysgo.WithTestSequencer(testSequencerID, l1CLID, firstL2CLID, l1ELID, firstL2ELID))
	}

	// Add faucets for all L2 chains
	var l2ELIDs []stack.L2ELNodeID
	for _, l2ID := range l2ChainIDs {
		l2ELIDs = append(l2ELIDs, stack.NewL2ELNodeID("sequencer", l2ID))
	}
	opt.Add(sysgo.WithFaucets([]stack.L1ELNodeID{l1ELID}, l2ELIDs))

	return opt
}

func NewMinimalWithNChains(t devtest.T, n int) *MinimalWithNChains {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	t.Gate().Equal(len(system.TestSequencers()), 1, "expected exactly one test sequencer")

	l1Net := system.L1Network(match.FirstL1Network)
	l2Networks := system.L2Networks()

	t.Gate().Equal(len(l2Networks), n, "expected exactly %d L2 networks", n)

	out := &MinimalWithNChains{
		Log:           t.Logger(),
		T:             t,
		ControlPlane:  orch.ControlPlane(),
		L1Network:     dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L1EL:          dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2Chains:      make([]*dsl.L2Network, n),
		L2Batchers:    make([]*dsl.L2Batcher, n),
		L2ELs:         make([]*dsl.L2ELNode, n),
		L2CLs:         make([]*dsl.L2CLNode, n),
		TestSequencer: dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer))),
		Wallet:        dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		FaucetL2:      make([]*dsl.Faucet, n),
		FunderL2:      make([]*dsl.Funder, n),
	}

	// Initialize each L2 chain and its components
	for i, l2Net := range l2Networks {
		chainMatcher := match.L2ChainById(l2Net.ID())
		l2 := system.L2Network(match.Assume(t, chainMatcher))

		out.L2Chains[i] = dsl.NewL2Network(l2)
		out.L2Batchers[i] = dsl.NewL2Batcher(l2.L2Batcher(match.Assume(t, match.FirstL2Batcher)))
		out.L2ELs[i] = dsl.NewL2ELNode(l2.L2ELNode(match.Assume(t, match.FirstL2EL)))
		out.L2CLs[i] = dsl.NewL2CLNode(l2.L2CLNode(match.Assume(t, match.FirstL2CL)), orch.ControlPlane())
		out.FaucetL2[i] = dsl.NewFaucet(l2.Faucet(match.Assume(t, match.FirstFaucet)))
	}

	out.FaucetL1 = dsl.NewFaucet(out.L1Network.Escape().Faucet(match.Assume(t, match.FirstFaucet)))
	out.FunderL1 = dsl.NewFunder(out.Wallet, out.FaucetL1, out.L1EL)

	// Create funders for each L2 chain
	for i := range out.L2Chains {
		out.FunderL2[i] = dsl.NewFunder(out.Wallet, out.FaucetL2[i], out.L2ELs[i])
	}

	return out
}
