package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-sync-tester/config"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown level: foo", addRequiredArgs("--log.level=foo"))
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := dryRunWithArgs(addRequiredArgs("--log.level", lvl))
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestEmptyRun(t *testing.T) {
	cfg := configForArgs(t, addRequiredArgs("--config=testdata/empty.yaml"))
	require.NoError(t, cfg.Check())
}

func verifyArgsInvalid(t *testing.T, messageContains string, cliArgs []string) {
	_, _, err := dryRunWithArgs(cliArgs)
	require.ErrorContains(t, err, messageContains)
}

func configForArgs(t *testing.T, cliArgs []string) *config.Config {
	_, cfg, err := dryRunWithArgs(cliArgs)
	require.NoError(t, err)
	return cfg
}

func dryRunWithArgs(cliArgs []string) (log.Logger, *config.Config, error) {
	cfg := new(config.Config)
	var logger log.Logger
	fullArgs := append([]string{"op-sync-tester"}, cliArgs...)
	testErr := errors.New("dry-run")
	err := run(context.Background(), io.Discard, io.Discard, fullArgs, func(ctx context.Context, config *config.Config, log log.Logger) (cliapp.Lifecycle, error) {
		logger = log
		cfg = config
		return nil, testErr
	})
	if errors.Is(err, testErr) {
		err = nil
	}
	return logger, cfg, err
}

func addRequiredArgs(args ...string) []string {
	req := requiredArgs()
	combined := toArgList(req)
	return append(combined, args...)
}

func toArgList(req map[string]string) []string {
	var combined []string
	for name, value := range req {
		combined = append(combined, fmt.Sprintf("%s=%s", name, value))
	}
	return combined
}

func requiredArgs() map[string]string {
	args := map[string]string{
		"--config": "testdata/empty.yaml",
	}
	return args
}
