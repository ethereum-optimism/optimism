package sequencer

import (
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal(),
		presets.WithCompatibleTypes(compat.SysGo),
		presets.WithLogLevel(slog.LevelDebug),
	)
}
