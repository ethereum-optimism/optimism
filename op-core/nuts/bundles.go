package nuts

import (
	_ "embed"
)

// KarstNUTBundleJSON is the embedded Karst NUT bundle.
//
//go:embed bundles/karst_nut_bundle.json
var KarstNUTBundleJSON []byte
