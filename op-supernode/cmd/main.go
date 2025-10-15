package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.0.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	// First parse the chains only args
	chains, err := flags.ParseChains(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	// Build the full dynamic flags for the supernode using the chains
	dynamicFlags := flags.FullDynamicFlags(chains)

	// Build the app with the full dynamic flags
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(dynamicFlags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-supernode"
	app.Usage = "Supernode"
	app.Description = "Supernode service that starts up, prints, and exits"

	// Lifecycle Action for the app
	app.Action = cliapp.LifecycleCmd(func(cliCtx *cli.Context, close context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		// Confirm top level supernode flags are set
		if err := flags.CheckRequired(cliCtx); err != nil {
			return nil, err
		}

		// Create supernode the config from the CLI context
		cfg := config.NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		// Create the logger for the app
		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.Handler())

		// Validate the environment variables for the app (requres logs so has to be later)
		opservice.ValidateEnvVars(flags.EnvVarPrefix, dynamicFlags, l)

		// Build virtual Configs from the CLI Context for each chain
		vnCfgs := make(map[eth.ChainID]*opnodecfg.Config)
		for _, chainID := range cfg.Chains {
			// Create a new VirtualCLI for the chain which will serve as an opnode config
			vcli := flags.NewVirtualCLI(cliCtx, chainID)
			// Override the L1 and Beacon addresses for the virtual node
			// Based on the top level L1 and Beacon addresses
			vcli.WithStringOverride(opnodeflags.L1NodeAddr.Name, cfg.L1NodeAddr)
			vcli.WithStringOverride(opnodeflags.BeaconAddr.Name, cfg.L1BeaconAddr)
			// Disable P2P for virtual nodes and set the peerstore and discovery paths to memory
			// this is disabled at the CLI level to allow config construction
			vcli.WithBoolOverride(opnodeflags.DisableP2PName, true)
			vcli.WithStringOverride(opnodeflags.PeerstorePathName, "memory")
			vcli.WithStringOverride(opnodeflags.DiscoveryPathName, "memory")
			cfg, err := opnode.NewConfig(vcli, l)
			if err != nil {
				return nil, fmt.Errorf("failed to create virtual node config: %w", err)
			}
			vnCfgs[eth.ChainIDFromUInt64(chainID)] = cfg
		}

		// Create the supernode, supplying the logger, version, and close function
		// as well as the config and virtual node configs for each chain
		ctx := cliCtx.Context
		sn, err := supernode.New(ctx,
			l,
			Version,
			close,
			cfg,
			vnCfgs)
		if err != nil {
			return nil, fmt.Errorf("failed to create supernode: %w", err)
		}
		return sn, nil
	})

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
