package sync

type Config struct {
	// EngineP2PEnabled is true when the EngineQueue can trigger execution engine P2P sync.
	EngineP2PEnabled bool `json:"engine_p2p_enabled"`
}
