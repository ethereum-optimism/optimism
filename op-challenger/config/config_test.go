package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	validL1EthRpc                         = "http://localhost:8545"
	validL1BeaconUrl                      = "http://localhost:9000"
	validGameFactoryAddress               = common.Address{0x23}
	validCannonBin                        = "./bin/cannon"
	validCannonOpProgramBin               = "./bin/op-program"
	validCannonNetwork                    = "mainnet"
	validCannonAbsolutePreState           = "pre.json"
	validCannonAbsolutePreStateBaseURL, _ = url.Parse("http://localhost/foo/")
	validDatadir                          = "/tmp/data"
	validL2Rpc                            = "http://localhost:9545"
	validRollupRpc                        = "http://localhost:8555"
	validSuperRpc                         = "http://localhost/super"

	nonExistingFile = "path/to/nonexistent/file"

	validCannonKonaBin                        = "./bin/cannon"
	validCannonKonaServerBin                  = "./bin/kona-host"
	validCannonKonaNetwork                    = "mainnet"
	validCannonKonaAbsolutePreStateBaseURL, _ = url.Parse("http://localhost/bar/")
)

var singleCannonGameTypes = []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.PermissionedGameType}
var superCannonGameTypes = []gameTypes.GameType{gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType}
var allCannonGameTypes []gameTypes.GameType
var cannonKonaGameTypes = []gameTypes.GameType{gameTypes.CannonKonaGameType, gameTypes.SuperCannonKonaGameType}

func init() {
	allCannonGameTypes = append(allCannonGameTypes, singleCannonGameTypes...)
	allCannonGameTypes = append(allCannonGameTypes, superCannonGameTypes...)
}

func ensureExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func applyValidConfigForSuperCannon(t *testing.T, cfg *Config) {
	cfg.SuperRPC = validSuperRpc
	applyValidConfigForCannon(t, cfg)
}

func applyValidConfigForCannon(t *testing.T, cfg *Config) {
	tmpDir := t.TempDir()
	vmBin := filepath.Join(tmpDir, validCannonBin)
	server := filepath.Join(tmpDir, validCannonOpProgramBin)
	err := ensureExists(vmBin)
	require.NoError(t, err)
	err = ensureExists(server)
	require.NoError(t, err)
	cfg.Cannon.VmBin = vmBin
	cfg.Cannon.Server = server
	cfg.CannonAbsolutePreStateBaseURL = validCannonAbsolutePreStateBaseURL
	cfg.Cannon.Networks = []string{validCannonNetwork}
}

func applyValidConfigForCannonKona(t *testing.T, cfg *Config) {
	tmpDir := t.TempDir()
	vmBin := filepath.Join(tmpDir, validCannonKonaBin)
	server := filepath.Join(tmpDir, validCannonKonaServerBin)
	err := ensureExists(vmBin)
	require.NoError(t, err)
	err = ensureExists(server)
	require.NoError(t, err)
	cfg.CannonKona.VmBin = vmBin
	cfg.CannonKona.Server = server
	cfg.CannonKonaAbsolutePreStateBaseURL = validCannonKonaAbsolutePreStateBaseURL
	cfg.CannonKona.Networks = []string{validCannonKonaNetwork}
}

func applyValidConfigForSuperCannonKona(t *testing.T, cfg *Config) {
	cfg.SuperRPC = validSuperRpc
	applyValidConfigForCannonKona(t, cfg)
}

func applyValidConfigForOptimisticZK(cfg *Config) {
	cfg.RollupRpc = validRollupRpc
}

func validConfig(t *testing.T, gameType gameTypes.GameType) Config {
	cfg := NewConfig(validGameFactoryAddress, validL1EthRpc, validL1BeaconUrl, validRollupRpc, validL2Rpc, validDatadir, gameType)
	if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType {
		applyValidConfigForSuperCannon(t, &cfg)
	}
	if gameType == gameTypes.CannonGameType || gameType == gameTypes.PermissionedGameType {
		applyValidConfigForCannon(t, &cfg)
	}
	if gameType == gameTypes.CannonKonaGameType {
		applyValidConfigForCannonKona(t, &cfg)
	}
	if gameType == gameTypes.SuperCannonKonaGameType {
		applyValidConfigForSuperCannonKona(t, &cfg)
	}
	if gameType == gameTypes.OptimisticZKGameType {
		applyValidConfigForOptimisticZK(&cfg)
	}
	return cfg
}

