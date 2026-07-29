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
	// TimestampFlagName defines the flag name for the optional target timestamp.
	TimestampFlagName = "timestamp"
)

// Config holds the configuration for the check-super-root command.
type Config struct {
	Logger          log.Logger
	RPCEndpoints    []string
	TargetTimestamp *uint64 // Optional target timestamp
}

// NewConfig parses the Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) (*Config, error) {
	rpcs := ctx.StringSlice(RPCEndpointsFlagName)
	if len(rpcs) == 0 {
		return nil, fmt.Errorf("flag %s is required", RPCEndpointsFlagName)
	}

	var targetTimestamp *uint64
	if ctx.IsSet(TimestampFlagName) {
		ts := ctx.Uint64(TimestampFlagName)
		targetTimestamp = &ts
	}

	return &Config{
		Logger:          oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx)),
		RPCEndpoints:    rpcs,
		TargetTimestamp: targetTimestamp,
	}, nil
}

// Main is the entrypoint for the check-super-root command.
func Main(cfg *Config, ctx *cli.Context) error {
	migrator, err := script.NewSuperRootMigrator(cfg.Logger, cfg.RPCEndpoints, cfg.TargetTimestamp)
	if err != nil {
		return fmt.Errorf("failed to create SuperRootMigrator: %w", err)
	}

	if _, err := migrator.Run(ctx.Context); err != nil {
		return err
	}

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
	&cli.Uint64Flag{
		Name:    TimestampFlagName,
		Usage:   "Optional: Target timestamp for super root calculation. If not set, uses the latest common finalized block timestamp.",
		EnvVars: []string{"CHECK_SUPER_ROOT_TIMESTAMP"},
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
