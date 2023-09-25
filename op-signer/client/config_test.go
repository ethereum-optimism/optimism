package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	optls "github.com/ethereum-optimism/optimism/op-service/tls"
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
		expected     string
		configChange func(config *optls.SignerCLIConfig)
	}{
		{
			name:     "MissingEndpoint",
			expected: "signer endpoint and address must both be set or not set",
			configChange: func(config *optls.SignerCLIConfig) {
				config.Address = "0x1234"
			},
		},
		{
			name:     "MissingAddress",
			expected: "signer endpoint and address must both be set or not set",
			configChange: func(config *optls.SignerCLIConfig) {
				config.Endpoint = "http://localhost"
			},
		},
		{
			name:     "InvalidTLSConfig",
			expected: "all tls flags must be set if at least one is set",
			configChange: func(config *optls.SignerCLIConfig) {
				config.TLSConfig.TLSKey = ""
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := NewCLIConfig()
			test.configChange(&cfg)
			err := cfg.Check()
			require.ErrorContains(t, err, test.expected)
		})
	}
}

func configForArgs(args ...string) optls.SignerCLIConfig {
	app := cli.NewApp()
	app.Flags = CLIFlags("TEST_")
	app.Name = "test"
	var config optls.SignerCLIConfig
	app.Action = func(ctx *cli.Context) error {
		config = ReadCLIConfig(ctx)
		return nil
	}
	_ = app.Run(args)
	return config
}
