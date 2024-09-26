package opcm

import "embed"

//go:embed standard-versions-mainnet.toml
var StandardVersionsMainnetData string

//go:embed standard-versions-sepolia.toml
var StandardVersionsSepoliaData string

var _ embed.FS
