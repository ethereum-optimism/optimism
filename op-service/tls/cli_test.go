package tls

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs()
	defaultCfg := NewCLIConfig()
	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	err := NewCLIConfig().Check()
	require.NoError(t, err)
}

func TestInvalidConfig(t *testing.T) {
	tests := []struct {
		name         string
		configChange func(config *CLIConfig)
	}{
		{"MissingCaCert", func(config *CLIConfig) {
			config.TLSCaCert = ""
		}},
		{"MissingCert", func(config *CLIConfig) {
			config.TLSCert = ""
		}},
		{"MissingKey", func(config *CLIConfig) {
			config.TLSKey = ""
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := NewCLIConfig()
			test.configChange(&cfg)
			err := cfg.Check()
			require.ErrorContains(t, err, "all tls flags must be set if at least one is set")
		})
	}
}

func configForArgs(args ...string) CLIConfig {
	app := cli.NewApp()
	app.Flags = CLIFlagsWithFlagPrefix("TEST_", "test")
	app.Name = "test"
	var config CLIConfig
	app.Action = func(ctx *cli.Context) error {
		config = ReadCLIConfigWithPrefix(ctx, "test")
		return nil
	}
	_ = app.Run(args)
	return config
}

func TestDefaultSignerCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := signerConfigForArgs()
	defaultCfg := NewSignerCLIConfig()
	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultSignerConfigIsValid(t *testing.T) {
	err := NewSignerCLIConfig().Check()
	require.NoError(t, err)
}

func TestInvalidSignerConfig(t *testing.T) {
	tests := []struct {
		name         string
		expected     string
		configChange func(config *SignerCLIConfig)
	}{
		{
			name:     "MissingEndpoint",
			expected: "signer endpoint and address must both be set or not set",
			configChange: func(config *SignerCLIConfig) {
				config.Address = "0x1234"
			},
		},
		{
			name:     "MissingAddress",
			expected: "signer endpoint and address must both be set or not set",
			configChange: func(config *SignerCLIConfig) {
				config.Endpoint = "http://localhost"
			},
		},
		{
			name:     "InvalidTLSConfig",
			expected: "all tls flags must be set if at least one is set",
			configChange: func(config *SignerCLIConfig) {
				config.TLSConfig.TLSKey = ""
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := NewSignerCLIConfig()
			test.configChange(&cfg)
			err := cfg.Check()
			require.ErrorContains(t, err, test.expected)
		})
	}
}

func signerConfigForArgs(args ...string) SignerCLIConfig {
	app := cli.NewApp()
	app.Flags = CLIFlags("TEST_")
	app.Name = "test"
	var config SignerCLIConfig
	app.Action = func(ctx *cli.Context) error {
		config = ReadSignerCLIConfig(ctx)
		return nil
	}
	_ = app.Run(args)
	return config
}
