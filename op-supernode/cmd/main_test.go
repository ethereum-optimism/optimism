package main

import (
	"testing"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opsync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVirtualNodeSyncConfigForcesCLSync(t *testing.T) {
	t.Parallel()

	cfg := &opnodecfg.Config{
		Sync: opsync.Config{
			SyncMode:           opsync.ELSync,
			SkipSyncStartCheck: true,
			SyncModeReqResp:    true,
		},
	}

	normalizeVirtualNodeSyncConfig(10, cfg, log.New())

	require.Equal(t, opsync.CLSync, cfg.Sync.SyncMode)
	require.False(t, cfg.Sync.SkipSyncStartCheck)
	require.True(t, cfg.Sync.SyncModeReqResp)
}

func TestNormalizeVirtualNodeSyncConfigLeavesCLSyncUntouched(t *testing.T) {
	t.Parallel()

	cfg := &opnodecfg.Config{
		Sync: opsync.Config{
			SyncMode:           opsync.CLSync,
			SkipSyncStartCheck: true,
			SyncModeReqResp:    true,
		},
	}

	normalizeVirtualNodeSyncConfig(10, cfg, log.New())

	require.Equal(t, opsync.CLSync, cfg.Sync.SyncMode)
	require.True(t, cfg.Sync.SkipSyncStartCheck)
	require.True(t, cfg.Sync.SyncModeReqResp)
}
