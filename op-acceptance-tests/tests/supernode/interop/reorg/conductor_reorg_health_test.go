package reorg

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestSupernodeInteropInvalidMessageReorgKeepsConductorHealthy(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInteropWithConductors(t, 0,
		presets.WithConductorHealthCheck(5, 5, 3600),
		presets.WithConductorHealthCheckMinPeerCount(2),
	)

	ctx := t.Ctx()
	l2BConductor := conductorForChain(t, sys.ConductorSets, sys.L2B.Escape().ChainID())
	require.Eventually(t, func() bool {
		return conductorHealthy(ctx, l2BConductor)
	}, 30*time.Second, time.Second, "chain B conductor should start healthy")

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()

	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	paused := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", paused)

	rng := rand.New(rand.NewSource(12345))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()

	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)

	require.Eventually(t, func() bool {
		return sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	stopHealthWatch := watchConductorHealth(ctx, l2BConductor)
	sys.Supernode.ResumeInterop()
	require.Eventually(t, func() bool {
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			if !errors.Is(eth.MaybeAsNotFoundErr(err), ethereum.NotFound) {
				t.Logger().Warn("unexpected error checking block",
					"block_number", invalidBlockNumber,
					"err", err,
				)
			}
			return false
		}
		return currentBlock.Hash != invalidBlockHash
	}, 60*time.Second, time.Second, "reset should replace the invalid block")

	sys.Supernode.AwaitValidatedTimestamp(invalidBlockTimestamp)
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)
	require.NoError(t, stopHealthWatch(), "chain B conductor should stay healthy during invalid-message reorg")
	require.True(t, conductorHealthy(ctx, l2BConductor), "chain B conductor should remain healthy after invalid-message reorg")
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

func watchConductorHealth(ctx context.Context, conductor *dsl.Conductor) func() error {
	watchCtx, cancel := context.WithCancel(ctx)
	failures := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				callCtx, callCancel := context.WithTimeout(watchCtx, 5*time.Second)
				healthy, err := conductor.Escape().RpcAPI().SequencerHealthy(callCtx)
				callCancel()
				if watchCtx.Err() != nil {
					return
				}
				if err != nil {
					select {
					case failures <- fmt.Errorf("fetch conductor health: %w", err):
					default:
					}
					return
				}
				if !healthy {
					select {
					case failures <- errors.New("conductor reported sequencer unhealthy"):
					default:
					}
					return
				}
			}
		}
	}()
	return func() error {
		cancel()
		<-done
		select {
		case err := <-failures:
			return err
		default:
			return nil
		}
	}
}
