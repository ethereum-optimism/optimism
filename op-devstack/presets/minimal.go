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
)

type Minimal struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network

	L2Chain   *dsl.L2Network
	L2Batcher *dsl.L2Batcher
	L2EL      *dsl.L2ELNode
	L2CLNode  *dsl.L2CLNode

	Wallet *dsl.HDWallet

	Faucet *dsl.Faucet
	Funder *dsl.Funder
}

func (m *Minimal) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func NewMinimal(dest *TestSetup[*Minimal]) stack.CommonOption {
	return stack.Combine(
		stack.MakeCommon(startInProcessMinimal()),
		stack.Finally(func(orch stack.Orchestrator, hook stack.SystemHook) {
			*dest = func(t devtest.T) *Minimal {
				return hydrateMinimal(t, orch, hook)
			}
		}))
}

func startInProcessMinimal() stack.Option[*sysgo.Orchestrator] {
	var ids sysgo.DefaultInteropSystemIDs
	// TODO: this is way overkill for a minimal system
	return sysgo.DefaultInteropSystem(&ids)
}

func hydrateMinimal(t devtest.T, orch stack.Orchestrator, hook stack.SystemHook) *Minimal {
	system := shim.NewSystem(t)
	orch.Hydrate(system)
	hook.PostHydrate(system)

	l2 := system.L2Network(match.Assume(t, match.L2ChainA))
	out := &Minimal{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(system.L1Network(match.FirstL1Network)),
		L2Chain:      dsl.NewL2Network(l2),
		L2Batcher:    dsl.NewL2Batcher(l2.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2EL:         dsl.NewL2ELNode(l2.L2ELNode(match.Assume(t, match.FirstL2EL))),
		L2CLNode:     dsl.NewL2CLNode(l2.L2CLNode(match.Assume(t, match.FirstL2CL))),
		Wallet:       dsl.NewHDWallet(t, devkeys.TestMnemonic, 30),
		Faucet:       dsl.NewFaucet(l2.Faucet(match.Assume(t, match.FirstFaucet))),
	}
	out.Funder = dsl.NewFunder(out.Wallet, out.Faucet, out.L2EL)
	return out
}
