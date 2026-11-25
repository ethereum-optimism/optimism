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
	validSuperRootRpcs      = []string{"http://localhost:8999"}
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

func TestRollupRpcOrSuperRootRpcRequired(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperRootRpcs = nil
	require.ErrorIs(t, config.Check(), ErrMissingRollupAndSuperRootRpc)
	require.EqualError(t, config.Check(), "must specify rollup RPC or super root RPC")
}

func TestRollupRpcNotRequiredWhenSuperRootRpcSet(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperRootRpcs = validSuperRootRpcs
	require.NoError(t, config.Check())
}

func TestSuperRootRpcNotRequiredWhenRollupRpcSet(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = validRollupRpcs
	config.SuperRootRpcs = nil
	require.NoError(t, config.Check())
}

func TestMaxConcurrencyRequired(t *testing.T) {
	config := validConfig()
	config.MaxConcurrency = 0
	require.ErrorIs(t, config.Check(), ErrMissingMaxConcurrency)
}

func TestMultipleSuperRootRpcs(t *testing.T) {
	config := validConfig()
	config.RollupRpcs = nil
	config.SuperRootRpcs = []string{"http://localhost:8999", "http://localhost:9000", "http://localhost:9001"}
	require.NoError(t, config.Check())
}

func TestInteropConfig(t *testing.T) {
	gameFactoryAddr := common.Address{0x42}
	l1RPC := "http://localhost:8545"
	superRootRpcs := []string{"http://localhost:8999", "http://localhost:9000"}

	config := NewInteropConfig(gameFactoryAddr, l1RPC, superRootRpcs)
	require.Equal(t, gameFactoryAddr, config.GameFactoryAddress)
	require.Equal(t, l1RPC, config.L1EthRpc)
	require.Equal(t, superRootRpcs, config.SuperRootRpcs)
	require.Nil(t, config.RollupRpcs)
	require.NoError(t, config.Check())
}

func TestCombinedConfig(t *testing.T) {
	gameFactoryAddr := common.Address{0x42}
	l1RPC := "http://localhost:8545"
	rollupRpcs := []string{"http://localhost:8555"}
	superRootRpcs := []string{"http://localhost:8999"}

	config := NewCombinedConfig(gameFactoryAddr, l1RPC, rollupRpcs, superRootRpcs)
	require.Equal(t, gameFactoryAddr, config.GameFactoryAddress)
	require.Equal(t, l1RPC, config.L1EthRpc)
	require.Equal(t, rollupRpcs, config.RollupRpcs)
	require.Equal(t, superRootRpcs, config.SuperRootRpcs)
	require.NoError(t, config.Check())
}