func validConfigWithNoNetworks(t *testing.T, gameType gameTypes.GameType) Config {
	cfg := validConfig(t, gameType)

	mutateVmConfig := func(cfg *vm.Config) {
		cfg.Networks = nil
		cfg.RollupConfigPaths = []string{"foo.json"}
		cfg.L2GenesisPaths = []string{"genesis.json"}
		cfg.L1GenesisPath = "bar.json"
		cfg.DepsetConfigPath = "foo.json"
	}
	if slices.Contains(allCannonGameTypes, gameType) {
		mutateVmConfig(&cfg.Cannon)
	}
	if slices.Contains(cannonKonaGameTypes, gameType) {
		mutateVmConfig(&cfg.CannonKona)
	}
	return cfg
}

// TestValidConfigIsValid checks that the config provided by validConfig is actually valid
func TestValidConfigIsValid(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		t.Run(gameType.String(), func(t *testing.T) {
			err := validConfig(t, gameType).Check()
			require.NoError(t, err)
		})
	}
}

func TestTxMgrConfig(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		config := validConfig(t, gameTypes.CannonGameType)
		config.TxMgrConfig = txmgr.CLIConfig{}
		require.Equal(t, config.Check().Error(), "must provide a L1 RPC url")
	})
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	config.L1EthRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1EthRPC)
}

func TestL1EthRpcKindRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	config.L1RPCKind = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1RPCKind)
}

func TestL1BeaconRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	config.L1Beacon = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1Beacon)
}

func TestGameFactoryAddressRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	config.GameFactoryAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameFactoryAddress)
}

func TestSelectiveClaimResolutionNotRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	require.Equal(t, false, config.SelectiveClaimResolution)
	require.NoError(t, config.Check())
}

func TestGameAllowlistNotRequired(t *testing.T) {
	config := validConfig(t, gameTypes.CannonGameType)
	config.GameAllowlist = []common.Address{}
	require.NoError(t, config.Check())
}

