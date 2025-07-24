package sync

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-devstack/presets"
	"github.com/HashKeyChain/verse/op-devstack/stack"
	"github.com/HashKeyChain/verse/op-service/log/logfilter"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMultiSupervisorInterop(),
		presets.WithLogFilter(logfilter.DefaultMute(
			stack.KindSelector(stack.SupervisorKind).And(logfilter.Level(log.LevelInfo)).Show(),
			logfilter.Level(log.LevelError).Show(),
		)))
}
