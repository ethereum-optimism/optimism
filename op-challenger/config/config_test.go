package config

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	validL1EthRpc              = "http://localhost:8545"
	validL1BeaconUrl           = "http://localhost:9000"
	validGameFactoryAddress    = common.Address{0x23}
	validCannonBin             = "./bin/cannon"
	validCannonOpProgramBin    = "./bin/op-program"
	validCannonNetwork         = "mainnet"
	validCannonAbsolutPreState = "pre.json"
	validDatadir               = "/tmp/data"
	validCannonL2              = "http://localhost:9545"
	validRollupRpc             = "http://localhost:8555"
)

var cannonTraceTypes = []TraceType{TraceTypeCannon, TraceTypePermissioned}

func validConfig(traceType TraceType) Config {
	cfg := NewConfig(validGameFactoryAddress, validL1EthRpc, validL1BeaconUrl, validDatadir, traceType)
	if traceType == TraceTypeCannon || traceType == TraceTypePermissioned {
		cfg.CannonBin = validCannonBin
		cfg.CannonServer = validCannonOpProgramBin
		cfg.CannonAbsolutePreState = validCannonAbsolutPreState
		cfg.CannonL2 = validCannonL2
		cfg.CannonNetwork = validCannonNetwork
	}
	cfg.RollupRpc = validRollupRpc
	return cfg
}

// TestValidConfigIsValid checks that the config provided by validConfig is actually valid
func TestValidConfigIsValid(t *testing.T) {
	for _, traceType := range TraceTypes {
		traceType := traceType
		t.Run(traceType.String(), func(t *testing.T) {
			err := validConfig(traceType).Check()
			require.NoError(t, err)
		})
	}
}

func TestTxMgrConfig(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		config := validConfig(TraceTypeCannon)
		config.TxMgrConfig = txmgr.CLIConfig{}
		require.Equal(t, config.Check().Error(), "must provide a L1 RPC url")
	})
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.L1EthRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1EthRPC)
}

func TestL1BeaconRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.L1Beacon = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1Beacon)
}

func TestGameFactoryAddressRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.GameFactoryAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameFactoryAddress)
}

func TestSelectiveClaimResolutionNotRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	require.Equal(t, false, config.SelectiveClaimResolution)
	require.NoError(t, config.Check())
}

func TestGameAllowlistNotRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.GameAllowlist = []common.Address{}
	require.NoError(t, config.Check())
}

