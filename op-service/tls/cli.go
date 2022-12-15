// This file contains CLI and env TLS configurations that can be used by clients or servers
package tls

import (
	"errors"

	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	TLSCaCertFlagName = "tls.ca"
	TLSCertFlagName   = "tls.cert"
	TLSKeyFlagName    = "tls.key"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   TLSCaCertFlagName,
			Usage:  "tls ca cert path",
			Value:  "tls/ca.crt",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TLS_CA"),
		},
		cli.StringFlag{
			Name:   TLSCertFlagName,
			Usage:  "tls cert path",
			Value:  "tls/tls.crt",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TLS_CERT"),
		},
		cli.StringFlag{
			Name:   TLSKeyFlagName,
			Usage:  "tls key",
			Value:  "tls/tls.key",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TLS_KEY"),
		},
	}
}

type CLIConfig struct {
	TLSCaCert string
	TLSCert   string
	TLSKey    string
}

func (c CLIConfig) Check() error {
	if c.TLSEnabled() && (c.TLSCaCert == "" || c.TLSCert == "" || c.TLSKey == "") {
		return errors.New("all tls flags must be set if at least one is set")
	}

	return nil
}

func (c CLIConfig) TLSEnabled() bool {
	return !(c.TLSCaCert == "" && c.TLSCert == "" && c.TLSKey == "")
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		TLSCaCert: ctx.GlobalString(TLSCaCertFlagName),
		TLSCert:   ctx.GlobalString(TLSCertFlagName),
		TLSKey:    ctx.GlobalString(TLSKeyFlagName),
	}
}
