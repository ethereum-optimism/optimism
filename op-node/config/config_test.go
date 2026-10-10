package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestCheckSafeDBCompatible(t *testing.T) {
	followSource := sync.Config{L2FollowSourceEndpoint: "http://localhost:9545"}
	tests := []struct {
		name       string
		safeDBPath string
		syncCfg    sync.Config
		wantErr    error
	}{
		{name: "neither"},
		{name: "safedb only", safeDBPath: "/safedb"},
		{name: "follow source only", syncCfg: followSource},
		{name: "both", safeDBPath: "/safedb", syncCfg: followSource, wantErr: ErrSafeDBWithFollowSource},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, checkSafeDBCompatible(tt.safeDBPath, &tt.syncCfg), tt.wantErr)
		})
	}
}

func TestConfigCheck_RejectsSafeDBWithFollowSource(t *testing.T) {
	cfg := &Config{
		SafeDBPath: "/safedb",
		Sync:       sync.Config{L2FollowSourceEndpoint: "http://localhost:9545"},
	}
	require.ErrorIs(t, cfg.Check(), ErrSafeDBWithFollowSource)
}
