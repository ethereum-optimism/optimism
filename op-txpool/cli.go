package op_txpool

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	SendRawTransactionConditionalEnabledFlagName   = "sendRawTxConditional.enabled"
	SendRawTransactionConditionalBackendsFlagName  = "sendRawTxConditional.backends"
	SendRawTransactionConditionalRateLimitFlagName = "sendRawTxConditional.ratelimit"
)

type CLIConfig struct {
	SendRawTransactionConditionalEnabled   bool
	SendRawTransactionConditionalBackends  []string
	SendRawTransactionConditionalRateLimit uint64
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    SendRawTransactionConditionalEnabledFlagName,
			Usage:   "Decider if eth_sendRawTransactionConditional requests should passthrough or be rejected",
			Value:   true,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SENDRAWTXCONDITIONAL_ENABLED"),
		},
		&cli.StringSliceFlag{
			Name:    SendRawTransactionConditionalBackendsFlagName,
			Usage:   "List of backends to broadcast conditional transactions",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SENDRAWTXCONDITIONAL_BACKENDS"),
		},
		&cli.Uint64Flag{
			Name:    SendRawTransactionConditionalRateLimitFlagName,
			Usage:   "Maximum cost -- storage lookups -- allowed for conditional transactions in a given second",
			Value:   5000,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "SENDRAWTXCONDITIONAL_RATELIMIT"),
		},
	}
}

// TODO: Entrypoint addresses? somewhere to read preinstalls
func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		SendRawTransactionConditionalEnabled:   ctx.Bool(SendRawTransactionConditionalEnabledFlagName),
		SendRawTransactionConditionalBackends:  ctx.StringSlice(SendRawTransactionConditionalBackendsFlagName),
		SendRawTransactionConditionalRateLimit: ctx.Uint64(SendRawTransactionConditionalRateLimitFlagName),
	}
}
