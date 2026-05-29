package presets

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func singleSupernodeWithSyncTesterFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *SingleSupernodeWithSyncTester {
	require := t.Require()
	chain := runtime.Chains["l2a"]
	require.NotNil(chain, "missing l2a chain")
	require.NotNil(runtime.Supernode, "missing supernode")
	require.NotNil(runtime.SyncTester, "missing sync tester support")
	require.NotNil(runtime.SyncTester.EL, "missing sync tester EL")
	require.NotNil(runtime.SyncTester.CL, "missing sync tester CL")

	l1ChainID := runtime.L1Network.ChainID()
	l2ChainID := chain.Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1ELFront := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CLFront := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1ELFront)
	l1Network.AddL1CLNode(l1CLFront)

	l2Net := newPresetL2Network(
		t,
		"l2a",
		chain.Network.ChainConfig(),
		chain.Network.RollupConfig(),
		chain.Network.Deployment(),
		newKeyring(runtime.Keys, require),
		l1Network,
	)
	seqEL := newL2ELFrontend(t, "sequencer", l2ChainID,
		chain.EL.UserRPC(), chain.EL.EngineRPC(), chain.EL.JWTPath(),
		chain.Network.RollupConfig())
	seqCL := newL2CLFrontend(t, "sequencer", l2ChainID, chain.CL.UserRPC(), chain.CL)
	seqCL.attachEL(seqEL)
	seqBatcher := newL2BatcherFrontend(t, "main", l2ChainID, chain.Batcher.UserRPC())
	l2Net.AddL2ELNode(seqEL)
	l2Net.AddL2CLNode(seqCL)
	l2Net.AddL2Batcher(seqBatcher)

	faucetL1Front := newFaucetFrontendForChain(t, runtime.FaucetService, l1ChainID)
	faucetL2Front := newFaucetFrontendForChain(t, runtime.FaucetService, l2ChainID)
	l1Network.AddFaucet(faucetL1Front)
	l2Net.AddFaucet(faucetL2Front)
	faucetL1 := dsl.NewFaucet(faucetL1Front)
	faucetL2 := dsl.NewFaucet(faucetL2Front)

	// Sync-tester frontends.
	syncTesterName, syncTesterRPC, ok := runtime.SyncTester.Service.DefaultEndpoint(l2ChainID)
	require.Truef(ok, "missing sync tester for chain %s", l2ChainID)
	syncTesterFront := newSyncTesterFrontend(t, syncTesterName, l2ChainID, syncTesterRPC)
	syncTesterELFront := newL2ELFrontend(
		t, "sync-tester-el", l2ChainID,
		runtime.SyncTester.EL.UserRPC(),
		runtime.SyncTester.EL.EngineRPC(),
		runtime.SyncTester.EL.JWTPath(),
		chain.Network.RollupConfig(),
	)

	// Per-chain supernode VN as the verifier CL.
	supernodeCLFront := newL2CLFrontend(t, "supernode-vn", l2ChainID,
		runtime.SyncTester.CL.UserRPC(), runtime.SyncTester.CL)
	supernodeCLFront.attachEL(syncTesterELFront)

	l2Net.AddSyncTester(syncTesterFront)
	l2Net.AddL2ELNode(syncTesterELFront)
	l2Net.AddL2CLNode(supernodeCLFront)

	supernodeFront := newSupernodeFrontend(t, "supernode-sync-tester-system", runtime.Supernode.UserRPC())

	l1ELDSL := dsl.NewL1ELNode(l1ELFront)
	l1CLDSL := dsl.NewL1CLNode(l1CLFront)
	seqELDSL := dsl.NewL2ELNode(seqEL)
	seqCLDSL := dsl.NewL2CLNode(seqCL)
	supernodeCLDSL := dsl.NewL2CLNode(supernodeCLFront)

	minimal := Minimal{
		Log:        t.Logger(),
		T:          t,
		timeTravel: runtime.TimeTravel,
		L1Network:  dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:       l1ELDSL,
		L1CL:       l1CLDSL,
		L2Chain:    dsl.NewL2Network(l2Net, seqELDSL, seqCLDSL, l1ELDSL, nil, nil),
		L2Batcher:  dsl.NewL2Batcher(seqBatcher),
		L2EL:       seqELDSL,
		L2CL:       seqCLDSL,
		Wallet:     dsl.NewRandomHDWallet(t, 30),
		FaucetL1:   faucetL1,
		FaucetL2:   faucetL2,
	}
	minimal.FunderL1 = dsl.NewFunder(minimal.Wallet, minimal.FaucetL1, minimal.L1EL)
	minimal.FunderL2 = dsl.NewFunder(minimal.Wallet, minimal.FaucetL2, minimal.L2EL)

	genesisTime := chain.Network.RollupConfig().Genesis.L2Time
	preset := &SingleSupernodeWithSyncTester{
		Minimal:               minimal,
		Supernode:             dsl.NewSupernodeWithTestControl(supernodeFront, runtime.Supernode),
		SupernodeCL:           supernodeCLDSL,
		SyncTester:            dsl.NewSyncTester(syncTesterFront),
		SyncTesterL2EL:        dsl.NewL2ELNode(syncTesterELFront),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
	}
	preset.SupernodeCL.ManagePeer(preset.L2CL)
	preset.Supernode.ManageVN(preset.SupernodeCL)
	return preset
}
