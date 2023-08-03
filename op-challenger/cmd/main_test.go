package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	l1EthRpc                   = "http://example.com:8545"
	gameAddressValue           = "0xaa00000000000000000000000000000000000000"
	preimageOracleAddressValue = "0xbb00000000000000000000000000000000000000"
	cannonBin                  = "./bin/cannon"
	cannonServer               = "./bin/op-program"
	cannonPreState             = "./pre.json"
	cannonDatadir              = "./test_data"
	cannonL2                   = "http://example.com:9545"
	alphabetTrace              = "abcdefghijz"
	agreeWithProposedOutput    = "true"
	gameDepth                  = "4"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown level: foo", addRequiredArgs(config.TraceTypeAlphabet, "--log.level=foo"))
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		lvl := lvl
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := runWithArgs(addRequiredArgs(config.TraceTypeAlphabet, "--log.level", lvl))
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs(t, addRequiredArgs(config.TraceTypeAlphabet))
	defaultCfg := config.NewConfig(l1EthRpc, common.HexToAddress(gameAddressValue), common.HexToAddress(preimageOracleAddressValue), config.TraceTypeAlphabet, true, 4)
	// Add in the extra CLI options required when using alphabet trace type
	defaultCfg.AlphabetTrace = alphabetTrace
	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := config.NewConfig(l1EthRpc, common.HexToAddress(gameAddressValue), common.HexToAddress(preimageOracleAddressValue), config.TraceTypeAlphabet, true, 4)
	// Add in options that are required based on the specific trace type
	// To avoid needing to specify unused options, these aren't included in the params for NewConfig
	cfg.AlphabetTrace = alphabetTrace
	require.NoError(t, cfg.Check())
}

func TestL1ETHRPCAddress(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l1-eth-rpc is required", addRequiredArgsExcept(config.TraceTypeAlphabet, "--l1-eth-rpc"))
	})

	t.Run("Valid", func(t *testing.T) {
		url := "http://example.com:8888"
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--l1-eth-rpc", "--l1-eth-rpc="+url))
		require.Equal(t, url, cfg.L1EthRpc)
		require.Equal(t, url, cfg.TxMgrConfig.L1RPCURL)
	})
}

func TestTraceType(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag trace-type is required", addRequiredArgsExcept("", "--trace-type"))
	})

	for _, traceType := range config.TraceTypes {
		traceType := traceType
		t.Run("Valid_"+traceType.String(), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgs(traceType))
			require.Equal(t, traceType, cfg.TraceType)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown trace type: \"foo\"", addRequiredArgsExcept(config.TraceTypeAlphabet, "--trace-type", "--trace-type=foo"))
	})
}

func TestGameAddress(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag game-address is required", addRequiredArgsExcept(config.TraceTypeAlphabet, "--game-address"))
	})

	t.Run("Valid", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--game-address", "--game-address="+addr.Hex()))
		require.Equal(t, addr, cfg.GameAddress)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid address: foo", addRequiredArgsExcept(config.TraceTypeAlphabet, "--game-address", "--game-address=foo"))
	})
}

func TestTxManagerFlagsSupported(t *testing.T) {
	// Not a comprehensive list of flags, just enough to sanity check the txmgr.CLIFlags were defined
	cfg := configForArgs(t, addRequiredArgs(config.TraceTypeAlphabet, "--"+txmgr.NumConfirmationsFlagName, "7"))
	require.Equal(t, uint64(7), cfg.TxMgrConfig.NumConfirmations)
}

func TestAgreeWithProposedOutput(t *testing.T) {
	t.Run("MustBeProvided", func(t *testing.T) {
		verifyArgsInvalid(t, "flag agree-with-proposed-output is required", addRequiredArgsExcept(config.TraceTypeAlphabet, "--agree-with-proposed-output"))
	})
	t.Run("Enabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeAlphabet, "--agree-with-proposed-output"))
		require.True(t, cfg.AgreeWithProposedOutput)
	})
	t.Run("EnabledWithArg", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeAlphabet, "--agree-with-proposed-output=true"))
		require.True(t, cfg.AgreeWithProposedOutput)
	})
	t.Run("Disabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeAlphabet, "--agree-with-proposed-output=false"))
		require.False(t, cfg.AgreeWithProposedOutput)
	})
}

