package reorg

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type rpcGatePollStats struct {
	success        atomic.Int64
	methodNotFound atomic.Int64
	notFound       atomic.Int64
	unavailable    atomic.Int64
}

const maxSyncStatusPollsInFlight = 4

func (s *rpcGatePollStats) total() int64 {
	return s.success.Load() +
		s.methodNotFound.Load() +
		s.notFound.Load() +
		s.unavailable.Load()
}

func (s *rpcGatePollStats) record(err error) {
	if err == nil {
		s.success.Add(1)
		return
	}

	var rpcErr gethrpc.Error
	if errors.As(err, &rpcErr) && rpcErr.ErrorCode() == int(eth.MethodNotFound) {
		s.methodNotFound.Add(1)
		return
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "404") || strings.Contains(msg, "not found"):
		s.notFound.Add(1)
	case strings.Contains(msg, "503") || strings.Contains(msg, "service unavailable") || errors.Is(err, context.DeadlineExceeded):
		s.unavailable.Add(1)
	}
}

func startSyncStatusPoller(ctx context.Context, t devtest.T, rpcCl client.RPC) (*rpcGatePollStats, <-chan struct{}) {
	stats := new(rpcGatePollStats)
	done := make(chan struct{})

	go func() {
		defer close(done)

		var wg sync.WaitGroup
		defer wg.Wait()
		inFlight := make(chan struct{}, maxSyncStatusPollsInFlight)

		poll := func() {
			select {
			case inFlight <- struct{}{}:
			case <-ctx.Done():
				return
			default:
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					<-inFlight
				}()

				callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				var out eth.SyncStatus
				err := rpcCl.CallContext(callCtx, &out, "optimism_syncStatus")
				if err != nil && ctx.Err() != nil {
					return
				}
				stats.record(err)
				if err != nil {
					t.Logger().Warn("supernode sync status poll failed", "err", err)
				}
			}()
		}

		poll()
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	}()

	return stats, done
}

// TestSupernodeInteropRPCGatedDuringReorg proves patient callers do not see
// method-not-found or route-missing responses while supernode restarts chain B's
// virtual node during interop reorg recovery.
func TestSupernodeInteropRPCGatedDuringReorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	ctx := t.Ctx()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()

	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	paused := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", paused)

	rng := rand.New(rand.NewSource(12345))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)

	t.Logger().Info("initiating message sent on chain A",
		"block", initMsg.BlockNumber(),
		"hash", initMsg.BlockHash(),
	)

	sys.L2B.WaitForBlock()

	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)
	t.Logger().Info("invalid executing message sent on chain B",
		"block", invalidBlockNumber,
		"hash", invalidBlockHash,
		"timestamp", invalidBlockTimestamp,
	)

	require.Eventually(t, func() bool {
		return sys.L2BCL.SyncStatus().LocalSafeL2.Number >= invalidBlockNumber
	}, 60*time.Second, time.Second, "invalid block should become locally safe")

	rpcURL := sys.L2BCL.Escape().UserRPC()
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), rpcURL, client.WithLazyDial(), client.WithCallTimeout(30*time.Second))
	require.NoError(t, err)
	defer rpcCl.Close()

	pollCtx, cancelPoll := context.WithCancel(ctx)
	stats, pollDone := startSyncStatusPoller(pollCtx, t, rpcCl)
	require.Eventually(t, func() bool {
		return stats.success.Load() > 0
	}, 10*time.Second, 200*time.Millisecond, "sync status poller should succeed before reorg")

	sys.Supernode.ResumeInterop()
	require.Eventually(t, func() bool {
		currentBlock, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, invalidBlockNumber)
		if err != nil {
			if errors.Is(eth.MaybeAsNotFoundErr(err), ethereum.NotFound) {
				t.Logger().Info("reset detected: block no longer exists",
					"block_number", invalidBlockNumber,
				)
			} else {
				t.Logger().Warn("unexpected error checking block",
					"block_number", invalidBlockNumber,
					"err", err,
				)
			}
		} else if currentBlock.Hash != invalidBlockHash {
			t.Logger().Info("reset detected: block hash changed",
				"block_number", invalidBlockNumber,
				"old_hash", invalidBlockHash,
				"new_hash", currentBlock.Hash,
			)
			return true
		}
		return false
	}, 60*time.Second, time.Second, "reset should be detected")

	sys.Supernode.AwaitValidatedTimestamp(invalidBlockTimestamp)
	sys.L2ELB.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)

	bruce := sys.FunderB.NewFundedEOA(eth.OneEther)
	tx := bruce.Transfer(alice.Address(), eth.OneHundredthEther)
	txBlock := bigs.Uint64Strict(tx.Included.Value().BlockNumber)
	sys.L2ELB.AssertTxInBlock(txBlock, tx.Included.Value().TxHash)

	txTimestamp := sys.L2B.TimestampForBlockNum(txBlock)
	sys.Supernode.AwaitValidatedTimestamp(txTimestamp)
	sys.L2ELB.AssertTxInBlock(txBlock, tx.Included.Value().TxHash)

	postReorgSuccesses := stats.success.Load()
	require.Eventually(t, func() bool {
		return stats.success.Load() > postReorgSuccesses
	}, 10*time.Second, 200*time.Millisecond, "sync status poller should succeed after reorg")

	cancelPoll()
	<-pollDone

	require.Greater(t, stats.total(), int64(0), "sync status poller should observe RPC calls")
	require.Zero(t, stats.methodNotFound.Load(), "supernode route returned JSON-RPC method-not-found during reorg")
	require.Zero(t, stats.notFound.Load(), "supernode route returned 404 during reorg")
}
