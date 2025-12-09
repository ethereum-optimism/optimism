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
	validSupervisorRpc                    = "http://localhost/supervisor"

	validAsteriscBin                        = "./bin/asterisc"
	validAsteriscOpProgramBin               = "./bin/op-program"
	validAsteriscNetwork                    = "mainnet"
	validAsteriscAbsolutePreState           = "pre.json"
	validAsteriscAbsolutePreStateBaseURL, _ = url.Parse("http://localhost/bar/")

	nonExistingFile                             = "path/to/nonexistent/file"
	validAsteriscKonaBin                        = "./bin/asterisc"
	validAsteriscKonaServerBin                  = "./bin/kona-host"
	validAsteriscKonaNetwork                    = "mainnet"
	validAsteriscKonaAbsolutePreState           = "pre.json"
	validAsteriscKonaAbsolutePreStateBaseURL, _ = url.Parse("http://localhost/bar/")

	validCannonKonaBin                        = "./bin/cannon"
	validCannonKonaServerBin                  = "./bin/kona-host"
	validCannonKonaNetwork                    = "mainnet"
	validCannonKonaAbsolutePreStateBaseURL, _ = url.Parse("http://localhost/bar/")
)

var singleCannonGameTypes = []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.PermissionedGameType}
var superCannonGameTypes = []gameTypes.GameType{gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType}
var allCannonGameTypes []gameTypes.GameType
var cannonKonaGameTypes = []gameTypes.GameType{gameTypes.CannonKonaGameType, gameTypes.SuperCannonKonaGameType}
var asteriscGameTypes = []gameTypes.GameType{gameTypes.AsteriscGameType}
var asteriscKonaGameTypes = []gameTypes.GameType{gameTypes.AsteriscKonaGameType}
var superAsteriscKonaGameTypes = []gameTypes.GameType{gameTypes.SuperAsteriscKonaGameType}

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
	cfg.SupervisorRPC = validSupervisorRpc
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

func applyValidConfigForAsterisc(t *testing.T, cfg *Config) {
	tmpDir := t.TempDir()
	vmBin := filepath.Join(tmpDir, validAsteriscBin)
	server := filepath.Join(tmpDir, validAsteriscOpProgramBin)
	err := ensureExists(vmBin)
	require.NoError(t, err)
	err = ensureExists(server)
	require.NoError(t, err)
	cfg.Asterisc.VmBin = vmBin
	cfg.Asterisc.Server = server
	cfg.AsteriscAbsolutePreStateBaseURL = validAsteriscAbsolutePreStateBaseURL
	cfg.Asterisc.Networks = []string{validAsteriscNetwork}
}

func applyValidConfigForAsteriscKona(t *testing.T, cfg *Config) {
	tmpDir := t.TempDir()
	vmBin := filepath.Join(tmpDir, validAsteriscKonaBin)
	server := filepath.Join(tmpDir, validAsteriscKonaServerBin)
	err := ensureExists(vmBin)
	require.NoError(t, err)
	err = ensureExists(server)
	require.NoError(t, err)
	cfg.AsteriscKona.VmBin = vmBin
	cfg.AsteriscKona.Server = server
	cfg.AsteriscKonaAbsolutePreStateBaseURL = validAsteriscKonaAbsolutePreStateBaseURL
	cfg.AsteriscKona.Networks = []string{validAsteriscKonaNetwork}
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
	cfg.SupervisorRPC = validSupervisorRpc
	applyValidConfigForCannonKona(t, cfg)
}

