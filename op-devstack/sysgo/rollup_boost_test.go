package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/stretchr/testify/require"
)

func TestRollupBoostLaunchSpecUsesEphemeralRPCPort(t *testing.T) {
	cfg := DefaultRollupBoostConfig()
	cfg.ExtraArgs = []string{"--rpc-port=12345"}
	args, _ := cfg.LaunchSpec(devtest.SerialT(t))

	require.NotContains(t, args, "--rpc-port=12345")
	require.Equal(t, "--rpc-port=0", args[len(args)-1])
}
