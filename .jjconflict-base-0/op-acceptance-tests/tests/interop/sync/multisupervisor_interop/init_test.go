package sync

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMultiSupervisorInterop(),
		presets.WithLogFilter(logfilter.DefaultMute(
			stack.KindSelector(stack.SupervisorKind).And(logfilter.Level(log.LevelInfo)).Show(),
			logfilter.Level(log.LevelError).Show(),
		)))
}