func applyValidConfigForSuperAsteriscKona(t *testing.T, cfg *Config) {
	cfg.SupervisorRPC = validSupervisorRpc
	applyValidConfigForAsteriscKona(t, cfg)
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
	if gameType == gameTypes.AsteriscGameType {
		applyValidConfigForAsterisc(t, &cfg)
	}
	if gameType == gameTypes.AsteriscKonaGameType {
		applyValidConfigForAsteriscKona(t, &cfg)
	}
	if gameType == gameTypes.SuperAsteriscKonaGameType {
		applyValidConfigForSuperAsteriscKona(t, &cfg)
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
	if slices.Contains(asteriscGameTypes, gameType) {
		mutateVmConfig(&cfg.Asterisc)
	}
	if slices.Contains(asteriscKonaGameTypes, gameType) {
		mutateVmConfig(&cfg.AsteriscKona)
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

	for _, gameType := range superAsteriscKonaGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("TestAsteriscNetworkOrDepsetConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.Networks = nil
			cfg.AsteriscKona.RollupConfigPaths = []string{"foo.json"}
			cfg.AsteriscKona.L2GenesisPaths = []string{"genesis.json"}
			cfg.AsteriscKona.DepsetConfigPath = ""
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

	for _, gameType := range asteriscKonaGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("TestDepsetConfigNotRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.Networks = nil
			cfg.AsteriscKona.RollupConfigPaths = []string{"foo.json"}
			cfg.AsteriscKona.L1GenesisPath = "bar.json"
			cfg.AsteriscKona.L2GenesisPaths = []string{"genesis.json"}
			cfg.AsteriscKona.DepsetConfigPath = ""
			require.NoError(t, cfg.Check())
		})
	}
}

func TestAsteriscRequiredArgs(t *testing.T) {
	for _, gameType := range asteriscGameTypes {
		gameType := gameType

		t.Run(fmt.Sprintf("TestAsteriscBinRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.Asterisc.VmBin = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestAsteriscServerRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.Asterisc.Server = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingServer)
		})

		t.Run(fmt.Sprintf("TestAsteriscAbsolutePreStateOrBaseURLRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscAbsolutePreState = ""
			config.AsteriscAbsolutePreStateBaseURL = nil
			require.ErrorIs(t, config.Check(), ErrMissingAsteriscAbsolutePreState)
		})

		t.Run(fmt.Sprintf("TestAsteriscAbsolutePreState-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscAbsolutePreState = validAsteriscAbsolutePreState
			config.AsteriscAbsolutePreStateBaseURL = nil
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAsteriscAbsolutePreStateBaseURL-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscAbsolutePreState = ""
			config.AsteriscAbsolutePreStateBaseURL = validAsteriscAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAllowSupplingBothAsteriscAbsolutePreStateAndBaseURL-%v", gameType), func(t *testing.T) {
			// Since the prestate base URL might be inherited from the --prestate-urls option, allow overriding it with a specific prestate
			config := validConfig(t, gameType)
			config.AsteriscAbsolutePreState = validAsteriscAbsolutePreState
			config.AsteriscAbsolutePreStateBaseURL = validAsteriscAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestL2RpcRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.L2Rpcs = nil
			require.ErrorIs(t, config.Check(), ErrMissingL2Rpc)
		})

		t.Run(fmt.Sprintf("TestAsteriscSnapshotFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.Asterisc.SnapshotFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscSnapshotFreq)
			})
		})

		t.Run(fmt.Sprintf("TestAsteriscInfoFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.Asterisc.InfoFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscInfoFreq)
			})
		})

		t.Run(fmt.Sprintf("TestAsteriscNetworkOrRollupConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.Asterisc.RollupConfigPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingRollupConfig)
		})

		t.Run(fmt.Sprintf("TestAsteriscNetworkOrL2GenesisRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.Asterisc.L2GenesisPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingL2Genesis)
		})

		t.Run(fmt.Sprintf("MaySpecifyNetworkAndCustomConfigs-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Asterisc.Networks = []string{validAsteriscNetwork}
			cfg.Asterisc.RollupConfigPaths = []string{"foo.json"}
			cfg.Asterisc.L2GenesisPaths = []string{"genesis.json"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkMustBeValid-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Asterisc.Networks = []string{"unknown"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestDebugInfoDisabled-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			require.False(t, cfg.Asterisc.DebugInfo)
		})

		t.Run(fmt.Sprintf("TestVMBinExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Asterisc.VmBin = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestServerExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.Asterisc.Server = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
		})
	}
}

