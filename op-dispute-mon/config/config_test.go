package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	validL1EthRpc = "http://localhost:8545"
)

func validConfig() Config {
	return NewConfig(validL1EthRpc)
}

func TestL1EthRpcRequired(t *testing.T) {
	config := validConfig()
	config.L1EthRpc = ""
	require.ErrorIs(t, config.Check(), ErrMissingL1EthRPC)
}