func TestCannonRequiredArgs(t *testing.T) {
	for _, gameType := range allCannonGameTypes {
		gameType := gameType

		t.Run(fmt.Sprintf("TestCannonBinRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.Cannon.VmBin = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestCannonServerRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.Cannon.Server = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingServer)
		})

		t.Run(fmt.Sprintf("TestCannonAbsolutePreStateOrBaseURLRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonAbsolutePreState = ""
			config.CannonAbsolutePreStateBaseURL = nil
			require.ErrorIs(t, config.Check(), ErrMissingCannonAbsolutePreState)
		})

		t.Run(fmt.Sprintf("TestCannonAbsolutePreState-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonAbsolutePreState = validCannonAbsolutePreState
			config.CannonAbsolutePreStateBaseURL = nil
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestCannonAbsolutePreStateBaseURL-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonAbsolutePreState = ""
			config.CannonAbsolutePreStateBaseURL = validCannonAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAllowSupplyingBothCannonAbsolutePreStateAndBaseURL-%v", gameType), func(t *testing.T) {
			// Since the prestate baseURL might be inherited from the --prestate-urls option, allow overriding it with a specific prestate
			config := validConfig(t, gameType)
			config.CannonAbsolutePreState = validCannonAbsolutePreState
			config.CannonAbsolutePreStateBaseURL = validCannonAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestL2RpcRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.L2Rpcs = nil
			require.ErrorIs(t, config.Check(), ErrMissingL2Rpc)
		})

		t.Run(fmt.Sprintf("TestCannonSnapshotFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.Cannon.SnapshotFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonSnapshotFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonInfoFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.Cannon.InfoFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonInfoFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonNetworkOrRollupConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.Cannon.RollupConfigPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingRollupConfig)
		})

		t.Run(fmt.Sprintf("TestCannonNetworkOrL2GenesisRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.Cannon.L2GenesisPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingL2Genesis)
		})

		t.Run(fmt.Sprintf("TestMaySpecifyNetworkAndCustomConfigs-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = []string{validCannonNetwork}
			cfg.Cannon.RollupConfigPaths = []string{"foo.json"}
			cfg.Cannon.L2GenesisPaths = []string{"genesis.json"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkMustBeValid-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = []string{"unknown"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestNetworkMayBeAnyChainID-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = []string{"467294"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkInvalidWhenNotEntirelyNumeric-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = []string{"467294a"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestDebugInfoEnabled-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			require.True(t, cfg.Cannon.DebugInfo)
		})

		t.Run(fmt.Sprintf("TestVMBinExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.VmBin = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestServerExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Server = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
		})
	}
}

func TestCannonKonaRequiredArgs(t *testing.T) {
	for _, gameType := range cannonKonaGameTypes {
		gameType := gameType

		t.Run(fmt.Sprintf("TestCannonKonaBinRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonKona.VmBin = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestCannonKonaServerRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonKona.Server = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingServer)
		})

		t.Run(fmt.Sprintf("TestCannonKonaAbsolutePreStateOrBaseURLRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonKonaAbsolutePreState = ""
			config.CannonKonaAbsolutePreStateBaseURL = nil
			require.ErrorIs(t, config.Check(), ErrMissingCannonKonaAbsolutePreState)
		})

		t.Run(fmt.Sprintf("TestCannonKonaAbsolutePreState-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonKonaAbsolutePreState = validCannonAbsolutePreState
			config.CannonKonaAbsolutePreStateBaseURL = nil
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestCannonKonaAbsolutePreStateBaseURL-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.CannonKonaAbsolutePreState = ""
			config.CannonKonaAbsolutePreStateBaseURL = validCannonAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAllowSupplyingBothCannonKonaAbsolutePreStateAndBaseURL-%v", gameType), func(t *testing.T) {
			// Since the prestate baseURL might be inherited from the --prestate-urls option, allow overriding it with a specific prestate
			config := validConfig(t, gameType)
			config.CannonKonaAbsolutePreState = validCannonAbsolutePreState
			config.CannonKonaAbsolutePreStateBaseURL = validCannonAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestL2RpcRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.L2Rpcs = nil
			require.ErrorIs(t, config.Check(), ErrMissingL2Rpc)
		})

		t.Run(fmt.Sprintf("TestCannonKonaSnapshotFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.CannonKona.SnapshotFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonKonaSnapshotFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonKonaInfoFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.CannonKona.InfoFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonKonaInfoFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonKonaNetworkOrRollupConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.CannonKona.RollupConfigPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingRollupConfig)
		})

		t.Run(fmt.Sprintf("TestCannonKonaNetworkOrL2GenesisRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.CannonKona.L2GenesisPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingL2Genesis)
		})

		t.Run(fmt.Sprintf("TestMaySpecifyNetworkAndCustomConfigs-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.Networks = []string{validCannonNetwork}
			cfg.CannonKona.RollupConfigPaths = []string{"foo.json"}
			cfg.CannonKona.L2GenesisPaths = []string{"genesis.json"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkMustBeValid-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.Networks = []string{"unknown"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestNetworkMayBeAnyChainID-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.Networks = []string{"467294"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkInvalidWhenNotEntirelyNumeric-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.Networks = []string{"467294a"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestDebugInfoEnabled-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			require.True(t, cfg.CannonKona.DebugInfo)
		})

		t.Run(fmt.Sprintf("TestVMBinExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.VmBin = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestServerExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.CannonKona.Server = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
		})
	}
}

func TestDepsetConfig(t *testing.T) {
	for _, gameType := range superCannonGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("TestCannonNetworkOrDepsetConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = nil
			cfg.Cannon.RollupConfigPaths = []string{"foo.json"}
			cfg.Cannon.L2GenesisPaths = []string{"genesis.json"}
			cfg.Cannon.DepsetConfigPath = ""
			require.ErrorIs(t, cfg.Check(), ErrMissingDepsetConfig)
		})
	}

	for _, gameType := range singleCannonGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("TestDepsetConfigNotRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Cannon.Networks = nil
			cfg.Cannon.RollupConfigPaths = []string{"foo.json"}
			cfg.Cannon.L1GenesisPath = "bar.json"
			cfg.Cannon.L2GenesisPaths = []string{"genesis.json"}
			cfg.Cannon.DepsetConfigPath = ""
			require.NoError(t, cfg.Check())
		})
	}
}

func TestDatadirRequired(t *testing.T) {
	config := validConfig(t, gameTypes.AlphabetGameType)
	config.Datadir = ""
	require.ErrorIs(t, config.Check(), ErrMissingDatadir)
}

func TestMaxConcurrency(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		config := validConfig(t, gameTypes.AlphabetGameType)
		config.MaxConcurrency = 0
		require.ErrorIs(t, config.Check(), ErrMaxConcurrencyZero)
	})

	t.Run("DefaultToNumberOfCPUs", func(t *testing.T) {
		config := validConfig(t, gameTypes.AlphabetGameType)
		require.EqualValues(t, runtime.NumCPU(), config.MaxConcurrency)
	})
}

func TestHttpPollInterval(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		config := validConfig(t, gameTypes.AlphabetGameType)
		require.EqualValues(t, DefaultPollInterval, config.PollInterval)
	})
}

