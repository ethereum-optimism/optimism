package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

var (
	validL1EthRpc           = "http://localhost:8545"
	validGameFactoryAddress = common.Address{0x23}
)

func validConfig() Config {
	return NewConfig(validGameFactoryAddress, validL1EthRpc)
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
