package config

import (
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	validL1EthRpc              = "http://localhost:8545"
	validGameFactoryAddress    = common.Address{0x23}
	validAlphabetTrace         = "abcdefgh"
	validCannonBin             = "./bin/cannon"
	validCannonOpProgramBin    = "./bin/op-program"
	validCannonNetwork         = "mainnet"
	validCannonAbsolutPreState = "pre.json"
	validDatadir               = "/tmp/data"
	validCannonL2              = "http://localhost:9545"
	validRollupRpc             = "http://localhost:8555"
)

func validConfig(traceType TraceType) Config {
	cfg := NewConfig(validGameFactoryAddress, validL1EthRpc, validDatadir, traceType)
	switch traceType {
	case TraceTypeAlphabet:
		cfg.AlphabetTrace = validAlphabetTrace
	case TraceTypeCannon, TraceTypeOutputCannon:
		cfg.CannonBin = validCannonBin
		cfg.CannonServer = validCannonOpProgramBin
		cfg.CannonAbsolutePreState = validCannonAbsolutPreState
		cfg.CannonL2 = validCannonL2
		cfg.CannonNetwork = validCannonNetwork
	}
	if traceType == TraceTypeOutputCannon || traceType == TraceTypeOutputAlphabet {
		cfg.RollupRpc = validRollupRpc
	}
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

func TestGameFactoryAddressRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.GameFactoryAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameFactoryAddress)
}

func TestGameAllowlistNotRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.GameAllowlist = []common.Address{}
	require.NoError(t, config.Check())
}

func TestAlphabetTraceRequired(t *testing.T) {
	config := validConfig(TraceTypeAlphabet)
	config.AlphabetTrace = ""
	require.ErrorIs(t, config.Check(), ErrMissingAlphabetTrace)
}

func TestAlphabetTraceNotRequiredForOutputAlphabet(t *testing.T) {
	config := validConfig(TraceTypeOutputAlphabet)
	config.AlphabetTrace = ""
	require.NoError(t, config.Check())
}

func TestCannonBinRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonBin = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonBin)
}

func TestCannonServerRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonServer = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonServer)
}

func TestCannonAbsolutePreStateRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonAbsolutePreState = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonAbsolutePreState)
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

func TestRollupRpcRequired_OutputCannon(t *testing.T) {
	config := validConfig(TraceTypeOutputCannon)
	config.RollupRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingRollupRpc)
}

func TestRollupRpcRequired_OutputAlphabet(t *testing.T) {
	config := validConfig(TraceTypeOutputAlphabet)
	config.RollupRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingRollupRpc)
}

func TestCannonL2Required(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonL2 = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonL2)
}

func TestCannonSnapshotFreq(t *testing.T) {
	t.Run("MustNotBeZero", func(t *testing.T) {
		cfg := validConfig(TraceTypeCannon)
		cfg.CannonSnapshotFreq = 0
		require.ErrorIs(t, cfg.Check(), ErrMissingCannonSnapshotFreq)
	})
}

func TestCannonInfoFreq(t *testing.T) {
	t.Run("MustNotBeZero", func(t *testing.T) {
		cfg := validConfig(TraceTypeCannon)
		cfg.CannonInfoFreq = 0
		require.ErrorIs(t, cfg.Check(), ErrMissingCannonInfoFreq)
	})
}

func TestCannonNetworkOrRollupConfigRequired(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.CannonNetwork = ""
	cfg.CannonRollupConfigPath = ""
	cfg.CannonL2GenesisPath = "genesis.json"
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonRollupConfig)
}

func TestCannonNetworkOrL2GenesisRequired(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.CannonNetwork = ""
	cfg.CannonRollupConfigPath = "foo.json"
	cfg.CannonL2GenesisPath = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonL2Genesis)
}

func TestMustNotSpecifyNetworkAndRollup(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.CannonNetwork = validCannonNetwork
	cfg.CannonRollupConfigPath = "foo.json"
	cfg.CannonL2GenesisPath = ""
	require.ErrorIs(t, cfg.Check(), ErrCannonNetworkAndRollupConfig)
}

func TestMustNotSpecifyNetworkAndL2Genesis(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.CannonNetwork = validCannonNetwork
	cfg.CannonRollupConfigPath = ""
	cfg.CannonL2GenesisPath = "foo.json"
	require.ErrorIs(t, cfg.Check(), ErrCannonNetworkAndL2Genesis)
}

func TestNetworkMustBeValid(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.CannonNetwork = "unknown"
	require.ErrorIs(t, cfg.Check(), ErrCannonNetworkUnknown)
}

func TestRequireConfigForMultipleTraceTypes(t *testing.T) {
	cfg := validConfig(TraceTypeCannon)
	cfg.TraceTypes = []TraceType{TraceTypeCannon, TraceTypeAlphabet, TraceTypeOutputCannon}
	// Set all required options and check its valid
	cfg.RollupRpc = validRollupRpc
	cfg.AlphabetTrace = validAlphabetTrace
	require.NoError(t, cfg.Check())

	// Require cannon specific args
	cfg.CannonL2 = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingCannonL2)
	cfg.CannonL2 = validCannonL2

	// Require alphabet specific args
	cfg.AlphabetTrace = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingAlphabetTrace)
	cfg.AlphabetTrace = validAlphabetTrace

	// Require output cannon specific args
	cfg.RollupRpc = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingRollupRpc)
}
