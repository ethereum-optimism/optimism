package main

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	l1EthRpc                = "http://example.com:8545"
	gameAddressValue        = "0xaa00000000000000000000000000000000000000"
	alphabetTrace           = "abcdefghijz"
	agreeWithProposedOutput = "true"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown level: foo", addRequiredArgs("--log.level=foo"))
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		lvl := lvl
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := runWithArgs(addRequiredArgs("--log.level", lvl))
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs(t, addRequiredArgs())
	defaultCfg := config.NewConfig(l1EthRpc, common.HexToAddress(gameAddressValue), alphabetTrace, true)
	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := config.NewConfig(l1EthRpc, common.HexToAddress(gameAddressValue), alphabetTrace, true)
	require.NoError(t, cfg.Check())
}

func TestL1ETHRPCAddress(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l1-eth-rpc is required", addRequiredArgsExcept("--l1-eth-rpc"))
	})

	t.Run("Valid", func(t *testing.T) {
		url := "http://example.com:8888"
		cfg := configForArgs(t, addRequiredArgsExcept("--l1-eth-rpc", "--l1-eth-rpc="+url))
		require.Equal(t, url, cfg.L1EthRpc)
		require.Equal(t, url, cfg.TxMgrConfig.L1RPCURL)
	})
}

func TestAlphabetTrace(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag alphabet is required", addRequiredArgsExcept("--alphabet"))
	})

	t.Run("Valid", func(t *testing.T) {
		value := "abcde"
		cfg := configForArgs(t, addRequiredArgsExcept("--alphabet", "--alphabet="+value))
		require.Equal(t, value, cfg.AlphabetTrace)
	})
}

func TestGameAddress(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag game-address is required", addRequiredArgsExcept("--game-address"))
	})

	t.Run("Valid", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept("--game-address", "--game-address="+addr.Hex()))
		require.Equal(t, addr, cfg.GameAddress)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid address: foo", addRequiredArgsExcept("--game-address", "--game-address=foo"))
	})
}

func TestTxManagerFlagsSupported(t *testing.T) {
	// Not a comprehensive list of flags, just enough to sanity check the txmgr.CLIFlags were defined
	cfg := configForArgs(t, addRequiredArgs("--"+txmgr.NumConfirmationsFlagName, "7"))
	require.Equal(t, uint64(7), cfg.TxMgrConfig.NumConfirmations)
}

func TestAgreeWithProposedOutput(t *testing.T) {
	t.Run("MustBeProvided", func(t *testing.T) {
		verifyArgsInvalid(t, "flag agree-with-proposed-output is required", addRequiredArgsExcept("--agree-with-proposed-output"))
	})
	t.Run("Enabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--agree-with-proposed-output"))
		require.True(t, cfg.AgreeWithProposedOutput)
	})
	t.Run("EnabledWithArg", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--agree-with-proposed-output=true"))
		require.True(t, cfg.AgreeWithProposedOutput)
	})
	t.Run("Disabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--agree-with-proposed-output=false"))
		require.False(t, cfg.AgreeWithProposedOutput)
	})
}

func verifyArgsInvalid(t *testing.T, messageContains string, cliArgs []string) {
	_, _, err := runWithArgs(cliArgs)
	require.ErrorContains(t, err, messageContains)
}

func configForArgs(t *testing.T, cliArgs []string) config.Config {
	_, cfg, err := runWithArgs(cliArgs)
	require.NoError(t, err)
	return cfg
}

func runWithArgs(cliArgs []string) (log.Logger, config.Config, error) {
	cfg := new(config.Config)
	var logger log.Logger
	fullArgs := append([]string{"op-challenger"}, cliArgs...)
	err := run(fullArgs, func(log log.Logger, config *config.Config) error {
		logger = log
		cfg = config
		return nil
	})
	return logger, *cfg, err
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

func requiredArgs() map[string]string {
	return map[string]string{
		"--agree-with-proposed-output": agreeWithProposedOutput,
		"--l1-eth-rpc":                 l1EthRpc,
		"--game-address":               gameAddressValue,
		"--alphabet":                   alphabetTrace,
	}
}

func toArgList(req map[string]string) []string {
	var combined []string
	for name, value := range req {
		combined = append(combined, fmt.Sprintf("%s=%s", name, value))
	}
	return combined
}
