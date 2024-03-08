package batcher

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// Main is the entry point for the Batch Submitter CLI application. It returns a cliapp.LifecycleAction,
// which is used to create a batch-submitter service managed by the op-service CLI lifecycle.
func Main(version string) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		// Check if all required flags are provided.
		if err := flags.CheckRequired(cliCtx); err != nil {
			return nil, err
		}
		
		// Create a new configuration based on the CLI context.
		cfg := NewConfig(cliCtx)
		
		// Validate the configuration to ensure it meets all requirements.
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		// Initialize a new logger with the specified output and log configuration.
		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		
		// Set the global log handler to the newly created logger.
		oplog.SetGlobalLogHandler(l.Handler())
		
		// Validate environment variables related to the Batch Submitter.
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)

		// Log the initialization of the Batch Submitter.
		l.Info("Initializing Batch Submitter")
		
		// Create and return the BatcherService based on the CLI context, version, and configuration.
		return BatcherServiceFromCLIConfig(cliCtx.Context, version, cfg, l)
	}
}
