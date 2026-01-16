package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	l1EthRpc                = "http://example.com:8545"
	l1Beacon                = "http://example.com:9000"
	gameFactoryAddressValue = "0xbb00000000000000000000000000000000000000"
	network                 = "op-mainnet"
	testNetwork             = "op-sepolia"
	l2EthRpc                = "http://example.com:9545"
	superRpc                = "http://example.com/super"
	cannonBin               = "./bin/cannon"
	cannonServer            = "./bin/op-program"
	cannonPreState          = "./pre.json"
	cannonKonaServer        = "./bin/kona-host"
	cannonKonaPreState      = "./cannon-kona-pre.json"
	datadir                 = "./test_data"
	rollupRpc               = "http://example.com:8555"
)

func TestLogLevel(t *testing.T) {
	t.Run("RejectInvalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown level: foo", addRequiredArgs(gameTypes.AlphabetGameType, "--log.level=foo"))
	})

	for _, lvl := range []string{"trace", "debug", "info", "error", "crit"} {
		lvl := lvl
		t.Run("AcceptValid_"+lvl, func(t *testing.T) {
			logger, _, err := dryRunWithArgs(addRequiredArgs(gameTypes.AlphabetGameType, "--log.level", lvl))
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestL2Experimental(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		url := "http://example.com:8888"
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--l2-experimental-eth-rpc="+url))
		require.Equal(t, url, cfg.Cannon.L2Experimental)
	})
}

func TestDefaultCLIOptionsMatchDefaultConfig(t *testing.T) {
	cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType))
	defaultCfg := config.NewConfig(common.HexToAddress(gameFactoryAddressValue), l1EthRpc, l1Beacon, rollupRpc, l2EthRpc, datadir, gameTypes.AlphabetGameType)
	require.Equal(t, defaultCfg, cfg)
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := config.NewConfig(common.HexToAddress(gameFactoryAddressValue), l1EthRpc, l1Beacon, rollupRpc, l2EthRpc, datadir, gameTypes.AlphabetGameType)
	require.NoError(t, cfg.Check())
}

func TestL1ETHRPCAddress(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l1-eth-rpc is required", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--l1-eth-rpc"))
	})

	t.Run("Valid", func(t *testing.T) {
		url := "http://example.com:8888"
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--l1-eth-rpc", "--l1-eth-rpc="+url))
		require.Equal(t, url, cfg.L1EthRpc)
		require.Equal(t, url, cfg.TxMgrConfig.L1RPCURL)
	})
}

func TestL1ETHRPCKind(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		const kind = sources.RPCKindAlchemy
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--l1-rpc-kind", kind.String()))
		require.Equal(t, kind, cfg.L1RPCKind)
	})
	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid value \"bob\" for flag -l1-rpc-kind", addRequiredArgs(gameTypes.AlphabetGameType, "--l1-rpc-kind", "bob"))
	})
}

func TestL1Beacon(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		verifyArgsInvalid(t, "flag l1-beacon is required", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--l1-beacon"))
	})

	t.Run("Valid", func(t *testing.T) {
		url := "http://example.com:8888"
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--l1-beacon", "--l1-beacon="+url))
		require.Equal(t, url, cfg.L1Beacon)
	})
}

func TestSuperNodeRpc(t *testing.T) {
	t.Run("RequiredForSuperCannon", func(t *testing.T) {
		verifyArgsInvalid(t, "flag supernode-rpc is required", addRequiredArgsExcept(gameTypes.SuperCannonGameType, "--supernode-rpc"))
	})
	t.Run("RequiredForSuperPermissioned", func(t *testing.T) {
		verifyArgsInvalid(t, "flag supernode-rpc is required", addRequiredArgsExcept(gameTypes.SuperPermissionedGameType, "--supernode-rpc"))
	})
	t.Run("RequiredForSuperCannonKona", func(t *testing.T) {
		verifyArgsInvalid(t, "flag supernode-rpc is required", addRequiredArgsExcept(gameTypes.SuperCannonKonaGameType, "--supernode-rpc"))
	})

	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
			continue
		}

		t.Run("NotRequiredForGameType-"+gameType.String(), func(t *testing.T) {
			configForArgs(t, addRequiredArgsExcept(gameType, "--supernode-rpc"))
		})
	}

	t.Run("Valid-SuperCannon", func(t *testing.T) {
		url := "http://localhost/super"
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.SuperCannonGameType, "--supernode-rpc", "--supernode-rpc", url))
		require.Equal(t, url, cfg.SuperRPC)
	})

	t.Run("Valid-SuperPermissioned", func(t *testing.T) {
		url := "http://localhost/super"
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.SuperPermissionedGameType, "--supernode-rpc", "--supernode-rpc", url))
		require.Equal(t, url, cfg.SuperRPC)
	})

	t.Run("Valid-SuperCannonKona", func(t *testing.T) {
		url := "http://localhost/super"
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.SuperCannonKonaGameType, "--supernode-rpc", "--supernode-rpc", url))
		require.Equal(t, url, cfg.SuperRPC)
	})
}

