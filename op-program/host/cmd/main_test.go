package main

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	// Use HexToHash(...).Hex() to ensure the strings are the correct length for a hash
	l1HeadValue        = common.HexToHash("0x111111").Hex()
	l2HeadValue        = common.HexToHash("0x222222").Hex()
	l2ClaimValue       = common.HexToHash("0x333333").Hex()
	l2ClaimBlockNumber = uint64(1203)
	// Note: This is actually the L1 goerli genesis config. Just using it as an arbitrary, valid genesis config
	l2Genesis       = core.DefaultGoerliGenesisBlock()
	l2GenesisConfig = l2Genesis.Config
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
	defaultCfg := config.NewConfig(
		&chaincfg.Goerli,
		config.OPGoerliChainConfig,
		common.HexToHash(l1HeadValue),
		common.HexToHash(l2HeadValue),
		common.HexToHash(l2ClaimValue),
		l2ClaimBlockNumber)
	require.Equal(t, defaultCfg, cfg)
}

func TestNetwork(t *testing.T) {
	t.Run("Unknown", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid network bar", replaceRequiredArg("--network", "bar"))
	})

	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag rollup.config or network is required", addRequiredArgsExcept("--network"))
	})

	t.Run("DisallowNetworkAndRollupConfig", func(t *testing.T) {
		verifyArgsInvalid(t, "cannot specify both rollup.config and network", addRequiredArgs("--rollup.config=foo"))
	})

	t.Run("RollupConfig", func(t *testing.T) {
		configFile := writeValidRollupConfig(t)
		genesisFile := writeValidGenesis(t)

		cfg := configForArgs(t, addRequiredArgsExcept("--network", "--rollup.config", configFile, "--l2.genesis", genesisFile))
		require.Equal(t, chaincfg.Goerli, *cfg.Rollup)
	})

	for name, cfg := range chaincfg.NetworksByName {
		name := name
		expected := cfg
		t.Run("Network_"+name, func(t *testing.T) {
			// TODO(CLI-3936) Re-enable test for other networks once bedrock migration is complete
			if name != "goerli" {
				t.Skipf("Not requiring chain config for network %s", name)
				return
			}
			args := replaceRequiredArg("--network", name)
			cfg := configForArgs(t, args)
			require.Equal(t, expected, *cfg.Rollup)
		})
	}
}

func TestDataDir(t *testing.T) {
	expected := "/tmp/mainTestDataDir"
	cfg := configForArgs(t, addRequiredArgs("--datadir", expected))
	require.Equal(t, expected, cfg.DataDir)
}

func TestL2(t *testing.T) {
	expected := "https://example.com:8545"
	cfg := configForArgs(t, addRequiredArgs("--l2", expected))
	require.Equal(t, expected, cfg.L2URL)
}

func TestL2Genesis(t *testing.T) {
	t.Run("RequiredWithCustomNetwork", func(t *testing.T) {
		rollupCfgFile := writeValidRollupConfig(t)
		verifyArgsInvalid(t, "flag l2.genesis is required", addRequiredArgsExcept("--network", "--rollup.config", rollupCfgFile))
	})

	t.Run("Valid", func(t *testing.T) {
		rollupCfgFile := writeValidRollupConfig(t)
		genesisFile := writeValidGenesis(t)
		cfg := configForArgs(t, addRequiredArgsExcept("--network", "--rollup.config", rollupCfgFile, "--l2.genesis", genesisFile))
		require.Equal(t, l2GenesisConfig, cfg.L2ChainConfig)
	})

	t.Run("NotRequiredForGoerli", func(t *testing.T) {
		cfg := configForArgs(t, replaceRequiredArg("--network", "goerli"))
		require.Equal(t, config.OPGoerliChainConfig, cfg.L2ChainConfig)
	})
}

func TestL2Head(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l2.head is required", addRequiredArgsExcept("--l2.head"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, replaceRequiredArg("--l2.head", l2HeadValue))
		require.Equal(t, common.HexToHash(l2HeadValue), cfg.L2Head)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, config.ErrInvalidL2Head.Error(), replaceRequiredArg("--l2.head", "something"))
	})
}

func TestL1Head(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l1.head is required", addRequiredArgsExcept("--l1.head"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, replaceRequiredArg("--l1.head", l1HeadValue))
		require.Equal(t, common.HexToHash(l1HeadValue), cfg.L1Head)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, config.ErrInvalidL1Head.Error(), replaceRequiredArg("--l1.head", "something"))
	})
}

func TestL1(t *testing.T) {
	expected := "https://example.com:8545"
	cfg := configForArgs(t, addRequiredArgs("--l1", expected))
	require.Equal(t, expected, cfg.L1URL)
}

