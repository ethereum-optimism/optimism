package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

var (
	ValidL2RPCs  = []string{"http;//localhost:8545"}
	ValidDatadir = "./supervisor_test_datadir"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown level: foo", addRequiredArgs("--log.level=foo"))
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		lvl := lvl
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := dryRunWithArgs(addRequiredArgs("--log.level", lvl))
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs(t, addRequiredArgs())
	depSet := &depset.JsonDependencySetLoader{Path: "test"}
	defaultCfgTempl := config.NewConfig(ValidL2RPCs, depSet, ValidDatadir)
	defaultCfg := *defaultCfgTempl
	defaultCfg.Version = Version
	require.Equal(t, defaultCfg, *cfg)
}

func TestL2RPCs(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l2-rpcs is required", addRequiredArgsExcept("--l2-rpcs"))
	})

	t.Run("Valid", func(t *testing.T) {
		url1 := "http://example.com:1234"
		url2 := "http://foobar.com:1234"
		cfg := configForArgs(t, addRequiredArgsExcept("--l2-rpcs", "--l2-rpcs="+url1+","+url2))
		require.Equal(t, []string{url1, url2}, cfg.L2RPCs)
	})
}

func TestDatadir(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag datadir is required", addRequiredArgsExcept("--datadir"))
	})

	t.Run("Valid", func(t *testing.T) {
		dir := "foo"
		cfg := configForArgs(t, addRequiredArgsExcept("--datadir", "--datadir", dir))
		require.Equal(t, dir, cfg.Datadir)
	})
}

func TestMockRun(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--mock-run"))
		require.Equal(t, true, cfg.MockRun)
	})
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
	fullArgs := append([]string{"op-supervisor"}, cliArgs...)
	testErr := errors.New("dry-run")
	err := run(context.Background(), fullArgs, func(ctx context.Context, config *config.Config, log log.Logger) (cliapp.Lifecycle, error) {
		logger = log
		cfg = config
		return nil, testErr
	})
	if errors.Is(err, testErr) { // expected error
		err = nil
	}
	return logger, cfg, err
}

func addRequiredArgs(args ...string) []string {
	req := requiredArgs()
	combined := toArgList(req)
	return append(combined, args...)
}

func addRequiredArgsExcept(name string, optionalArgs ...string) []string {
	req := requiredArgs()
	delete(req, name)
	return append(toArgList(req), optionalArgs...)
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
		"--l2-rpcs":        ValidL2RPCs[0],
		"--dependency-set": "test",
		"--datadir":        ValidDatadir,
	}
	return args
}
