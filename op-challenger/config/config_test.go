package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	validL1EthRpc           = "http://localhost:8545"
	validGameAddress        = common.HexToAddress("0x7bdd3b028C4796eF0EAf07d11394d0d9d8c24139")
	validAlphabetTrace      = "abcdefgh"
	agreeWithProposedOutput = true
	gameDepth               = 4
)

func validConfig() Config {
	cfg := NewConfig(validL1EthRpc, validGameAddress, validAlphabetTrace, agreeWithProposedOutput, gameDepth)
	return cfg
}

// TestValidConfigIsValid checks that the config provided by validConfig is actually valid
func TestValidConfigIsValid(t *testing.T) {
	err := validConfig().Check()
	require.NoError(t, err)
}

func TestTxMgrConfig(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		config := validConfig()
		config.TxMgrConfig = txmgr.CLIConfig{}
		err := config.Check()
		require.Equal(t, err.Error(), "must provide a L1 RPC url")
	})
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig()
	config.L1EthRpc = ""
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingL1EthRPC)
}

func TestGameAddressRequired(t *testing.T) {
	config := validConfig()
	config.GameAddress = common.Address{}
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingGameAddress)
}

func TestAlphabetTraceRequired(t *testing.T) {
	config := validConfig()
	config.AlphabetTrace = ""
	err := config.Check()
	require.ErrorIs(t, err, ErrMissingAlphabetTrace)
}
