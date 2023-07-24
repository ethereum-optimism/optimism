package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	validL1EthRpc           = "http://localhost:8545"
	validGameAddress        = common.HexToAddress("0x7bdd3b028C4796eF0EAf07d11394d0d9d8c24139")
	validAlphabetTrace      = "abcdefgh"
	validCannonDatadir      = "/tmp/cannon"
	agreeWithProposedOutput = true
	gameDepth               = 4
)

func validConfig(traceType flags.TraceType) Config {
	cfg := NewConfig(validL1EthRpc, validGameAddress, traceType, validAlphabetTrace, validCannonDatadir, agreeWithProposedOutput, gameDepth)
	return cfg
}

// TestValidConfigIsValid checks that the config provided by validConfig is actually valid
func TestValidConfigIsValid(t *testing.T) {
	err := validConfig(flags.TraceTypeCannon).Check()
	require.NoError(t, err)
}

func TestTxMgrConfig(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		config := validConfig(flags.TraceTypeCannon)
		config.TxMgrConfig = txmgr.CLIConfig{}
		require.Equal(t, config.Check().Error(), "must provide a L1 RPC url")
	})
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig(flags.TraceTypeCannon)
	config.L1EthRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1EthRPC)
	config.L1EthRpc = validL1EthRpc
	require.NoError(t, config.Check())
}

func TestGameAddressRequired(t *testing.T) {
	config := validConfig(flags.TraceTypeCannon)
	config.GameAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameAddress)
	config.GameAddress = validGameAddress
	require.NoError(t, config.Check())
}

func TestAlphabetTraceRequired(t *testing.T) {
	config := validConfig(flags.TraceTypeAlphabet)
	config.AlphabetTrace = ""
	require.ErrorIs(t, config.Check(), ErrMissingAlphabetTrace)
	config.AlphabetTrace = validAlphabetTrace
	require.NoError(t, config.Check())
}

func TestCannonTraceRequired(t *testing.T) {
	config := validConfig(flags.TraceTypeCannon)
	config.CannonDatadir = ""
	require.ErrorIs(t, config.Check(), ErrMissingCannonDatadir)
	config.CannonDatadir = validCannonDatadir
	require.NoError(t, config.Check())
}
