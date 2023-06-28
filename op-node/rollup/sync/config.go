package sync

type Config struct {
	// EngineP2PEnabled is true when the EngineQueue can trigger execution engine P2P sync.
	EngineP2PEnabled bool `json:"engine_p2p_enabled"`
	// SkipSanityCheck is true when the EngineQueue does not do sanity check on pipeline reset.
	SkipSanityCheck bool `json:"skip_sanity_check"`
}
