// This file contains CLI and env TLS configurations that can be used by clients or servers
package tls

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	TLSCaCertFlagName = "tls.ca"
	TLSCertFlagName   = "tls.cert"
	TLSKeyFlagName    = "tls.key"
)

// CLIFlags returns flags with env var envPrefix
// This should be used for server TLS configs, or when client and server tls configs are the same
func CLIFlags(envPrefix string) []cli.Flag {
	return CLIFlagsWithFlagPrefix(envPrefix, "")
}

// CLIFlagsWithFlagPrefix returns flags with env var and cli flag prefixes
// Should be used for client TLS configs when different from server on the same process
func CLIFlagsWithFlagPrefix(envPrefix string, flagPrefix string) []cli.Flag {
	prefixFunc := func(flagName string) string {
		return strings.Trim(fmt.Sprintf("%s.%s", flagPrefix, flagName), ".")
	}
	return []cli.Flag{
		cli.StringFlag{
			Name:   prefixFunc(TLSCaCertFlagName),
			Usage:  "tls ca cert path",
			Value:  "tls/ca.crt",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TLS_CA"),
		},
		cli.StringFlag{
			Name:   prefixFunc(TLSCertFlagName),
			Usage:  "tls cert path",
			Value:  "tls/tls.crt",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "TLS_CERT"),
		},
		cli.StringFlag{
			Name:   prefixFunc(TLSKeyFlagName),
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

// ReadCLIConfig reads tls cli configs
// This should be used for server TLS configs, or when client and server tls configs are the same
func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		TLSCaCert: ctx.GlobalString(TLSCaCertFlagName),
		TLSCert:   ctx.GlobalString(TLSCertFlagName),
		TLSKey:    ctx.GlobalString(TLSKeyFlagName),
	}
}

// ReadCLIConfigWithPrefix reads tls cli configs with flag prefix
// Should be used for client TLS configs when different from server on the same process
func ReadCLIConfigWithPrefix(ctx *cli.Context, flagPrefix string) CLIConfig {
	prefixFunc := func(flagName string) string {
		return strings.Trim(fmt.Sprintf("%s.%s", flagPrefix, flagName), ".")
	}
	return CLIConfig{
		TLSCaCert: ctx.GlobalString(prefixFunc(TLSCaCertFlagName)),
		TLSCert:   ctx.GlobalString(prefixFunc(TLSCertFlagName)),
		TLSKey:    ctx.GlobalString(prefixFunc(TLSKeyFlagName)),
	}
}
