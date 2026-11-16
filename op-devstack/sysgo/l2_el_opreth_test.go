package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpRethWithEthProofWindow(t *testing.T) {
	cfg := DefaultOpRethConfig()
	OpRethWithEthProofWindow(1_000_000).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, []string{"--rpc.eth-proof-window=1000000"}, cfg.ExtraArgs)
}
