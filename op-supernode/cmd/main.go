package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
	Version   = "v0.1.1"
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

		// Create supernode config from the CLI context
		cfg := config.NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		// Create the logger for the app
		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.Handler())

		// Validate the environment variables for the app
		opservice.ValidateEnvVars(flags.EnvVarPrefix, dynamicFlags, l)

		// Build virtual Configs from the CLI Context for each chain
		vnCfgs, err := createVirtualNodeConfigs(cliCtx, cfg, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual node configs: %w", err)
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

func createVirtualNodeConfigs(cliCtx *cli.Context, cfg *config.CLIConfig, l log.Logger) (map[eth.ChainID]*opnodecfg.Config, error) {
	vnCfgs := make(map[eth.ChainID]*opnodecfg.Config)
	for _, chainID := range cfg.Chains {
		// Create a new VirtualCLI for the chain which will serve as an opnode config
		vcli := flags.NewVirtualCLI(cliCtx, chainID)
		// force P2P settings to match top level DisableP2P flag
		if cliCtx.Bool(flags.DisableP2P.Name) {
			if err := withNoP2P(vcli); err != nil {
				return nil, fmt.Errorf("failed to disable P2P for chain %d: %w", chainID, err)
			}
		} else {
			if err := withNamespacedP2P(vcli, cfg.DataDir, strconv.FormatUint(chainID, 10)); err != nil {
				return nil, fmt.Errorf("failed to configure P2P for chain %d: %w", chainID, err)
			}
		}
		// Override the L1 and Beacon addresses for the virtual node
		// Based on the top level L1 and Beacon addresses
		vcli.WithStringOverride(opnodeflags.L1NodeAddr.Name, cfg.L1NodeAddr)
		vcli.WithStringOverride(opnodeflags.BeaconAddr.Name, cfg.L1BeaconAddr)
		cfg, err := opnode.NewConfig(vcli, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual node config: %w", err)
		}
		vnCfgs[eth.ChainIDFromUInt64(chainID)] = cfg
	}
	return vnCfgs, nil
}

func withNoP2P(vcli *flags.VirtualCLI) error {
	vcli.WithBoolOverride(opnodeflags.DisableP2PName, true)
	vcli.WithStringOverride(opnodeflags.P2PPrivPathName, "")
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, "")
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, "")
	vcli.WithUintOverride(opnodeflags.ListenTCPPortName, 0)
	vcli.WithUintOverride(opnodeflags.ListenUDPPortName, 0)
	return nil
}

func withNamespacedP2P(vcli *flags.VirtualCLI, datadir string, namespace string) error {
	// Configure per-VN P2P using namespaced DataDir and dynamic ports
	p2pDir := filepath.Join(datadir, namespace, "p2p")
	// Ensure per-VN p2p directory exists for key and databases
	if err := os.MkdirAll(p2pDir, 0o700); err != nil {
		return fmt.Errorf("failed creating p2p dir for chain %s: %w", namespace, err)
	}
	vcli.WithStringOverride(opnodeflags.P2PPrivPathName, filepath.Join(p2pDir, "opnode_p2p_priv.txt"))
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, filepath.Join(p2pDir, "peerstore_db"))
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, filepath.Join(p2pDir, "discovery_db"))
	// Force dynamic TCP/UDP listen ports to avoid collisions
	vcli.WithUintOverride(opnodeflags.ListenTCPPortName, 0)
	vcli.WithUintOverride(opnodeflags.ListenUDPPortName, 0)
	return nil
}
