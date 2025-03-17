package systemgo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSystem(t *testing.T) {
	ids, opt := DefaultInteropSystem(ContractPaths{
		FoundryArtifacts: "../../packages/contracts-bedrock/forge-artifacts",
		SourceMap:        "../../packages/contracts-bedrock",
	})
	logger := testlog.Logger(t, log.LevelInfo)
	orch := &Orchestrator{
		t: t,
	}
	// TODO(#15026): known issue, setup needs helper functions / polish
	setup := &system2.Setup{
		Ctx:          context.Background(),
		Log:          logger,
		T:            t,
		Require:      require.New(t),
		System:       nil,
		Orchestrator: orch,
	}
	setup.System = system2.NewSystem(system2.SystemConfig{
		CommonConfig: setup.CommonConfig(),
	})
	opt(setup)

	seqA := setup.System.L2Network(ids.L2A).L2CLNode(ids.L2ACL)
	seqB := setup.System.L2Network(ids.L2B).L2CLNode(ids.L2BCL)
	for i := 0; i < 20*2+10; i++ {
		time.Sleep(time.Second * 2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		statusA, err := seqA.RollupAPI().SyncStatus(ctx)
		require.NoError(t, err)
		statusB, err := seqB.RollupAPI().SyncStatus(ctx)
		require.NoError(t, err)
		cancel()
		logger.Info("chain A", "tip", statusA.UnsafeL2)
		logger.Info("chain B", "tip", statusB.UnsafeL2)

		if statusA.UnsafeL2.Number > 20 && statusB.UnsafeL2.Number > 20 {
			return
		}
	}
	t.Fatal("Expected to reach block 20 on both chains")
}