func TestL1TrustRPC(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs())
		require.False(t, cfg.L1TrustRPC)
	})
	t.Run("Enabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--l1.trustrpc"))
		require.True(t, cfg.L1TrustRPC)
	})
	t.Run("EnabledWithArg", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--l1.trustrpc=true"))
		require.True(t, cfg.L1TrustRPC)
	})
	t.Run("Disabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--l1.trustrpc=false"))
		require.False(t, cfg.L1TrustRPC)
	})
}

func TestL1RPCKind(t *testing.T) {
	t.Run("DefaultBasic", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs())
		require.Equal(t, sources.RPCKindBasic, cfg.L1RPCKind)
	})
	for _, kind := range sources.RPCProviderKinds {
		t.Run(kind.String(), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgs("--l1.rpckind", kind.String()))
			require.Equal(t, kind, cfg.L1RPCKind)
		})
	}
	t.Run("RequireLowercase", func(t *testing.T) {
		verifyArgsInvalid(t, "rpc kind", addRequiredArgs("--l1.rpckind", "AlChemY"))
	})
	t.Run("UnknownKind", func(t *testing.T) {
		verifyArgsInvalid(t, "\"foo\"", addRequiredArgs("--l1.rpckind", "foo"))
	})
}

func TestL2Claim(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l2.claim is required", addRequiredArgsExcept("--l2.claim"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, replaceRequiredArg("--l2.claim", l2ClaimValue))
		require.EqualValues(t, common.HexToHash(l2ClaimValue), cfg.L2Claim)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, config.ErrInvalidL2Claim.Error(), replaceRequiredArg("--l2.claim", "something"))
	})
}

func TestL2BlockNumber(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l2.blocknumber is required", addRequiredArgsExcept("--l2.blocknumber"))
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, replaceRequiredArg("--l2.blocknumber", strconv.FormatUint(l2ClaimBlockNumber, 10)))
		require.EqualValues(t, l2ClaimBlockNumber, cfg.L2ClaimBlockNumber)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid value \"something\" for flag -l2.blocknumber", replaceRequiredArg("--l2.blocknumber", "something"))
	})
}

func TestExec(t *testing.T) {
	t.Run("DefaultEmpty", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs())
		require.Equal(t, "", cfg.ExecCmd)
	})
	t.Run("Set", func(t *testing.T) {
		cmd := "/bin/echo"
		cfg := configForArgs(t, addRequiredArgs("--exec", cmd))
		require.Equal(t, cmd, cfg.ExecCmd)
	})
}

func TestServerMode(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs())
		require.False(t, cfg.ServerMode)
	})
	t.Run("Enabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--server"))
		require.True(t, cfg.ServerMode)
	})
	t.Run("EnabledWithArg", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--server=true"))
		require.True(t, cfg.ServerMode)
	})
	t.Run("DisabledWithArg", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs("--server=false"))
		require.False(t, cfg.ServerMode)
	})
	t.Run("InvalidArg", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid boolean value \"foo\" for -server", addRequiredArgs("--server=foo"))
	})
}

func verifyArgsInvalid(t *testing.T, messageContains string, cliArgs []string) {
	_, _, err := runWithArgs(cliArgs)
	require.ErrorContains(t, err, messageContains)
}

func configForArgs(t *testing.T, cliArgs []string) *config.Config {
	_, cfg, err := runWithArgs(cliArgs)
	require.NoError(t, err)
	return cfg
}

func runWithArgs(cliArgs []string) (log.Logger, *config.Config, error) {
	cfg := new(config.Config)
	var logger log.Logger
	fullArgs := append([]string{"op-program"}, cliArgs...)
	err := run(fullArgs, func(log log.Logger, config *config.Config) error {
		logger = log
		cfg = config
		return nil
	})
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

func replaceRequiredArg(name string, value string) []string {
	req := requiredArgs()
	req[name] = value
	return toArgList(req)
}

// requiredArgs returns map of argument names to values which are the minimal arguments required
// to create a valid Config
func requiredArgs() map[string]string {
	return map[string]string{
		"--network":        "goerli",
		"--l1.head":        l1HeadValue,
		"--l2.head":        l2HeadValue,
		"--l2.claim":       l2ClaimValue,
		"--l2.blocknumber": strconv.FormatUint(l2ClaimBlockNumber, 10),
	}
}

func writeValidGenesis(t *testing.T) string {
	dir := t.TempDir()
	j, err := json.Marshal(l2Genesis)
	require.NoError(t, err)
	genesisFile := dir + "/genesis.json"
	require.NoError(t, os.WriteFile(genesisFile, j, 0666))
	return genesisFile
}

func writeValidRollupConfig(t *testing.T) string {
	dir := t.TempDir()
	j, err := json.Marshal(chaincfg.Goerli)
	require.NoError(t, err)
	cfgFile := dir + "/rollup.json"
	require.NoError(t, os.WriteFile(cfgFile, j, 0666))
	return cfgFile
}

func toArgList(req map[string]string) []string {
	var combined []string
	for name, value := range req {
		combined = append(combined, name)
		combined = append(combined, value)
	}
	return combined
}
