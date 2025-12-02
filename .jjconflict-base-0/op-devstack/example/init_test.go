package example

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop(),
		// Logging can be adjusted with filters globally
		presets.WithPkgLogFilter(
			logfilter.DefaultShow( // Random configuration
				stack.KindSelector(stack.L2ProposerKind).Mute(),
				stack.KindSelector(stack.L2BatcherKind).And(logfilter.Level(log.LevelError)).Show(),
				stack.KindSelector(stack.L2CLNodeKind).Mute(),
			),
			// E.g. allow test interactions through while keeping background resource logs quiet
		),
		presets.WithTestLogFilter(logfilter.DefaultMute(logfilter.Level(log.LevelInfo).Show())),
	)
}