func TestGameTypes(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		expectedDefault := []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.CannonKonaGameType}
		cfg := configForArgs(t, addRequiredArgsForMultipleGameTypesExcept(expectedDefault, "--game-types"))
		require.Equal(t, expectedDefault, cfg.GameTypes)
	})

	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		t.Run("Valid_"+gameType.String(), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgs(gameType))
			require.Equal(t, []gameTypes.GameType{gameType}, cfg.GameTypes)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown game type: \"foo\"", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-types", "--game-types=foo"))
	})

	// Check we provide an alias for --trace-type to preserve backwards compatibility
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		t.Run("TraceTypeAlias-"+gameType.String(), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--game-types", "--trace-type", gameType.String()))
			require.Equal(t, []gameTypes.GameType{gameType}, cfg.GameTypes)
		})
	}
}

func TestMultipleGameTypes(t *testing.T) {
	t.Run("WithAllOptions", func(t *testing.T) {
		argsMap := requiredArgs(gameTypes.CannonGameType)
		// Add cannon-kona required flags
		addRequiredCannonKonaArgs(argsMap)
		args := toArgList(argsMap)
		// Add extra game types (cannon is already specified)
		args = append(args,
			"--game-types", gameTypes.AlphabetGameType.String())
		args = append(args,
			"--game-types", gameTypes.PermissionedGameType.String())
		args = append(args,
			"--game-types", gameTypes.CannonKonaGameType.String())
		cfg := configForArgs(t, args)
		require.Equal(t, []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.AlphabetGameType, gameTypes.PermissionedGameType, gameTypes.CannonKonaGameType}, cfg.GameTypes)
	})
	t.Run("WithSomeOptions", func(t *testing.T) {
		argsMap := requiredArgs(gameTypes.CannonGameType)
		args := toArgList(argsMap)
		// Add extra game types (cannon is already specified)
		args = append(args,
			"--game-types", gameTypes.AlphabetGameType.String())
		cfg := configForArgs(t, args)
		require.Equal(t, []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.AlphabetGameType}, cfg.GameTypes)
	})

	t.Run("SpecifySameOptionMultipleTimes", func(t *testing.T) {
		argsMap := requiredArgs(gameTypes.CannonGameType)
		args := toArgList(argsMap)
		// Add cannon game type again
		args = append(args, "--game-types", gameTypes.CannonGameType.String())
		// We're fine with the same option being listed multiple times, just deduplicate them.
		cfg := configForArgs(t, args)
		require.Equal(t, []gameTypes.GameType{gameTypes.CannonGameType}, cfg.GameTypes)
	})
}

func TestGameFactoryAddress(t *testing.T) {
	t.Run("RequiredWhenNetworkNotSupplied", func(t *testing.T) {
		verifyArgsInvalid(t, "flag game-factory-address or network is required", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address"))
	})

	t.Run("RequiredWhenMultipleNetworksSuppliedWithDifferentFactories", func(t *testing.T) {
		verifyArgsInvalid(t, "specified networks use different dispute game factories, flag game-factory-address required", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--network", "op-sepolia,op-mainnet"))
	})

	t.Run("Valid", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--game-factory-address="+addr.Hex()))
		require.Equal(t, addr, cfg.GameFactoryAddress)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid address: foo", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--game-factory-address=foo"))
	})

	t.Run("OverridesNetwork", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--game-factory-address", addr.Hex(), "--network", "op-sepolia"))
		require.Equal(t, addr, cfg.GameFactoryAddress)
	})
}

func TestNetwork(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		opSepoliaChainId := uint64(11155420)
		opSepolia, err := superchain.GetChain(opSepoliaChainId)
		require.NoError(t, err)
		opSepoliaCfg, err := opSepolia.Config()
		require.NoError(t, err)
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--network=op-sepolia"))
		require.EqualValues(t, *opSepoliaCfg.Addresses.DisputeGameFactoryProxy, cfg.GameFactoryAddress)
	})

	t.Run("UnknownNetwork", func(t *testing.T) {
		verifyArgsInvalid(t, "unknown chain: not-a-network", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--network=not-a-network"))
	})

	t.Run("ChainIDAllowedWhenGameFactoryAddressSupplied", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-factory-address", "--network=1234", "--game-factory-address="+addr.Hex()))
		require.Equal(t, addr, cfg.GameFactoryAddress)
		require.Equal(t, []string{"1234"}, cfg.Cannon.Networks)
	})
}

func TestGameAllowlist(t *testing.T) {
	t.Run("Optional", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-allowlist"))
		require.NoError(t, cfg.Check())
	})

	t.Run("Valid", func(t *testing.T) {
		addr := common.Address{0xbb, 0xcc, 0xdd}
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-allowlist", "--game-allowlist="+addr.Hex()))
		require.Contains(t, cfg.GameAllowlist, addr)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid address: foo", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--game-allowlist", "--game-allowlist=foo"))
	})
}

func TestTxManagerFlagsSupported(t *testing.T) {
	// Not a comprehensive list of flags, just enough to sanity check the txmgr.CLIFlags were defined
	cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--"+txmgr.NumConfirmationsFlagName, "7"))
	require.Equal(t, uint64(7), cfg.TxMgrConfig.NumConfirmations)
}

