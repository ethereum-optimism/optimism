package nuts

import (
	_ "embed"
)

// KarstNUTBundleJSON is the embedded Karst NUT bundle.
//
//go:embed bundles/karst_nut_bundle.json
var KarstNUTBundleJSON []byte

// InteropNUTBundleJSON is the embedded Interop NUT bundle.
//
//go:embed bundles/interop_nut_bundle.json
var InteropNUTBundleJSON []byte
