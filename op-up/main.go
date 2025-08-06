package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	opDir, ok := os.LookupEnv("OP_DIR")
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get user home dir: %w", err)
		}
		opDir = filepath.Join(homeDir, ".op")
	}
	if err := os.MkdirAll(opDir, 0o755); err != nil {
		return fmt.Errorf("create the op dir: %w", err)
	}
	deployerCacheDir := filepath.Join(opDir, "deployer", "cache")
	if err := os.MkdirAll(deployerCacheDir, 0o755); err != nil {
		return fmt.Errorf("create the deployer cache dir: %w", err)
	}

	ids := sysgo.NewDefaultMinimalSystemIDs(sysgo.DefaultL1ID, sysgo.DefaultL2AID)
	presets.DoMain(testingM{}, stack.MakeCommon(stack.Combine(
		sysgo.WithMnemonicKeys(devkeys.TestMnemonic),

		sysgo.WithDeployer(),
		sysgo.WithDeployerOptions(
			sysgo.WithEmbeddedContractSources(),
			sysgo.WithCommons(ids.L1.ChainID()),
			sysgo.WithPrefundedL2(ids.L1.ChainID(), ids.L2.ChainID()),
		),
		sysgo.WithDeployerPipelineOption(sysgo.WithDeployerCacheDir(deployerCacheDir)),

		sysgo.WithL1Nodes(ids.L1EL, ids.L1CL),

		sysgo.WithL2ELNode(ids.L2EL, nil),
		sysgo.WithL2CLNode(ids.L2CL, true, false, ids.L1CL, ids.L1EL, ids.L2EL),

		sysgo.WithBatcher(ids.L2Batcher, ids.L1EL, ids.L2CL, ids.L2EL),
		sysgo.WithProposer(ids.L2Proposer, ids.L1EL, &ids.L2CL, nil),

		sysgo.WithFaucets([]stack.L1ELNodeID{ids.L1EL}, []stack.L2ELNodeID{ids.L2EL}),
	)))

	return nil
}

type testingM struct{}

var _ presets.TestingM = testingM{}

func (t testingM) Run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	<-ctx.Done()
	return 0
}
