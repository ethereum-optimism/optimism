package seqwindow

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSimpleInterop(),
		// Short enough that we can run the test,
		// long enough that the batcher can still submit something before we make things expire.
		presets.WithSequencingWindow(10, 30))
}
