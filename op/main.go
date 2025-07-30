package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func main() {
	ids := sysgo.NewDefaultMinimalSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID)
	presets.DoMain(testingM{}, stack.MakeCommon(stack.Combine(
		sysgo.WithMnemonicKeys(devkeys.TestMnemonic),

		sysgo.WithDeployer(),
		sysgo.WithDeployerOptions(
			sysgo.WithEmbeddedContractSources(),
			sysgo.WithCommons(ids.L1.ChainID()),
			sysgo.WithPrefundedL2(ids.L1.ChainID(), ids.L2.ChainID()),
		),

		sysgo.WithL1Nodes(ids.L1EL, ids.L1CL),

		sysgo.WithL2ELNode(ids.L2EL, nil),
		sysgo.WithL2CLNode(ids.L2CL, true, false, ids.L1CL, ids.L1EL, ids.L2EL),

		sysgo.WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		sysgo.WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),

		sysgo.WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),

		sysgo.WithL2Challenger(ids.L2Challenger, ids.L1EL, ids.L1CL, nil, nil, &ids.L2CL, []stack.L2ELNodeID{
			ids.L2EL,
		}),
	)))
}

type testingM struct{}

var _ presets.TestingM = testingM{}

func (t testingM) Run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	<-ctx.Done()
	return 0
}