func TestMaxConcurrency(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := uint(345)
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--max-concurrency", "345"))
		require.Equal(t, expected, cfg.MaxConcurrency)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(
			t,
			"invalid value \"abc\" for flag -max-concurrency",
			addRequiredArgs(gameTypes.AlphabetGameType, "--max-concurrency", "abc"))
	})

	t.Run("Zero", func(t *testing.T) {
		verifyArgsInvalid(
			t,
			"max-concurrency must not be 0",
			addRequiredArgs(gameTypes.AlphabetGameType, "--max-concurrency", "0"))
	})
}

func TestMaxPendingTx(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := uint64(345)
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--max-pending-tx", "345"))
		require.Equal(t, expected, cfg.MaxPendingTx)
	})

	t.Run("Zero", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--max-pending-tx", "0"))
		require.Equal(t, uint64(0), cfg.MaxPendingTx)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(
			t,
			"invalid value \"abc\" for flag -max-pending-tx",
			addRequiredArgs(gameTypes.AlphabetGameType, "--max-pending-tx", "abc"))
	})
}

func TestPollInterval(t *testing.T) {
	t.Run("UsesDefault", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType))
		require.Equal(t, config.DefaultPollInterval, cfg.PollInterval)
	})

	t.Run("Valid", func(t *testing.T) {
		expected := 100 * time.Second
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--http-poll-interval", "100s"))
		require.Equal(t, expected, cfg.PollInterval)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(
			t,
			"invalid value \"abc\" for flag -http-poll-interval",
			addRequiredArgs(gameTypes.AlphabetGameType, "--http-poll-interval", "abc"))
	})
}

func TestMinUpdateInterval(t *testing.T) {
	t.Run("DefaultsToZero", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType))
		require.Equal(t, time.Duration(0), cfg.MinUpdateInterval)
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--min-update-interval", "10m"))
		require.Equal(t, 10*time.Minute, cfg.MinUpdateInterval)
	})

	t.Run("Invalid", func(t *testing.T) {
		verifyArgsInvalid(
			t,
			"invalid value \"abc\" for flag -min-update-interval",
			addRequiredArgs(gameTypes.AlphabetGameType, "--min-update-interval", "abc"))
	})
}

// validateCustomNetworkFlagsProhibitedWithNetworkFlag ensures custom network flags are not used simultaneously with the network flag.
// It validates disallowed flag combinations for a given game type and game type prefix configuration.
func validateCustomNetworkFlagsProhibitedWithNetworkFlag(t *testing.T, gameType gameTypes.GameType, gameTypeForFlagPrefix gameTypes.GameType, customNetworkFlag string) {
	expectedError := fmt.Sprintf("flag network can not be used with rollup-config/%v-rollup-config, l2-genesis/%v-l2-genesis, l1-genesis/%v-l1-genesis or %v", gameTypeForFlagPrefix, gameTypeForFlagPrefix, gameTypeForFlagPrefix, customNetworkFlag)

	// Test the custom l2 flag
	t.Run(fmt.Sprintf("TestMustNotSpecifyNetworkAndCustomL2Flag-%v", gameType), func(t *testing.T) {
		verifyArgsInvalid(
			t,
			expectedError,
			addRequiredArgs(gameType, fmt.Sprintf("--%v=true", customNetworkFlag)))
	})

	// Now test flags with trace-specific permutations
	customNetworkFlags := map[string]string{
		"RollupConfig": "rollup-config",
		"L2Genesis":    "l2-genesis",
		"L1Genesis":    "l1-genesis",
	}
	for testName, flag := range customNetworkFlags {
		for _, withTraceSpecificPrefix := range []bool{true, false} {
			var postFix string
			if withTraceSpecificPrefix {
				postFix = "-withTraceSpecificPrefix"
			}

			t.Run(fmt.Sprintf("TestMustNotSpecifyNetworkAnd%v-%v%v", testName, gameType, postFix), func(t *testing.T) {
				var prefix string
				if withTraceSpecificPrefix {
					prefix = fmt.Sprintf("%v-", gameTypeForFlagPrefix)
				}
				flagName := fmt.Sprintf("%v%v", prefix, flag)

				verifyArgsInvalid(
					t,
					expectedError,
					addRequiredArgs(gameType, fmt.Sprintf("--%v=somevalue.json", flagName)))
			})
		}
	}
}

func TestAlphabetRequiredArgs(t *testing.T) {
	t.Run(fmt.Sprintf("TestL2Rpc-%v", gameTypes.AlphabetGameType), func(t *testing.T) {
		t.Run("RequiredForAlphabetTrace", func(t *testing.T) {
			verifyArgsInvalid(t, "flag l2-eth-rpc is required", addRequiredArgsExcept(gameTypes.AlphabetGameType, "--l2-eth-rpc"))
		})

		t.Run("Valid", func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType))
			require.Equal(t, []string{l2EthRpc}, cfg.L2Rpcs)
		})
	})
}

