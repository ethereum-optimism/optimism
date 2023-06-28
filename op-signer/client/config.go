package client

import (
	"errors"

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

type CLIConfig struct {
	Endpoint  string
	Address   string
	TLSConfig optls.CLIConfig
}

func (c CLIConfig) Check() error {
	if err := c.TLSConfig.Check(); err != nil {
		return err
	}
	if !((c.Endpoint == "" && c.Address == "") || (c.Endpoint != "" && c.Address != "")) {
		return errors.New("signer endpoint and address must both be set or not set")
	}
	return nil
}

func (c CLIConfig) Enabled() bool {
	if c.Endpoint != "" && c.Address != "" {
		return true
	}
	return false
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	cfg := CLIConfig{
		Endpoint:  ctx.String(EndpointFlagName),
		Address:   ctx.String(AddressFlagName),
		TLSConfig: optls.ReadCLIConfigWithPrefix(ctx, "signer"),
	}
	return cfg
}
