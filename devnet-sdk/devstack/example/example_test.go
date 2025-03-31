package example

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m)
}

func TestExample1(t *testing.T) {
	preset := presets.NewSimpleInterop(t)
	preset.Log.Info("hello world")

	qapi := preset.Supervisor.QueryAPI()

	blocks := uint64(10)
	// wait for this many blocks, with some margin for delays
	for i := uint64(0); i < blocks*2+10; i++ {
		time.Sleep(time.Second * 2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		superStatus, err := qapi.SyncStatus(ctx)
		cancel()
		require.NoError(t, err)
		minHeight := ^uint64(0)
		for chID, status := range superStatus.Chains {
			preset.Log.Info("status", "chain", chID, "unsafe", status.LocalUnsafe)
			minHeight = min(status.LocalUnsafe.Number, minHeight)
		}
		// Once every chain has been synced, stop
		if minHeight > blocks {
			return
		}
	}
	t.Fatalf("Expected to reach block %d on both chains", blocks)
}

// TODO(#15138): adjust sysgo / syskt to be graceful
//  when things already exist
//  (just set the shim with existing orchestrator-managed service)
//func TestExample2(t *testing.T) {
//	preset := presets.NewSimpleInterop(t)
//	preset.Log.Info("foobar 123")
//}