func TestCannonCustomConfigArgs(t *testing.T) {
	for _, gameType := range []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.PermissionedGameType} {
		gameType := gameType

		t.Run(fmt.Sprintf("TestRequireEitherCannonNetworkOrRollupAndGenesis-%v", gameType), func(t *testing.T) {
			verifyArgsInvalid(
				t,
				"flag network or rollup-config/cannon-rollup-config and l2-genesis/cannon-l2-genesis is required",
				addRequiredArgsExcept(gameType, "--network"))
			verifyArgsInvalid(
				t,
				"flag network or rollup-config/cannon-rollup-config and l2-genesis/cannon-l2-genesis is required",
				addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json"))
			verifyArgsInvalid(
				t,
				"flag network or rollup-config/cannon-rollup-config and l2-genesis/cannon-l2-genesis is required",
				addRequiredArgsExcept(gameType, "--network", "--cannon-l2-genesis=gensis.json"))
		})

		validateCustomNetworkFlagsProhibitedWithNetworkFlag(t, gameType, gameTypes.CannonGameType, "cannon-l2-custom")

		t.Run(fmt.Sprintf("TestNetwork-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredWhenRollupAndGenesIsSpecified", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameType, "--network",
					"--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--network", testNetwork))
				require.Equal(t, []string{testNetwork}, cfg.Cannon.Networks)
			})
		})

		t.Run(fmt.Sprintf("TestSetCannonL2ChainId-%v", gameType), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network",
				"--cannon-rollup-config=rollup.json",
				"--cannon-l2-genesis=genesis.json",
				"--cannon-l2-custom"))
			require.True(t, cfg.Cannon.L2Custom)
		})

		t.Run(fmt.Sprintf("TestCannonRollupConfig-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-rollup-config"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json"))
				require.Equal(t, []string{"rollup.json"}, cfg.Cannon.RollupConfigPaths)
			})
		})

		t.Run(fmt.Sprintf("TestCannonL2Genesis-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-l2-genesis"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json"))
				require.Equal(t, []string{"genesis.json"}, cfg.Cannon.L2GenesisPaths)
			})
		})
	}
}

func TestSuperCannonCustomConfigArgs(t *testing.T) {
	for _, gameType := range []gameTypes.GameType{gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType} {
		gameType := gameType

		t.Run(fmt.Sprintf("TestRequireEitherCannonNetworkOrRollupAndGenesisAndDepset-%v", gameType), func(t *testing.T) {
			expectedErrorMessage := "flag network or rollup-config/cannon-rollup-config, l2-genesis/cannon-l2-genesis and depset-config/cannon-depset-config is required"
			// Missing all
			verifyArgsInvalid(
				t,
				expectedErrorMessage,
				addRequiredArgsExcept(gameType, "--network"))
			// Missing l2-genesis
			verifyArgsInvalid(
				t,
				expectedErrorMessage,
				addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-depset-config=depset.json"))
			// Missing rollup-config
			verifyArgsInvalid(
				t,
				expectedErrorMessage,
				addRequiredArgsExcept(gameType, "--network", "--cannon-l2-genesis=gensis.json", "--cannon-depset-config=depset.json"))
			// Missing depset-config
			verifyArgsInvalid(
				t,
				expectedErrorMessage,
				addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=gensis.json"))
		})

		validateCustomNetworkFlagsProhibitedWithNetworkFlag(t, gameType, gameTypes.CannonGameType, "cannon-l2-custom")

		t.Run(fmt.Sprintf("TestNetwork-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredWhenRollupGenesisAndDepsetIsSpecified", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameType, "--network",
					"--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json", "--cannon-depset-config=depset.json"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--network", testNetwork))
				require.Equal(t, []string{testNetwork}, cfg.Cannon.Networks)
			})
		})

		t.Run(fmt.Sprintf("TestSetCannonL2ChainId-%v", gameType), func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network",
				"--cannon-rollup-config=rollup.json",
				"--cannon-l2-genesis=genesis.json",
				"--cannon-depset-config=depset.json",
				"--cannon-l2-custom"))
			require.True(t, cfg.Cannon.L2Custom)
		})

		t.Run(fmt.Sprintf("TestCannonRollupConfig-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-rollup-config"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network",
					"--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json", "--cannon-depset-config=depset.json"))
				require.Equal(t, []string{"rollup.json"}, cfg.Cannon.RollupConfigPaths)
			})
		})

		t.Run(fmt.Sprintf("TestCannonL2Genesis-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-l2-genesis"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json", "--cannon-depset-config=depset.json"))
				require.Equal(t, []string{"genesis.json"}, cfg.Cannon.L2GenesisPaths)
			})
		})

		t.Run(fmt.Sprintf("TestCannonDepsetConfig-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-depset-config"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-rollup-config=rollup.json", "--cannon-l2-genesis=genesis.json", "--cannon-depset-config=depset.json"))
				require.Equal(t, "depset.json", cfg.Cannon.DepsetConfigPath)
			})
		})
	}
}

