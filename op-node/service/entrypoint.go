package service

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type MainFn func(ctx context.Context, cfg *config.Config, logger log.Logger) (cliapp.Lifecycle, error)

// Main is the entrypoint into the op-node-v2.
// This method returns a cliapp.LifecycleAction, to create an op-service CLI-lifecycle-managed op-node-v2 with.
func Main(version string, fn MainFn) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		// TODO(#16682): due to legacy config constraints
		// we need the logger to be fully initialized before
		// being able to run some of the CLI->config setup code.
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)
		opservice.WarnOnDeprecatedFlags(cliCtx, flags.DeprecatedFlags, l)

		cfg, err := config.NewConfig(cliCtx, l)
		if err != nil {
			return nil, err
		}
		cfg.Version = version
		cfg.Cancel = closeApp
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		// Only pretty-print the banner if it is a terminal log. Otherwise log it as key-value pairs.
		if logCfg.Format == "terminal" {
			l.Info("rollup config:\n" + cfg.Rollup.Description(chaincfg.L2ChainIDToNetworkDisplayName))
		} else {
			cfg.Rollup.LogDescription(l, chaincfg.L2ChainIDToNetworkDisplayName)
		}

		l.Info("Loaded op-node config")
		return fn(cliCtx.Context, cfg, l)
	}
}
