package eigenda

import (
	"errors"
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	RPCFlagName                      = "eigenda-rpc"
	StatusQueryRetryIntervalFlagName = "eigenda-status-query-retry-interval"
	StatusQueryTimeoutFlagName       = "eigenda-status-query-timeout"
)

type Config struct {
	// TODO(eigenlayer): Update quorum ID command-line parameters to support passing
	// and arbitrary number of quorum IDs.

	// RPC is the HTTP provider URL for the Data Availability node.
	RPC string

	// The total amount of time that the batcher will spend waiting for EigenDA to confirm a blob
	StatusQueryTimeout time.Duration

	// The amount of time to wait between status queries of a newly dispersed blob
	StatusQueryRetryInterval time.Duration
}

// NewConfig parses the Config from the provided flags or environment variables.
func ReadConfig(ctx *cli.Context) Config {
	return Config{
		/* Required Flags */
		RPC:                      ctx.String(RPCFlagName),
		StatusQueryRetryInterval: ctx.Duration(StatusQueryRetryIntervalFlagName),
		StatusQueryTimeout:       ctx.Duration(StatusQueryTimeoutFlagName),
	}
}

func (m Config) Check() error {
	if m.StatusQueryTimeout == 0 {
		return errors.New("EigenDA status query timeout must be greater than 0")
	}
	if m.StatusQueryRetryInterval == 0 {
		return errors.New("EigenDA status query retry interval must be greater than 0")
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
			EnvVars: prefixEnvVars("EIGENDA_RPC"),
		},
		&cli.DurationFlag{
			Name:    StatusQueryTimeoutFlagName,
			Usage:   "Timeout for aborting an EigenDA blob dispersal if the disperser does not report that the blob has been confirmed dispersed.",
			Value:   1 * time.Minute,
			EnvVars: prefixEnvVars("EIGENDA_STATUS_QUERY_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    StatusQueryRetryIntervalFlagName,
			Usage:   "Wait time between retries of EigenDA blob status queries (made while waiting for a blob to be confirmed by)",
			Value:   5 * time.Second,
			EnvVars: prefixEnvVars("EIGENDA_STATUS_QUERY_INTERVAL"),
		},
	}
}