func TestSuperCannonKonaCustomConfigArgs(t *testing.T) {
	gameType := gameTypes.SuperCannonKonaGameType

	t.Run(fmt.Sprintf("TestRequireEitherCannonKonaNetworkOrRollupAndGenesisAndDepset-%v", gameType), func(t *testing.T) {
		expectedErrorMessage := "flag network or rollup-config/cannon-kona-rollup-config, l2-genesis/cannon-kona-l2-genesis and depset-config/cannon-kona-depset-config is required"
		// Missing all
		verifyArgsInvalid(
			t,
			expectedErrorMessage,
			addRequiredArgsExcept(gameType, "--network"))
		// Missing l2-genesis
		verifyArgsInvalid(
			t,
			expectedErrorMessage,
			addRequiredArgsExcept(gameType, "--network", "--cannon-kona-rollup-config=rollup.json", "--cannon-kona-depset-config=depset.json"))
		// Missing rollup-config
		verifyArgsInvalid(
			t,
			expectedErrorMessage,
			addRequiredArgsExcept(gameType, "--network", "--cannon-kona-l2-genesis=gensis.json", "--cannon-kona-depset-config=depset.json"))
		// Missing depset-config
		verifyArgsInvalid(
			t,
			expectedErrorMessage,
			addRequiredArgsExcept(gameType, "--network", "--cannon-kona-rollup-config=rollup.json", "--cannon-kona-l2-genesis=gensis.json"))
	})

	validateCustomNetworkFlagsProhibitedWithNetworkFlag(t, gameType, gameTypes.CannonKonaGameType, "cannon-kona-l2-custom")

	t.Run(fmt.Sprintf("TestNetwork-%v", gameType), func(t *testing.T) {
		t.Run("NotRequiredWhenRollupGenesisAndDepsetIsSpecified", func(t *testing.T) {
			configForArgs(t, addRequiredArgsExcept(gameType, "--network",
				"--cannon-kona-rollup-config=rollup.json", "--cannon-kona-l2-genesis=genesis.json", "--cannon-kona-depset-config=depset.json"))
		})

		t.Run("Valid", func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--network", testNetwork))
			require.Equal(t, []string{testNetwork}, cfg.CannonKona.Networks)
		})
	})

	t.Run(fmt.Sprintf("TestSetCannonKonaL2ChainId-%v", gameType), func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network",
			"--cannon-kona-rollup-config=rollup.json",
			"--cannon-kona-l2-genesis=genesis.json",
			"--cannon-kona-depset-config=depset.json",
			"--cannon-kona-l2-custom"))
		require.True(t, cfg.CannonKona.L2Custom)
	})

	t.Run(fmt.Sprintf("TestCannonKonaRollupConfig-%v", gameType), func(t *testing.T) {
		t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
			configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-kona-rollup-config"))
		})

		t.Run("Valid", func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network",
				"--cannon-kona-rollup-config=rollup.json", "--cannon-kona-l2-genesis=genesis.json", "--cannon-kona-depset-config=depset.json"))
			require.Equal(t, []string{"rollup.json"}, cfg.CannonKona.RollupConfigPaths)
		})
	})

	t.Run(fmt.Sprintf("TestCannonKonaL2Genesis-%v", gameType), func(t *testing.T) {
		t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
			configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-kona-l2-genesis"))
		})

		t.Run("Valid", func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-kona-rollup-config=rollup.json", "--cannon-kona-l2-genesis=genesis.json", "--cannon-kona-depset-config=depset.json"))
			require.Equal(t, []string{"genesis.json"}, cfg.CannonKona.L2GenesisPaths)
		})
	})

	t.Run(fmt.Sprintf("TestCannonKonaDepsetConfig-%v", gameType), func(t *testing.T) {
		t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
			configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-kona-depset-config"))
		})

		t.Run("Valid", func(t *testing.T) {
			cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--cannon-kona-rollup-config=rollup.json", "--cannon-kona-l2-genesis=genesis.json", "--cannon-kona-depset-config=depset.json"))
			require.Equal(t, "depset.json", cfg.CannonKona.DepsetConfigPath)
		})
	})
}

