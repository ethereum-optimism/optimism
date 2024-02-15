package superchain

type SuperchainConfig struct {
	// Simply take in the node addresses directly for now. We
	// may want to expand and accept more general endpoint
	// configuration like RateLimits, MaxConcurrency, etc

	// If we want op-node to use this as a library rather than an
	// external service, configuration should be extensible to support
	// plugging in configured client tooling that's ready to use.
	L2NodeAddr      string
	PeerL2NodeAddrs map[uint64]string
}
