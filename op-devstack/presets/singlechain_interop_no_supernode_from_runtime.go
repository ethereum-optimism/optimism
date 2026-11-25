package presets

import (
	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// singleChainInteropNoSupernodeFromRuntime builds a SingleChainInterop whose super roots
// are served by the single op-node (superroot_atTimestamp) rather than op-supernode. The
// runtime runs no supernode; the op-challenger is wired to the op-node's super-root RPC.
func singleChainInteropNoSupernodeFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *SingleChainInterop {
	l1ChainID := runtime.L1Network.ChainID()
	l2ChainID := runtime.L2Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2Chain := newPresetL2Network(
		t,
		"l2a",
		runtime.L2Network.ChainConfig(),
		runtime.L2Network.RollupConfig(),
		runtime.L2Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)
	l2EL := newL2ELFrontend(t, "sequencer", l2ChainID, runtime.L2EL.UserRPC(), runtime.L2EL.EngineRPC(), runtime.L2EL.JWTPath(), runtime.L2Network.RollupConfig(), runtime.L2EL)
	l2CL := newL2CLFrontend(t, "sequencer", l2ChainID, runtime.L2CL.UserRPC(), runtime.L2CL)
	l2CL.attachEL(l2EL)
	l2Batcher := newL2BatcherFrontend(t, "main", l2ChainID, runtime.L2Batcher.UserRPC())
	l2Chain.AddL2ELNode(l2EL)
	l2Chain.AddL2CLNode(l2CL)
	l2Chain.AddL2Batcher(l2Batcher)

	var challengerCfg *challengerConfig.Config
	if runtime.L2Challenger != nil {
		challengerCfg = runtime.L2Challenger.Config()
	}
	if challengerCfg != nil {
		l2Chain.AddL2Challenger(newPresetL2Challenger(t, "main", l2ChainID, challengerCfg))
	}

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)
	l2ELDSL := dsl.NewL2ELNode(l2EL)
	l2CLDSL := dsl.NewL2CLNode(l2CL)

	out := &SingleChainInterop{
		Log:                           t.Logger(),
		T:                             t,
		timeTravel:                    runtime.TimeTravel,
		SuperRoots:                    dsl.NewOpNodeSuperRoots(l2CLDSL),
		L1Network:                     dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:                          l1ELDSL,
		L1CL:                          l1CLDSL,
		L2ChainA:                      dsl.NewL2Network(l2Chain, l2ELDSL, l2CLDSL, l1ELDSL, nil, nil),
		L2BatcherA:                    dsl.NewL2Batcher(l2Batcher),
		L2ELA:                         l2ELDSL,
		L2CLA:                         l2CLDSL,
		Wallet:                        dsl.NewRandomHDWallet(t, 30),
		challengerConfig:              challengerCfg,
		zkChallengerSuperRootRPCProxy: runtime.ZKChallengerSuperRootRPCProxy,
	}
	out.FunderL1 = newFunderEOA(t, runtime.Keys, out.L1EL, out.Wallet)
	out.FunderA = newFunderEOA(t, runtime.Keys, out.L2ELA, out.Wallet)
	return out
}
