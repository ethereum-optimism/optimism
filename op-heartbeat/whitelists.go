package op_heartbeat

var AllowedChainIDs = map[uint64]bool{
	420: true,
	902: true,
	10:  true,
}

var AllowedVersions = map[string]bool{
	"":                          true,
	"v0.1.0-beta.1":             true,
	"v0.1.0-goerli-rehearsal.1": true,
}
