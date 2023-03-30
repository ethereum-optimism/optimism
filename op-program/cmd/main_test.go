package main

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-program/config"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		_, _, err := runWithArgs("--log.level=foo")
		require.ErrorContains(t, err, "unknown level: foo")
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		lvl := lvl
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := runWithArgs("--log.level", lvl)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs(t)
	require.Equal(t, config.DefaultConfig(), cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	err := config.DefaultConfig().Check()
	require.NoError(t, err)
}

func configForArgs(t *testing.T, cliArgs ...string) config.Config {
	_, cfg, err := runWithArgs(cliArgs...)
	require.NoError(t, err)
	return cfg
}

func runWithArgs(cliArgs ...string) (log.Logger, config.Config, error) {
	var cfg config.Config
	var logger log.Logger
	err := run(args(cliArgs...), func(log log.Logger, config config.Config) error {
		logger = log
		cfg = config
		return nil
	})
	return logger, cfg, err
}

func args(args ...string) []string {
	return append([]string{"op-program"}, args...)
}
