package conductor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestConductorMarksStoppedSequencerUnhealthy(gt *testing.T) {
	t := devtest.SerialT(gt)
	sysgo.SkipOnKonaNode(t, "not supported")

	sys := presets.NewMinimalWithConductors(t,
		presets.WithConductorHealthCheck(1, 3, 3600),
	)
	conductor := conductorForChain(t, sys.ConductorSets, sys.L2Chain.Escape().ChainID())
	ctx := t.Ctx()

	require.Eventually(t, func() bool {
		return conductorHealthy(ctx, conductor)
	}, 30*time.Second, time.Second, "conductor should start healthy")

	sys.L2CL.StopSequencer()

	require.Eventually(t, func() bool {
		return !conductorHealthy(ctx, conductor)
	}, 30*time.Second, time.Second, "stopped sequencer should become unhealthy")
}

func conductorForChain(t devtest.T, conductorSets map[eth.ChainID]dsl.ConductorSet, chainID eth.ChainID) *dsl.Conductor {
	conductors := conductorSets[chainID]
	require.NotEmpty(t, conductors, "expected conductors for chain %s", chainID)
	return conductors[0]
}

func conductorHealthy(ctx context.Context, conductor *dsl.Conductor) bool {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	healthy, err := conductor.Escape().RpcAPI().SequencerHealthy(callCtx)
	return err == nil && healthy
}
