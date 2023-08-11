package kms

import (
	"errors"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	KmsKeyIDName    = "kms.keyid"
	KmsEndpointName = "kms.endpoint"
	KmsRegionName   = "kms.region"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    KmsKeyIDName,
			Usage:   "KMS Key ID",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "KMS_KEY_ID"),
		},
		&cli.StringFlag{
			Name:    KmsEndpointName,
			Usage:   "KMS Endpoint",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "KMS_ENDPOINT"),
		},
		&cli.StringFlag{
			Name:    KmsRegionName,
			Usage:   "KMS Region",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "KMS_REGION"),
		},
	}
}

type CLIConfig struct {
	KmsKeyID    string
	KmsEndpoint string
	KmsRegion   string
}

func (c CLIConfig) Check() error {
	if c.KmsKeyID != "" {
		if c.KmsEndpoint == "" {
			return errors.New("KMS Endpoint must be provided")
		}
		if c.KmsRegion == "" {
			return errors.New("KMS Region must be provided")
		}
	}

	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		KmsKeyID:    ctx.String(KmsKeyIDName),
		KmsEndpoint: ctx.String(KmsEndpointName),
		KmsRegion:   ctx.String(KmsRegionName),
	}
}
