package opcm

import "embed"

//go:embed standard-versions.toml
var StandardVersionsData string

var _ embed.FS
