package sysgo

import (
	"fmt"
	"os"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	sv2proxy "github.com/ethereum-optimism/optimism/sv2-proxy"
)

// WithSV2TwoChainMinimalDepthProxy composes a minimal two-chain setup without CLs and starts a single SV2 across both chains,
// using a custom L1 confirmation depth for cross-safety gating. This is a local copy of the minimal preset so we can
// later modify it to route through an external sv2-proxy.
func WithSV2TwoChainMinimalDepthProxy(offset uint64, depth uint64) stack.Option[*Orchestrator] {
	// gate to assert the L2 network count after hydration
	gateTwo := stack.PostHydrate[*Orchestrator](func(sys stack.System) {
		sys.T().Gate().Lenf(sys.L2Networks(), 2, "Must have exactly %v chains", 2)
	})
	return stack.Combine[*Orchestrator](
		DefaultTwoMinimalSystemNoCL(&DefaultTwoMinimalSystemIDs{}),
		// ensure Interop2 activation is configured on rollup cfgs before SV2 starts
		WithInterop2ActivationOffsetForSV2(offset),
		// Before starting SV2, place proxies in front of each L2 EL user/auth RPC and rewrite endpoints
		stack.AfterDeploy(func(orch *Orchestrator) {
			ids := stack.SortL2ELNodeIDs(orch.l2ELs.Keys())
			for _, id := range ids {
				l2el, _ := orch.l2ELs.Get(id)
				// Create a composite proxy per L2 EL
				comp, err := sv2proxy.StartELToSV2Proxy(orch.P().Ctx(), l2el.userRPC, l2el.authRPC)
				orch.p.Require().NoError(err)
				orch.p.Cleanup(func() { _ = comp.Close(orch.P().Ctx()) })
				// Rewrite EL RPC endpoints to point to the composite proxy endpoints so SV2 connects via the proxies
				l2el.userRPC = comp.DownstreamUserURL
				l2el.authRPC = comp.DownstreamAuthURL
			}
		}),
		// Now start SV2 across all chains with the proxied endpoints
		WithSupervisorV2OnAllChainsConfirmDepth(depth),
		// Configure batchers to use SV2 /opnode/{chainId}/ proxy (set RollupRpc override)
		WithBatcherOption(func(id stack.L2BatcherID, cfg *bss.CLIConfig) {
			if v, ok := id.ChainID().Uint64(); ok {
				sv2URL := os.Getenv("SV2_DENYLIST_URL")
				cfg.RollupRpc = []string{fmt.Sprintf("%s/opnode/%d/", sv2URL, v)}
			}
		}),
		// start batchers in AfterDeploy (SV2 HTTP + CL shims are ready)
		stack.AfterDeploy(func(orch *Orchestrator) {
			orch.P().Logger().Info("Starting batchers for SV2 two-chain (after-deploy)")
			optA := WithBatcher(stack.NewL2BatcherID("main", DefaultL2AID), stack.NewL1ELNodeID("l1", DefaultL1ID), stack.NewL2CLNodeID("embedded", DefaultL2AID), stack.NewL2ELNodeID("sequencer", DefaultL2AID))
			optA.AfterDeploy(orch)
			optB := WithBatcher(stack.NewL2BatcherID("main", DefaultL2BID), stack.NewL1ELNodeID("l1", DefaultL1ID), stack.NewL2CLNodeID("embedded", DefaultL2BID), stack.NewL2ELNodeID("sequencer", DefaultL2BID))
			optB.AfterDeploy(orch)
		}),
		gateTwo,
	)
}
