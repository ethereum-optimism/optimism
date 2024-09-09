package versions

type StateVersion uint8

const (
	VersionSingleThreaded StateVersion = iota
	VersionMultiThreaded
)
