package sync

type Config struct {
	// EngineSync is true when the EngineQueue can trigger execution engine P2P sync.
	EngineSync bool `json:"engine_sync"`
	// SkipSyncStartCheck skip the sanity check of consistency of L1 origins of the unsafe L2 blocks when determining the sync-starting point. This defers the L1-origin verification, and is recommended to use in when utilizing l2.engine-sync
	SkipSyncStartCheck bool `json:"skip_sync_start_check"`
}