func TestCannonRequiredArgs(t *testing.T) {
	for _, gameType := range []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType} {
		gameType := gameType
		t.Run(fmt.Sprintf("TestCannonBin-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-bin"))
			})

			t.Run("Required", func(t *testing.T) {
				verifyArgsInvalid(t, "flag cannon-bin is required", addRequiredArgsExcept(gameType, "--cannon-bin"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--cannon-bin", "--cannon-bin=./cannon"))
				require.Equal(t, "./cannon", cfg.Cannon.VmBin)
			})
		})

		t.Run(fmt.Sprintf("TestCannonServer-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-server"))
			})

			t.Run("Required", func(t *testing.T) {
				verifyArgsInvalid(t, "flag cannon-server is required", addRequiredArgsExcept(gameType, "--cannon-server"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--cannon-server", "--cannon-server=./op-program"))
				require.Equal(t, "./op-program", cfg.Cannon.Server)
			})
		})

		t.Run(fmt.Sprintf("TestCannonAbsolutePrestate-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-prestate"))
			})

			t.Run("Required", func(t *testing.T) {
				verifyArgsInvalid(t, "flag prestates-url/cannon-prestates-url or cannon-prestate is required", addRequiredArgsExcept(gameType, "--cannon-prestate"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--cannon-prestate", "--cannon-prestate=./pre.json"))
				require.Equal(t, "./pre.json", cfg.CannonAbsolutePreState)
			})
		})

		t.Run(fmt.Sprintf("TestCannonPrestatesBaseURL-%v", gameType), func(t *testing.T) {
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--cannon-prestates-url"))
			})

			t.Run("Required", func(t *testing.T) {
				verifyArgsInvalid(t, "flag prestates-url/cannon-prestates-url or cannon-prestate is required", addRequiredArgsExcept(gameType, "--cannon-prestate"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--cannon-prestates-url", "--cannon-prestates-url=http://localhost/foo"))
				require.Equal(t, "http://localhost/foo", cfg.CannonAbsolutePreStateBaseURL.String())
			})
		})

		t.Run(fmt.Sprintf("TestPrestateBaseURL-%v", gameType), func(t *testing.T) {
			allPrestateOptions := []string{"--prestates-url", "--cannon-prestates-url", "--cannon-prestate"}
			t.Run("NotRequiredForAlphabetTrace", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExceptArr(gameTypes.AlphabetGameType, allPrestateOptions))
			})

			t.Run("NotRequiredIfCannonPrestatesBaseURLSet", func(t *testing.T) {
				configForArgs(t, addRequiredArgsExceptArr(gameType, allPrestateOptions, "--cannon-prestates-url=http://localhost/foo"))
			})

			t.Run("CannonPrestatesBaseURLTakesPrecedence", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExceptArr(gameType, allPrestateOptions, "--cannon-prestates-url=http://localhost/foo", "--prestates-url=http://localhost/bar"))
				require.Equal(t, "http://localhost/foo", cfg.CannonAbsolutePreStateBaseURL.String())
			})

			t.Run("RequiredIfCannonPrestatesBaseURLNotSet", func(t *testing.T) {
				verifyArgsInvalid(t, "flag prestates-url/cannon-prestates-url or cannon-prestate is required", addRequiredArgsExceptArr(gameType, allPrestateOptions))
			})

			t.Run("Invalid", func(t *testing.T) {
				verifyArgsInvalid(t, "invalid prestates-url (:foo/bar)", addRequiredArgsExceptArr(gameType, allPrestateOptions, "--prestates-url=:foo/bar"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExceptArr(gameType, allPrestateOptions, "--prestates-url=http://localhost/foo"))
				require.Equal(t, "http://localhost/foo", cfg.CannonAbsolutePreStateBaseURL.String())
			})
		})

		t.Run(fmt.Sprintf("TestL2Rpc-%v", gameType), func(t *testing.T) {
			t.Run("RequiredForCannonTrace", func(t *testing.T) {
				verifyArgsInvalid(t, "flag l2-eth-rpc is required", addRequiredArgsExcept(gameType, "--l2-eth-rpc"))
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgs(gameType))
				require.Equal(t, []string{l2EthRpc}, cfg.L2Rpcs)
			})
		})

		t.Run(fmt.Sprintf("TestCannonSnapshotFreq-%v", gameType), func(t *testing.T) {
			t.Run("UsesDefault", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgs(gameType))
				require.Equal(t, config.DefaultCannonSnapshotFreq, cfg.Cannon.SnapshotFreq)
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgs(gameType, "--cannon-snapshot-freq=1234"))
				require.Equal(t, uint(1234), cfg.Cannon.SnapshotFreq)
			})

			t.Run("Invalid", func(t *testing.T) {
				verifyArgsInvalid(t, "invalid value \"abc\" for flag -cannon-snapshot-freq",
					addRequiredArgs(gameType, "--cannon-snapshot-freq=abc"))
			})
		})

		t.Run(fmt.Sprintf("TestCannonInfoFreq-%v", gameType), func(t *testing.T) {
			t.Run("UsesDefault", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgs(gameType))
				require.Equal(t, config.DefaultCannonInfoFreq, cfg.Cannon.InfoFreq)
			})

			t.Run("Valid", func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgs(gameType, "--cannon-info-freq=1234"))
				require.Equal(t, uint(1234), cfg.Cannon.InfoFreq)
			})

			t.Run("Invalid", func(t *testing.T) {
				verifyArgsInvalid(t, "invalid value \"abc\" for flag -cannon-info-freq",
					addRequiredArgs(gameType, "--cannon-info-freq=abc"))
			})
		})
	}
}

