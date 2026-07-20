package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestResolveL2CLNodeConfig(t *testing.T) {
	dt := devtest.SerialT(t)
	applies := 0
	opt := L2CLOptionFn(func(_ devtest.T, target ComponentTarget, cfg *L2CLConfig) {
		applies++
		// Start-config fields are set before options run, so options can
		// observe and override them.
		require.True(t, cfg.NoDiscovery)
		require.Equal(t, "verifier", target.Name)
		cfg.FollowSource = "http://proxy.invalid"
	})
	startCfg := l2CLNodeStartConfig{
		Key:            "verifier",
		NoDiscovery:    true,
		EnableReqResp:  true,
		L2FollowSource: "http://origin.invalid",
		L2CLOptions:    []L2CLOption{nil, opt},
	}
	cfg := resolveL2CLNodeConfig(dt, NewComponentTarget("verifier", eth.ChainIDFromUInt64(901)), startCfg)
	require.Equal(t, 1, applies, "options must be applied exactly once")
	require.Equal(t, "http://proxy.invalid", cfg.FollowSource, "option rewrite of the follow source must win")
	require.True(t, cfg.EnableReqRespSync)
	require.False(t, cfg.IsSequencer)
}