func TestGameDepth(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag game-depth is required", addRequiredArgsExcept(config.TraceTypeAlphabet, "--game-depth"))
	})

	t.Run("Valid", func(t *testing.T) {
		value := "4"
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--game-depth", "--game-depth="+value))
		require.Equal(t, value, fmt.Sprint(cfg.GameDepth))
	})
}

func TestPreimageOracleAddress(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--preimage-oracle-address"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag preimage-oracle-address is required", addRequiredArgsExcept(config.TraceTypeCannon, "--preimage-oracle-address"))
	})

	t.Run("Valid", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeCannon, "--preimage-oracle-address", "--preimage-oracle-address="+addr.Hex()))
		require.Equal(t, addr, cfg.PreimageOracleAddress)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid address: foo", addRequiredArgsExcept(config.TraceTypeCannon, "--preimage-oracle-address", "--preimage-oracle-address=foo"))
	})
}

func TestCannonBin(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--cannon-bin"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag cannon-bin is required", addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-bin"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-bin", "--cannon-bin=./cannon"))
		require.Equal(t, "./cannon", cfg.CannonBin)
	})
}

func TestCannonServer(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--cannon-server"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag cannon-server is required", addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-server"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-server", "--cannon-server=./op-program"))
		require.Equal(t, "./op-program", cfg.CannonServer)
	})
}

func TestCannonAbsolutePrestate(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--cannon-prestate"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag cannon-prestate is required", addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-prestate"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-prestate", "--cannon-prestate=./pre.json"))
		require.Equal(t, "./pre.json", cfg.CannonAbsolutePreState)
	})
}

func TestCannonDataDir(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--cannon-datadir"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag cannon-datadir is required", addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-datadir"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-datadir", "--cannon-datadir=/foo/bar/cannon"))
		require.Equal(t, "/foo/bar/cannon", cfg.CannonDatadir)
	})
}

func TestCannonL2(t *testing.T) {
	t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
		configForArgs(t, addRequiredArgsExcept(config.TraceTypeAlphabet, "--cannon-l2"))
	})

	t.Run("RequiredForCannonTrace", func(t *testing.T) {
		verifyArgsInvalid(t, "flag cannon-l2 is required", addRequiredArgsExcept(config.TraceTypeCannon, "--cannon-l2"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeCannon))
		require.Equal(t, cannonL2, cfg.CannonL2)
	})
}

func TestCannonSnapshotFreq(t *testing.T) {
	t.Run("UsesDefault", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeCannon))
		require.Equal(t, config.DefaultCannonSnapshotFreq, cfg.CannonSnapshotFreq)
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(config.TraceTypeCannon, "--cannon-snapshot-freq=1234"))
		require.Equal(t, uint(1234), cfg.CannonSnapshotFreq)
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
	err := run(fullArgs, func(ctx context.Context, log log.Logger, config *config.Config) error {
		logger = log
		cfg = config
		return nil
	})
	return logger, *cfg, err
}

func addRequiredArgs(traceType config.TraceType, args ...string) []string {
	req := requiredArgs(traceType)
	combined := toArgList(req)
	return append(combined, args...)
}

func addRequiredArgsExcept(traceType config.TraceType, name string, optionalArgs ...string) []string {
	req := requiredArgs(traceType)
	delete(req, name)
	return append(toArgList(req), optionalArgs...)
}

func requiredArgs(traceType config.TraceType) map[string]string {
	args := map[string]string{
		"--game-depth":                 gameDepth,
		"--agree-with-proposed-output": agreeWithProposedOutput,
		"--l1-eth-rpc":                 l1EthRpc,
		"--game-address":               gameAddressValue,
		"--preimage-oracle-address":    preimageOracleAddressValue,
		"--trace-type":                 traceType.String(),
	}
	switch traceType {
	case config.TraceTypeAlphabet:
		args["--alphabet"] = alphabetTrace
	case config.TraceTypeCannon:
		args["--cannon-bin"] = cannonBin
		args["--cannon-server"] = cannonServer
		args["--cannon-prestate"] = cannonPreState
		args["--cannon-datadir"] = cannonDatadir
		args["--cannon-l2"] = cannonL2
	}
	return args
}

func toArgList(req map[string]string) []string {
	var combined []string
	for name, value := range req {
		combined = append(combined, fmt.Sprintf("%s=%s", name, value))
	}
	return combined
}