func TestRollupRpcRequired(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
			continue
		}
		t.Run(gameType.String(), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.RollupRpc = ""
			require.ErrorIs(t, config.Check(), ErrMissingRollupRpc)
		})
	}
}

func TestRollupRpcNotRequiredForInterop(t *testing.T) {
	t.Run("SuperCannon", func(t *testing.T) {
		config := validConfig(t, gameTypes.SuperCannonGameType)
		config.RollupRpc = ""
		require.NoError(t, config.Check())
	})

	t.Run("SuperPermissioned", func(t *testing.T) {
		config := validConfig(t, gameTypes.SuperPermissionedGameType)
		config.RollupRpc = ""
		require.NoError(t, config.Check())
	})

	t.Run("SuperCannonKona", func(t *testing.T) {
		config := validConfig(t, gameTypes.SuperCannonKonaGameType)
		config.RollupRpc = ""
		require.NoError(t, config.Check())
	})
}

func TestSuperRpc(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
			t.Run("RequiredFor"+gameType.String(), func(t *testing.T) {
				config := validConfig(t, gameType)
				config.SuperRPC = ""
				require.ErrorIs(t, config.Check(), ErrMissingSuperRpc)
			})
		} else {
			t.Run("NotRequiredFor"+gameType.String(), func(t *testing.T) {
				config := validConfig(t, gameType)
				config.SuperRPC = ""
				require.NoError(t, config.Check())
			})
		}
	}
}

func TestRequireConfigForMultipleGameTypesForCannon(t *testing.T) {
	cfg := validConfig(t, gameTypes.CannonGameType)
	cfg.GameTypes = []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.AlphabetGameType}
	// Set all required options and check its valid
	cfg.RollupRpc = validRollupRpc
	require.NoError(t, cfg.Check())

	// Require cannon specific args
	cfg.CannonAbsolutePreState = ""
	cfg.CannonAbsolutePreStateBaseURL = nil
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonAbsolutePreState)
	cfg.CannonAbsolutePreState = validCannonAbsolutePreState

	// Require output cannon specific args
	cfg.RollupRpc = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
}

func TestRequireConfigForMultipleGameTypesForCannonAndCannonKona(t *testing.T) {
	cfg := validConfig(t, gameTypes.CannonGameType)
	applyValidConfigForCannonKona(t, &cfg)

	cfg.GameTypes = []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.CannonKonaGameType, gameTypes.AlphabetGameType, gameTypes.FastGameType}
	// Set all required options and check its valid
	cfg.RollupRpc = validRollupRpc
	require.NoError(t, cfg.Check())

	// Require cannon specific args
	cfg.Cannon.VmBin = ""
	require.ErrorIs(t, cfg.Check(), vm.ErrMissingBin)
	tmpDir := t.TempDir()
	vmBin := filepath.Join(tmpDir, validCannonBin)
	err := ensureExists(vmBin)
	require.NoError(t, err)
	cfg.Cannon.VmBin = vmBin

	// Require cannon-kona specific args
	cfg.CannonKonaAbsolutePreState = ""
	cfg.CannonKonaAbsolutePreStateBaseURL = nil
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonKonaAbsolutePreState)
	cfg.CannonKonaAbsolutePreStateBaseURL = validCannonKonaAbsolutePreStateBaseURL

	cfg.CannonKona.Server = ""
	require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
	cfg.CannonKona.Server = vmBin

	// Check final config is valid
	require.NoError(t, cfg.Check())
}
