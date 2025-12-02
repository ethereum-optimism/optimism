package version

import opservice "github.com/ethereum-optimism/optimism/op-service"

var (
	GitCommit = ""
	GitDate   = ""
	Version   = "v0.0.0"
	Meta      = "dev"
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = opservice.FormatVersion(Version, GitCommit, GitDate, Meta)
