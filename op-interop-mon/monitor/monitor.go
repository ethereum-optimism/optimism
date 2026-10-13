package monitor

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-interop-mon/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

func Main(version string, opts ...Option) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		if err := flags.CheckRequired(cliCtx); err != nil {
			return nil, err
		}
		cfg := NewConfig(cliCtx)
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := logcli.NewLogger(logcli.AppOut(cliCtx), cfg.LogConfig)
		logcli.SetGlobalLogHandler(l.Handler())
		o := applyOptions(opts)
		opservice.ValidateEnvVars(flags.EnvVarPrefix, append(flags.Flags, o.extraFlags...), l)

		service, err := InteropMonitorServiceFromCLIConfig(cliCtx.Context, version, cfg, l, opts...)
		if err != nil {
			return nil, err
		}
		l.Info("starting monitor service")
		return service, nil
	}
}