func TestDepsetConfig(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType {
			t.Run("Required-"+gameType.String(), func(t *testing.T) {
				verifyArgsInvalid(t,
					"flag network or rollup-config/cannon-rollup-config, l2-genesis/cannon-l2-genesis and depset-config/cannon-depset-config is required",
					addRequiredArgsExcept(gameType, "--network", "--rollup-config=rollup.json", "--l2-genesis=genesis.json"))
			})
		} else if gameType == gameTypes.SuperCannonKonaGameType {
			t.Run("Required-"+gameType.String(), func(t *testing.T) {
				verifyArgsInvalid(t,
					"flag network or rollup-config/cannon-kona-rollup-config, l2-genesis/cannon-kona-l2-genesis and depset-config/cannon-kona-depset-config is required",
					addRequiredArgsExcept(gameType, "--network", "--rollup-config=rollup.json", "--l2-genesis=genesis.json"))
			})
		} else {
			t.Run("NotRequired-"+gameType.String(), func(t *testing.T) {
				cfg := configForArgs(t, addRequiredArgsExcept(gameType, "--network", "--rollup-config=rollup.json", "--l2-genesis=genesis.json"))
				require.Equal(t, "", cfg.Cannon.DepsetConfigPath)
			})
		}
	}
}

func TestDataDir(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType

		t.Run(fmt.Sprintf("RequiredFor-%v", gameType), func(t *testing.T) {
			verifyArgsInvalid(t, "flag datadir is required", addRequiredArgsExcept(gameType, "--datadir"))
		})
	}

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.CannonGameType, "--datadir", "--datadir=/foo/bar/cannon"))
		require.Equal(t, "/foo/bar/cannon", cfg.Datadir)
	})
}

func TestRollupRpc(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType

		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
			t.Run(fmt.Sprintf("NotRequiredFor-%v", gameType), func(t *testing.T) {
				configForArgs(t, addRequiredArgsExcept(gameType, "--rollup-rpc"))
			})
		} else {
			t.Run(fmt.Sprintf("RequiredFor-%v", gameType), func(t *testing.T) {
				verifyArgsInvalid(t, "flag rollup-rpc is required", addRequiredArgsExcept(gameType, "--rollup-rpc"))
			})
		}
	}

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType))
		require.Equal(t, rollupRpc, cfg.RollupRpc)
	})
}

func TestGameWindow(t *testing.T) {
	t.Run("UsesDefault", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType))
		require.Equal(t, config.DefaultGameWindow, cfg.GameWindow)
	})

	t.Run("Valid", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--game-window=1m"))
		require.Equal(t, time.Minute, cfg.GameWindow)
	})

	t.Run("ParsesDefault", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--game-window=672h"))
		require.Equal(t, config.DefaultGameWindow, cfg.GameWindow)
	})
}

func TestUnsafeAllowInvalidPrestate(t *testing.T) {
	t.Run("DefaultsToFalse", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--unsafe-allow-invalid-prestate"))
		require.False(t, cfg.AllowInvalidPrestate)
	})

	t.Run("EnabledWithNoValue", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType, "--unsafe-allow-invalid-prestate"))
		require.True(t, cfg.AllowInvalidPrestate)
	})

	t.Run("EnabledWithTrue", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType, "--unsafe-allow-invalid-prestate=true"))
		require.True(t, cfg.AllowInvalidPrestate)
	})

	t.Run("DisabledWithFalse", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.CannonGameType, "--unsafe-allow-invalid-prestate=false"))
		require.False(t, cfg.AllowInvalidPrestate)
	})
}

func TestAdditionalBondClaimants(t *testing.T) {
	t.Run("DefaultsToEmpty", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgsExcept(gameTypes.AlphabetGameType, "--additional-bond-claimants"))
		require.Empty(t, cfg.AdditionalBondClaimants)
	})

	t.Run("Valid-Single", func(t *testing.T) {
		claimant := common.Address{0xaa}
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--additional-bond-claimants", claimant.Hex()))
		require.Contains(t, cfg.AdditionalBondClaimants, claimant)
		require.Len(t, cfg.AdditionalBondClaimants, 1)
	})

	t.Run("Valid-Multiple", func(t *testing.T) {
		claimant1 := common.Address{0xaa}
		claimant2 := common.Address{0xbb}
		claimant3 := common.Address{0xcc}
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType,
			"--additional-bond-claimants", fmt.Sprintf("%v,%v,%v", claimant1.Hex(), claimant2.Hex(), claimant3.Hex())))
		require.Contains(t, cfg.AdditionalBondClaimants, claimant1)
		require.Contains(t, cfg.AdditionalBondClaimants, claimant2)
		require.Contains(t, cfg.AdditionalBondClaimants, claimant3)
		require.Len(t, cfg.AdditionalBondClaimants, 3)
	})

	t.Run("Invalid-Single", func(t *testing.T) {
		verifyArgsInvalid(t, "invalid additional claimant",
			addRequiredArgs(gameTypes.AlphabetGameType, "--additional-bond-claimants", "nope"))
	})

	t.Run("Invalid-Multiple", func(t *testing.T) {
		claimant1 := common.Address{0xaa}
		claimant2 := common.Address{0xbb}
		verifyArgsInvalid(t, "invalid additional claimant",
			addRequiredArgs(gameTypes.AlphabetGameType, "--additional-bond-claimants", fmt.Sprintf("%v,nope,%v", claimant1.Hex(), claimant2.Hex())))
	})
}

