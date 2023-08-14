package kms

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestInvalidConfig(t *testing.T) {
	tests := []struct {
		name         string
		configChange func(config *CLIConfig)
		err          string
	}{
		{"MissingKmsEndpoint", func(config *CLIConfig) {
			config.KmsKeyID = "test"
			config.KmsEndpoint = ""
		},
			"KMS Endpoint must be provided",
		},
		{"MissingKmsRegion", func(config *CLIConfig) {
			config.KmsKeyID = "test"
			config.KmsEndpoint = "test"
			config.KmsRegion = ""
		},
			"KMS Region must be provided",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := configForArgs()
			test.configChange(&cfg)
			err := cfg.Check()
			require.ErrorContains(t, err, test.err)
		})
	}
}

func configForArgs(args ...string) CLIConfig {
	app := cli.NewApp()
	app.Flags = CLIFlags("TEST_")
	app.Name = "test"
	var config CLIConfig
	app.Action = func(ctx *cli.Context) error {
		config = ReadCLIConfig(ctx)
		return nil
	}
	_ = app.Run(args)
	return config
}