func TestAsteriscKonaRequiredArgs(t *testing.T) {
	for _, gameType := range asteriscKonaGameTypes {
		gameType := gameType

		t.Run(fmt.Sprintf("TestAsteriscKonaBinRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscKona.VmBin = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaServerRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscKona.Server = ""
			require.ErrorIs(t, config.Check(), vm.ErrMissingServer)
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaAbsolutePreStateOrBaseURLRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscKonaAbsolutePreState = ""
			config.AsteriscKonaAbsolutePreStateBaseURL = nil
			require.ErrorIs(t, config.Check(), ErrMissingAsteriscKonaAbsolutePreState)
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaAbsolutePreState-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscKonaAbsolutePreState = validAsteriscKonaAbsolutePreState
			config.AsteriscKonaAbsolutePreStateBaseURL = nil
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaAbsolutePreStateBaseURL-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.AsteriscKonaAbsolutePreState = ""
			config.AsteriscKonaAbsolutePreStateBaseURL = validAsteriscKonaAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestAllowSupplyingBothAsteriscKonaAbsolutePreStateAndBaseURL-%v", gameType), func(t *testing.T) {
			// Since the prestate base URL might be inherited from the --prestate-urls option, allow overriding it with a specific prestate
			config := validConfig(t, gameType)
			config.AsteriscKonaAbsolutePreState = validAsteriscKonaAbsolutePreState
			config.AsteriscKonaAbsolutePreStateBaseURL = validAsteriscKonaAbsolutePreStateBaseURL
			require.NoError(t, config.Check())
		})

		t.Run(fmt.Sprintf("TestL2RpcRequired-%v", gameType), func(t *testing.T) {
			config := validConfig(t, gameType)
			config.L2Rpcs = nil
			require.ErrorIs(t, config.Check(), ErrMissingL2Rpc)
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaSnapshotFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.AsteriscKona.SnapshotFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscKonaSnapshotFreq)
			})
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaInfoFreq-%v", gameType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(t, gameType)
				cfg.AsteriscKona.InfoFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscKonaInfoFreq)
			})
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaNetworkOrRollupConfigRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.AsteriscKona.RollupConfigPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingRollupConfig)
		})

		t.Run(fmt.Sprintf("TestAsteriscKonaNetworkOrL2GenesisRequired-%v", gameType), func(t *testing.T) {
			cfg := validConfigWithNoNetworks(t, gameType)
			cfg.AsteriscKona.L2GenesisPaths = nil
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingL2Genesis)
		})

		t.Run(fmt.Sprintf("MaySpecifyNetworkAndCustomConfig-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.Networks = []string{validAsteriscKonaNetwork}
			cfg.AsteriscKona.RollupConfigPaths = []string{"foo.json"}
			cfg.AsteriscKona.L2GenesisPaths = []string{"genesis.json"}
			require.NoError(t, cfg.Check())
		})

		t.Run(fmt.Sprintf("TestNetworkMustBeValid-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.Networks = []string{"unknown"}
			require.ErrorIs(t, cfg.Check(), vm.ErrNetworkUnknown)
		})

		t.Run(fmt.Sprintf("TestDebugInfoDisabled-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			require.False(t, cfg.AsteriscKona.DebugInfo)
		})

		t.Run(fmt.Sprintf("TestVMBinExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.VmBin = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingBin)
		})

		t.Run(fmt.Sprintf("TestServerExists-%v", gameType), func(t *testing.T) {
			cfg := validConfig(t, gameType)
			cfg.AsteriscKona.Server = nonExistingFile
			require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
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
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperAsteriscKonaGameType || gameType == gameTypes.SuperCannonKonaGameType {
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

	t.Run("SuperAsteriscKona", func(t *testing.T) {
		config := validConfig(t, gameTypes.SuperAsteriscKonaGameType)
		config.RollupRpc = ""
		require.NoError(t, config.Check())
	})
}

func TestSupervisorRpc(t *testing.T) {
	for _, gameType := range gameTypes.SupportedGameTypes {
		gameType := gameType
		if gameType == gameTypes.SuperCannonGameType || gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperAsteriscKonaGameType || gameType == gameTypes.SuperCannonKonaGameType {
			t.Run("RequiredFor"+gameType.String(), func(t *testing.T) {
				config := validConfig(t, gameType)
				config.SupervisorRPC = ""
				require.ErrorIs(t, config.Check(), ErrMissingSupervisorRpc)
			})
		} else {
			t.Run("NotRequiredFor"+gameType.String(), func(t *testing.T) {
				config := validConfig(t, gameType)
				config.SupervisorRPC = ""
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

func TestRequireConfigForMultipleGameTypesForAsterisc(t *testing.T) {
	cfg := validConfig(t, gameTypes.AsteriscGameType)
	cfg.GameTypes = []gameTypes.GameType{gameTypes.AsteriscGameType, gameTypes.AlphabetGameType}
	// Set all required options and check its valid
	cfg.RollupRpc = validRollupRpc
	require.NoError(t, cfg.Check())

	// Require asterisc specific args
	cfg.AsteriscAbsolutePreState = ""
	cfg.AsteriscAbsolutePreStateBaseURL = nil
	require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscAbsolutePreState)
	cfg.AsteriscAbsolutePreState = validAsteriscAbsolutePreState

	// Require output asterisc specific args
	cfg.RollupRpc = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
}

func TestRequireConfigForMultipleGameTypesForCannonAndAsterisc(t *testing.T) {
	cfg := validConfig(t, gameTypes.CannonGameType)
	applyValidConfigForAsterisc(t, &cfg)

	cfg.GameTypes = []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.AsteriscGameType, gameTypes.AlphabetGameType, gameTypes.FastGameType}
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

	// Require asterisc specific args
	cfg.AsteriscAbsolutePreState = ""
	cfg.AsteriscAbsolutePreStateBaseURL = nil
	require.ErrorIs(t, cfg.Check(), ErrMissingAsteriscAbsolutePreState)
	cfg.AsteriscAbsolutePreState = validAsteriscAbsolutePreState

	cfg.Asterisc.Server = ""
	require.ErrorIs(t, cfg.Check(), vm.ErrMissingServer)
	server := filepath.Join(tmpDir, validAsteriscOpProgramBin)
	err = ensureExists(server)
	require.NoError(t, err)
	cfg.Asterisc.Server = server

	// Check final config is valid
	require.NoError(t, cfg.Check())
}
