package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	validL1EthRpc              = "http://localhost:8545"
	validGameAddress           = common.HexToAddress("0x7bdd3b028C4796eF0EAf07d11394d0d9d8c24139")
	validAlphabetTrace         = "abcdefgh"
	validCannonBin             = "./bin/cannon"
	validCannonAbsolutPreState = "pre.json"
	validCannonDatadir         = "/tmp/cannon"
	validCannonL2              = "http://localhost:9545"
	agreeWithProposedOutput    = true
	gameDepth                  = 4
)

func validConfig(traceType TraceType) Config {
	cfg := NewConfig(validL1EthRpc, validGameAddress, traceType, agreeWithProposedOutput, gameDepth)
	switch traceType {
	case TraceTypeAlphabet:
		cfg.AlphabetTrace = validAlphabetTrace
	case TraceTypeCannon:
		cfg.CannonBin = validCannonBin
		cfg.CannonAbsolutePreState = validCannonAbsolutPreState
		cfg.CannonDatadir = validCannonDatadir
		cfg.CannonL2 = validCannonL2
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

func TestGameAddressRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.GameAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameAddress)
}

func TestAlphabetTraceRequired(t *testing.T) {
	config := validConfig(TraceTypeAlphabet)
	config.AlphabetTrace = ""
	require.ErrorIs(t, config.Check(), ErrMissingAlphabetTrace)
}

func TestCannonBinRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonBin = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonBin)
}

func TestCannonAbsolutePreStateRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonAbsolutePreState = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonAbsolutePreState)
}

func TestCannonDatadirRequired(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonDatadir = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonDatadir)
}

func TestCannonL2Required(t *testing.T) {
	config := validConfig(TraceTypeCannon)
	config.CannonL2 = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonL2)
}
