package op_heartbeat

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-heartbeat/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/urfave/cli/v2"
)

type Config struct {
	HTTPAddr string
	HTTPPort int

	Log oplog.CLIConfig

	Metrics opmetrics.CLIConfig

	Pprof oppprof.CLIConfig
}

func (c Config) Check() error {
	if c.HTTPAddr == "" {
		return errors.New("must specify a valid HTTP address")
	}
	if c.HTTPPort <= 0 {
		return errors.New("must specify a valid HTTP port")
	}
	if err := c.Log.Check(); err != nil {
		return err
	}
	if err := c.Metrics.Check(); err != nil {
		return err
	}
	if err := c.Pprof.Check(); err != nil {
		return err
	}
	return nil
}

func NewConfig(ctx *cli.Context) Config {
	return Config{
		HTTPAddr: ctx.String(flags.HTTPAddrFlag.Name),
		HTTPPort: ctx.Int(flags.HTTPPortFlag.Name),
		Log:      oplog.ReadCLIConfig(ctx),
		Metrics:  opmetrics.ReadCLIConfig(ctx),
		Pprof:    oppprof.ReadCLIConfig(ctx),
	}
}
