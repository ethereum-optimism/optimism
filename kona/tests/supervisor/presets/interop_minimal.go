package presets

import (
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
)

// WithSimpleInteropMinimal specifies a system that meets the SimpleInterop criteria removing the Challenger.
func WithSimpleInteropMinimal() stack.CommonOption {
	return stack.MakeCommon(DefaultMinimalInteropSystem(&sysgo.DefaultInteropSystemIDs{}))
}

func DefaultMinimalInteropSystem(dest *sysgo.DefaultInteropSystemIDs) stack.Option[*sysgo.Orchestrator] {
	ids := sysgo.NewDefaultInteropSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID, sysgo.DefaultL2BID)
	opt := stack.Combine[*sysgo.Orchestrator]()

	// start with single chain interop system
	opt.Add(baseInteropSystem(&ids.DefaultSingleChainInteropSystemIDs))

	opt.Add(sysgo.WithDeployerOptions(
		sysgo.WithPrefundedL2(ids.L1.ChainID(), ids.L2B.ChainID()),
		sysgo.WithInteropAtGenesis(), // this can be overridden by later options
	))
	opt.Add(sysgo.WithL2ELNode(ids.L2BEL, sysgo.L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(sysgo.WithL2CLNode(ids.L2BCL, ids.L1CL, ids.L1EL, ids.L2BEL, sysgo.L2CLSequencer(), sysgo.L2CLIndexing()))
	opt.Add(sysgo.WithBatcher(ids.L2BBatcher, ids.L1EL, ids.L2BCL, ids.L2BEL))

	opt.Add(sysgo.WithManagedBySupervisor(ids.L2BCL, ids.Supervisor))

	// Note: we provide L2 CL nodes still, even though they are not used post-interop.
	// Since we may create an interop infra-setup, before interop is even scheduled to run.
	opt.Add(sysgo.WithProposer(ids.L2BProposer, ids.L1EL, &ids.L2BCL, &ids.Supervisor))

	opt.Add(sysgo.WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2AEL, ids.L2BEL}))

	// Upon evaluation of the option, export the contents we created.
	// Ids here are static, but other things may be exported too.
	opt.Add(stack.Finally(func(orch *sysgo.Orchestrator) {
		*dest = ids
	}))

	return opt
}

// baseInteropSystem defines a system that supports interop with a single chain
// Components which are shared across multiple chains are not started, allowing them to be added later including
// any additional chains that have been added.
func baseInteropSystem(ids *sysgo.DefaultSingleChainInteropSystemIDs) stack.Option[*sysgo.Orchestrator] {
	opt := stack.Combine[*sysgo.Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *sysgo.Orchestrator) {
		o.P().Logger().Info("Setting up")
	}))

	opt.Add(sysgo.WithMnemonicKeys(devkeys.TestMnemonic))

	// Get artifacts path
	artifactsPath := os.Getenv("OP_DEPLOYER_ARTIFACTS")
	if artifactsPath == "" {
		panic("OP_DEPLOYER_ARTIFACTS is not set")
	}

	opt.Add(sysgo.WithDeployer(),
		sysgo.WithDeployerPipelineOption(
			sysgo.WithDeployerCacheDir(artifactsPath),
		),
		sysgo.WithDeployerOptions(
			func(_ devtest.P, _ devkeys.Keys, builder intentbuilder.Builder) {
				builder.WithL1ContractsLocator(artifacts.MustNewFileLocator(filepath.Join(artifactsPath, "src")))
				builder.WithL2ContractsLocator(artifacts.MustNewFileLocator(filepath.Join(artifactsPath, "src")))
			},
			sysgo.WithCommons(ids.L1.ChainID()),
			sysgo.WithPrefundedL2(ids.L1.ChainID(), ids.L2A.ChainID()),
		),
	)

	opt.Add(sysgo.WithL1Nodes(ids.L1EL, ids.L1CL))

	opt.Add(sysgo.WithSupervisor(ids.Supervisor, ids.Cluster, ids.L1EL))

	opt.Add(sysgo.WithL2ELNode(ids.L2AEL, sysgo.L2ELWithSupervisor(ids.Supervisor)))
	opt.Add(sysgo.WithL2CLNode(ids.L2ACL, ids.L1CL, ids.L1EL, ids.L2AEL, sysgo.L2CLSequencer(), sysgo.L2CLIndexing()))
	opt.Add(sysgo.WithTestSequencer(ids.TestSequencer, ids.L1CL, ids.L2ACL, ids.L1EL, ids.L2AEL))
	opt.Add(sysgo.WithBatcher(ids.L2ABatcher, ids.L1EL, ids.L2ACL, ids.L2AEL))

	opt.Add(sysgo.WithManagedBySupervisor(ids.L2ACL, ids.Supervisor))

	// Note: we provide L2 CL nodes still, even though they are not used post-interop.
	// Since we may create an interop infra-setup, before interop is even scheduled to run.
	opt.Add(sysgo.WithProposer(ids.L2AProposer, ids.L1EL, &ids.L2ACL, &ids.Supervisor))
	return opt
}