func TestCannonRequiredArgs(t *testing.T) {
	for _, traceType := range cannonTraceTypes {
		traceType := traceType

		t.Run(fmt.Sprintf("TestCannonBinRequired-%v", traceType), func(t *testing.T) {
			config := validConfig(traceType)
			config.CannonBin = ""
			require.ErrorIs(t, config.Check(), ErrMissingCannonBin)
		})

		t.Run(fmt.Sprintf("TestCannonServerRequired-%v", traceType), func(t *testing.T) {
			config := validConfig(traceType)
			config.CannonServer = ""
			require.ErrorIs(t, config.Check(), ErrMissingCannonServer)
		})

		t.Run(fmt.Sprintf("TestCannonAbsolutePreStateRequired-%v", traceType), func(t *testing.T) {
			config := validConfig(traceType)
			config.CannonAbsolutePreState = ""
			require.ErrorIs(t, config.Check(), ErrMissingCannonAbsolutePreState)
		})

		t.Run(fmt.Sprintf("TestCannonL2Required-%v", traceType), func(t *testing.T) {
			config := validConfig(traceType)
			config.CannonL2 = ""
			require.ErrorIs(t, config.Check(), ErrMissingCannonL2)
		})

		t.Run(fmt.Sprintf("TestCannonSnapshotFreq-%v", traceType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(traceType)
				cfg.CannonSnapshotFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonSnapshotFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonInfoFreq-%v", traceType), func(t *testing.T) {
			t.Run("MustNotBeZero", func(t *testing.T) {
				cfg := validConfig(traceType)
				cfg.CannonInfoFreq = 0
				require.ErrorIs(t, cfg.Check(), ErrMissingCannonInfoFreq)
			})
		})

		t.Run(fmt.Sprintf("TestCannonNetworkOrRollupConfigRequired-%v", traceType), func(t *testing.T) {
			cfg := validConfig(traceType)
			cfg.CannonNetwork = ""
			cfg.CannonRollupConfigPath = ""
			cfg.CannonL2GenesisPath = "genesis.json"
			require.ErrorIs(t, cfg.Check(), ErrMissingCannonRollupConfig)
		})

		t.Run(fmt.Sprintf("TestCannonNetworkOrL2GenesisRequired-%v", traceType), func(t *testing.T) {
			cfg := validConfig(traceType)
			cfg.CannonNetwork = ""
			cfg.CannonRollupConfigPath = "foo.json"
			cfg.CannonL2GenesisPath = ""
			require.ErrorIs(t, cfg.Check(), ErrMissingCannonL2Genesis)
		})

		t.Run(fmt.Sprintf("TestMustNotSpecifyNetworkAndRollup-%v", traceType), func(t *testing.T) {
			cfg := validConfig(traceType)
			cfg.CannonNetwork = validCannonNetwork
			cfg.CannonRollupConfigPath = "foo.json"
			cfg.CannonL2GenesisPath = ""
			require.ErrorIs(t, cfg.Check(), ErrCannonNetworkAndRollupConfig)
		})

		t.Run(fmt.Sprintf("TestMustNotSpecifyNetworkAndL2Genesis-%v", traceType), func(t *testing.T) {
			cfg := validConfig(traceType)
			cfg.CannonNetwork = validCannonNetwork
			cfg.CannonRollupConfigPath = ""
			cfg.CannonL2GenesisPath = "foo.json"
			require.ErrorIs(t, cfg.Check(), ErrCannonNetworkAndL2Genesis)
		})

		t.Run(fmt.Sprintf("TestNetworkMustBeValid-%v", traceType), func(t *testing.T) {
			cfg := validConfig(traceType)
			cfg.CannonNetwork = "unknown"
			require.ErrorIs(t, cfg.Check(), ErrCannonNetworkUnknown)
		})
	}
}

func TestDatadirRequired(t *testing.T) {
	config := validConfig(TraceTypeAlphabet)
	config.Datadir = ""
	require.ErrorIs(t, config.Check(), ErrMissingDatadir)
}

func TestMaxConcurrency(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		config := validConfig(TraceTypeAlphabet)
		config.MaxConcurrency = 0
		require.ErrorIs(t, config.Check(), ErrMaxConcurrencyZero)
	})

	t.Run("DefaultToNumberOfCPUs", func(t *testing.T) {
		config := validConfig(TraceTypeAlphabet)
		require.EqualValues(t, runtime.NumCPU(), config.MaxConcurrency)
	})
}

func TestHttpPollInterval(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		config := validConfig(TraceTypeAlphabet)
		require.EqualValues(t, DefaultPollInterval, config.PollInterval)
	})
}

func TestRollupRpcRequired(t *testing.T) {
	for _, traceType := range TraceTypes {
		traceType := traceType
		t.Run(traceType.String(), func(t *testing.T) {
			config := validConfig(traceType)
			config.RollupRpc = ""
			require.ErrorIs(t, config.Check(), ErrMissingRollupRpc)
		})
	}
}

func TestRequireConfigForMultipleTraceTypes(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.TraceTypes = []TraceType{TraceTypeCannon, TraceTypeAlphabet}
	// Set all required options and check its valid
	cfg.RollupRpc = validRollupRpc
	require.NoError(t, cfg.Check())

	// Require cannon specific args
	cfg.CannonL2 = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonL2)
	cfg.CannonL2 = validCannonL2

	// Require output cannon specific args
	cfg.RollupRpc = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
}
