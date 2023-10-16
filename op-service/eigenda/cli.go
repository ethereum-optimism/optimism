package eigenda

import (
	"errors"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	RPCFlagName                      = "da-rpc"
	StatusQueryRetryIntervalFlagName = "da-status-query-retry-interval"
	StatusQueryTimeoutFlagName       = "da-status-query-timeout"
)

type CLIConfig struct {
	RPC                      string
	StatusQueryRetryInterval time.Duration
	StatusQueryTimeout       time.Duration
}

// NewConfig parses the Config from the provided flags or environment variables.
func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		/* Required Flags */
		RPC:                      ctx.String(RPCFlagName),
		StatusQueryRetryInterval: ctx.Duration(StatusQueryRetryIntervalFlagName),
		StatusQueryTimeout:       ctx.Duration(StatusQueryTimeoutFlagName),
	}
}

func (m CLIConfig) Check() error {
	if m.RPC == "" {
		return errors.New("must provide a DA RPC url")
	}
	if m.StatusQueryTimeout == 0 {
		return errors.New("DA status query timeout must be greater than 0")
	}
	if m.StatusQueryRetryInterval == 0 {
		return errors.New("DA status query retry interval must be greater than 0")
	}
	return nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	prefixEnvVars := func(name string) []string {
		return opservice.PrefixEnvVar(envPrefix, name)
	}
	return []cli.Flag{
		&cli.StringFlag{
			Name:    RPCFlagName,
			Usage:   "RPC endpoint of the EigenDA disperser",
			EnvVars: prefixEnvVars("DA_RPC"),
		},
		&cli.DurationFlag{
			Name:    StatusQueryTimeoutFlagName,
			Usage:   "Timeout for aborting an EigenDA blob dispersal if the disperser does not report that the blob has been confirmed dispersed.",
			Value:   1 * time.Minute,
			EnvVars: prefixEnvVars("DA_STATUS_QUERY_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    StatusQueryRetryIntervalFlagName,
			Usage:   "Wait time between retries of EigenDA blob status queries (made while waiting for a blob to be confirmed by)",
			Value:   5 * time.Second,
			EnvVars: prefixEnvVars("DA_STATUS_QUERY_INTERVAL"),
		},
	}
}
