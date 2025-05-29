package example

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/logfilter"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleInterop(),
		// Logging can be adjusted with filters globally
		presets.WithLogFilters(
			stack.KindLogFilter(stack.L2ProposerKind, logfilter.Mute()),
			stack.KindLogFilter(stack.L2BatcherKind, logfilter.Minimum(log.LevelError)),
			stack.KindLogFilter(stack.L2CLNodeKind, logfilter.Add(3)),
		),
		// E.g. elevate the logs of your test interactions, while keeping background resource logs quiet
		presets.WithTestLogFilters(logfilter.Add(4)),
	)
}
