package client

import (
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	optls "github.com/ethereum-optimism/optimism/op-service/tls"
)

const (
	EndpointFlagName = "signer.endpoint"
	AddressFlagName  = "signer.address"
)

func CLIFlags(envPrefix string) []cli.Flag {
	envPrefix += "_SIGNER"
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    EndpointFlagName,
			Usage:   "Signer endpoint the client will connect to",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "ENDPOINT"),
		},
		&cli.StringFlag{
			Name:    AddressFlagName,
			Usage:   "Address the signer is signing transactions for",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "ADDRESS"),
		},
	}
	flags = append(flags, optls.CLIFlagsWithFlagPrefix(envPrefix, "signer")...)
	return flags
}

func NewCLIConfig() optls.SignerCLIConfig {
	return optls.SignerCLIConfig{
		TLSConfig: optls.NewCLIConfig(),
	}
}


func ReadCLIConfig(ctx *cli.Context) optls.SignerCLIConfig {
	cfg := optls.SignerCLIConfig{
		Endpoint:  ctx.String(EndpointFlagName),
		Address:   ctx.String(AddressFlagName),
		TLSConfig: optls.ReadCLIConfigWithPrefix(ctx, "signer"),
	}
	return cfg
}