func TestSignerTLS(t *testing.T) {
	t.Run("EnabledByDefault", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType))
		require.True(t, cfg.TxMgrConfig.SignerCLIConfig.TLSConfig.Enabled)
	})

	t.Run("Disabled", func(t *testing.T) {
		cfg := configForArgs(t, addRequiredArgs(gameTypes.AlphabetGameType, "--signer.tls.enabled=false"))
		require.False(t, cfg.TxMgrConfig.SignerCLIConfig.TLSConfig.Enabled)
	})
}

func verifyArgsInvalid(t *testing.T, messageContains string, cliArgs []string) {
	_, _, err := dryRunWithArgs(cliArgs)
	require.ErrorContains(t, err, messageContains)
}

func configForArgs(t *testing.T, cliArgs []string) config.Config {
	_, cfg, err := dryRunWithArgs(cliArgs)
	require.NoError(t, err)
	return cfg
}

func dryRunWithArgs(cliArgs []string) (log.Logger, config.Config, error) {
	cfg := new(config.Config)
	var logger log.Logger
	fullArgs := append([]string{"op-challenger"}, cliArgs...)
	testErr := errors.New("dry-run")
	err := run(context.Background(), fullArgs, func(ctx context.Context, log log.Logger, config *config.Config) (cliapp.Lifecycle, error) {
		logger = log
		cfg = config
		return nil, testErr
	})
	if errors.Is(err, testErr) { // expected error
		err = nil
	}
	return logger, *cfg, err
}

func addRequiredArgs(gameType gameTypes.GameType, args ...string) []string {
	req := requiredArgs(gameType)
	combined := toArgList(req)
	return append(combined, args...)
}

func addRequiredArgsExcept(gameType gameTypes.GameType, name string, optionalArgs ...string) []string {
	req := requiredArgs(gameType)
	delete(req, name)
	return append(toArgList(req), optionalArgs...)
}

func addRequiredArgsForMultipleGameTypesExcept(gameType []gameTypes.GameType, name string, optionalArgs ...string) []string {
	req := requiredArgsMultiple(gameType)
	delete(req, name)
	return append(toArgList(req), optionalArgs...)
}

func addRequiredArgsExceptArr(gameType gameTypes.GameType, names []string, optionalArgs ...string) []string {
	req := requiredArgs(gameType)
	for _, name := range names {
		delete(req, name)
	}
	return append(toArgList(req), optionalArgs...)
}

func requiredArgsMultiple(gameType []gameTypes.GameType) map[string]string {
	args := make(map[string]string)
	for _, t := range gameType {
		for name, value := range requiredArgs(t) {
			args[name] = value
		}
	}
	return args
}

func requiredArgs(gameType gameTypes.GameType) map[string]string {
	args := map[string]string{
		"--l1-eth-rpc":           l1EthRpc,
		"--l1-beacon":            l1Beacon,
		"--l2-eth-rpc":           l2EthRpc,
		"--game-factory-address": gameFactoryAddressValue,
		"--game-types":           gameType.String(),
		"--datadir":              datadir,
	}
	switch gameType {
	case gameTypes.CannonGameType, gameTypes.PermissionedGameType:
		addRequiredCannonArgs(args)
	case gameTypes.CannonKonaGameType:
		addRequiredCannonKonaArgs(args)
	case gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType:
		addRequiredSuperCannonArgs(args)
	case gameTypes.SuperCannonKonaGameType:
		addRequiredSuperCannonKonaArgs(args)
	case gameTypes.OptimisticZKGameType, gameTypes.AlphabetGameType, gameTypes.FastGameType:
		addRequiredOutputRootArgs(args)
	}
	return args
}

func addRequiredSuperCannonArgs(args map[string]string) {
	addRequiredCannonBaseArgs(args)
	args["--supernode-rpc"] = superRpc
}

func addRequiredCannonArgs(args map[string]string) {
	addRequiredCannonBaseArgs(args)
	addRequiredOutputRootArgs(args)
}

func addRequiredCannonKonaArgs(args map[string]string) {
	addRequiredCannonKonaBaseArgs(args)
	addRequiredOutputRootArgs(args)
}

func addRequiredOutputRootArgs(args map[string]string) {
	args["--rollup-rpc"] = rollupRpc
}

func addRequiredCannonBaseArgs(args map[string]string) {
	args["--network"] = network
	args["--cannon-bin"] = cannonBin
	args["--cannon-server"] = cannonServer
	args["--cannon-prestate"] = cannonPreState
}

func addRequiredCannonKonaBaseArgs(args map[string]string) {
	args["--network"] = network
	args["--cannon-bin"] = cannonBin
	args["--cannon-kona-server"] = cannonKonaServer
	args["--cannon-kona-prestate"] = cannonKonaPreState
}

func addRequiredSuperCannonKonaArgs(args map[string]string) {
	addRequiredCannonKonaBaseArgs(args)
	args["--supernode-rpc"] = superRpc
}

func toArgList(req map[string]string) []string {
	var combined []string
	for name, value := range req {
		combined = append(combined, fmt.Sprintf("%s=%s", name, value))
	}
	return combined
}
