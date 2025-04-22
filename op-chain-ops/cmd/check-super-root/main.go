package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	// RPCEndpointsFlagName defines the flag name for the RPC endpoints.
	RPCEndpointsFlagName = "rpc-endpoints"
)

// Config holds the configuration for the check-super-root command.
type Config struct {
	Logger       log.Logger
	RPCEndpoints []string
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) (*Config, error) {
	rpcs := ctx.StringSlice(RPCEndpointsFlagName)
	if len(rpcs) == 0 {
		return nil, fmt.Errorf("flag %s is required", RPCEndpointsFlagName)
	}
	return &Config{
		Logger:       oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)),
		RPCEndpoints: rpcs,
	}, nil
}

// Main is the entrypoint for the check-super-root command.
func Main(cfg *Config, ctx *cli.Context) error {
	cfg.Logger.Info("Initializing Super Root Check Tool")

	migrator, err := script.NewSuperRootMigrator(cfg.Logger, cfg.RPCEndpoints)
	if err != nil {
		cfg.Logger.Crit("Failed to create SuperRootMigrator", "err", err)
		// Crit already exits, but return error for good measure
		return err
	}

	// Run the migrator logic using the application context
	if err := migrator.Run(ctx.Context); err != nil {
		cfg.Logger.Error("Super root calculation failed", "err", err)
		return err
	}

	cfg.Logger.Info("Super root check tool finished successfully")
	return nil
}

// Flags contains the list of configuration options available to the binary.
var Flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:     RPCEndpointsFlagName,
		Usage:    "Required: List of L2 execution client RPC endpoints (e.g., http://host:port).",
		Required: true,
		EnvVars:  []string{"CHECK_SUPER_ROOT_RPC_ENDPOINTS"},
	},
}

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "check-super-root"
	app.Usage = "Calculates a super root from multiple L2 EL endpoints based on their common finalized state."
	// Combine specific flags with log flags
	app.Flags = append(Flags, oplog.CLIFlags("CHECK_SUPER_ROOT")...)

	app.Action = cliapp.LifecycleCmd(func(ctx *cli.Context, close context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		// Parse config from CLI flags
		cfg, err := NewConfig(ctx)
		if err != nil {
			return nil, err
		}
		// Create a lifecycle that wraps our Main function
		return &superRootLifecycle{
			cfg:   cfg,
			ctx:   ctx,
			close: close,
		}, nil
	})

	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "err", err)
	}
}

type superRootLifecycle struct {
	cfg   *Config
	ctx   *cli.Context
	close context.CancelCauseFunc
}

func (s *superRootLifecycle) Start(ctx context.Context) error {
	// Execute the main function
	err := Main(s.cfg, s.ctx)

	// Signal that the application should terminate, regardless of whether there was an error
	s.close(err)

	// Return the error from Main, if any
	return err
}

func (s *superRootLifecycle) Stop(ctx context.Context) error {
	return nil
}

func (s *superRootLifecycle) Stopped() bool {
	return true
}
