package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

var (
	validL1EthRpc           = "http://localhost:8545"
	validGameFactoryAddress = common.Address{0x23}
	validRollupRpcs         = []string{"http://localhost:8555"}
	validSuperNodeRpcs      = []string{"http://localhost:8999"}
)

func validConfig() Config {
	return NewConfig(validGameFactoryAddress, validL1EthRpc, validRollupRpcs)
}

func TestValidConfigIsValid(t *testing.T) {
	require.NoError(t, validConfig().Check())
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig()
	config.L1EthRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1EthRPC)
}

func TestGameFactoryAddressRequired(t *testing.T) {
	config := validConfig()
	config.GameFactoryAddress = common.Address{}
	require.ErrorIs(t, config.Check(), ErrMissingGameFactoryAddress)
}

func TestRollupRpcOrSuperNodeRpcRequired(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperNodeRpcs = nil
	require.ErrorIs(t, config.Check(), ErrMissingRollupAndSuperNodeRpc)
}

func TestRollupRpcNotRequiredWhenSuperNodeRpcSet(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperNodeRpcs = validSuperNodeRpcs
	require.NoError(t, config.Check())
}

func TestSuperNodeRpcNotRequiredWhenRollupRpcSet(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = validRollupRpcs
	config.SuperNodeRpcs = nil
	require.NoError(t, config.Check())
}

func TestMaxConcurrencyRequired(t *testing.T) {
	config := validConfig()
	config.MaxConcurrency = 0
	require.ErrorIs(t, config.Check(), ErrMissingMaxConcurrency)
}

func TestMultipleSuperNodeRpcs(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperNodeRpcs = []string{"http://localhost:8999", "http://localhost:9000", "http://localhost:9001"}
	require.NoError(t, config.Check())
}

func TestInteropConfig(t *testing.T) {
	gameFactoryAddr := common.Address{0x42}
	l1RPC := "http://localhost:8545"
	superNodeRpcs := []string{"http://localhost:8999", "http://localhost:9000"}

	config := NewInteropConfig(gameFactoryAddr, l1RPC, superNodeRpcs)
	require.Equal(t, gameFactoryAddr, config.GameFactoryAddress)
	require.Equal(t, l1RPC, config.L1EthRpc)
	require.Equal(t, superNodeRpcs, config.SuperNodeRpcs)
	require.Nil(t, config.RollupRpcs)
	require.NoError(t, config.Check())
}

func TestCombinedConfig(t *testing.T) {
	gameFactoryAddr := common.Address{0x42}
	l1RPC := "http://localhost:8545"
	rollupRpcs := []string{"http://localhost:8555"}
	superNodeRpcs := []string{"http://localhost:8999"}

	config := NewCombinedConfig(gameFactoryAddr, l1RPC, rollupRpcs, superNodeRpcs)
	require.Equal(t, gameFactoryAddr, config.GameFactoryAddress)
	require.Equal(t, l1RPC, config.L1EthRpc)
	require.Equal(t, rollupRpcs, config.RollupRpcs)
	require.Equal(t, superNodeRpcs, config.SuperNodeRpcs)
	require.NoError(t, config.Check())
}
