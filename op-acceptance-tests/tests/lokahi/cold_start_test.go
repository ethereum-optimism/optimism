package lokahi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// coldStartHistory is how many blocks each chain's safe head passes before lokahi loses its
// store, so that the cold start has history to verify again.
const coldStartHistory = 100

// TestLokahiColdStart checks that lokahi started with an empty store against chains with history
// verifies them again from activation up to where it was, and that a restart partway resumes from
// the store. Until lokahi caught up it answers no heads, so no node's safe head moves back
// meanwhile. It logs the rate of each leg.
func TestLokahiColdStart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2Lokahi(t)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL}
	rollupCfg := sys.L2A.Escape().RollupConfig()
	t.Require().Equal(rollupCfg.BlockTime, sys.L2B.Escape().RollupConfig().BlockTime, "both chains have one block time")
	activation := rollupCfg.Genesis.L2Time

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, safeReachesWithinLocalSafe(t, node, coldStartHistory, 10*time.Minute))
	}
	dsl.CheckAll(t, checks...)
	target := verifiedHeads(t, sys.Lokahi, nodes)[nodes[0]]

	// While lokahi verifies again it answers no heads, so the safe heads hold.
	watch := watchSafeHeads(t, nodes)
	sys.Lokahi.Stop()
	sys.Lokahi.WipeData()
	sys.Lokahi.Start()
	start := time.Now()

	halfway := activation + (target.Timestamp-activation)/2
	mid, midAt := waitVerifiedTimestamp(t, sys.Lokahi, nodes[0], halfway)
	logRate(t, "cold start to halfway", activation, mid.Timestamp, midAt.Sub(start), rollupCfg.BlockTime, len(nodes))

	// A restart partway resumes from the store rather than from activation.
	sys.Lokahi.Stop()
	sys.Lokahi.Start()
	restart := time.Now()
	resumed := firstProgress(t, sys.Lokahi, nodes[0])
	t.Require().Greater(resumed, mid.Timestamp, "lokahi resumes from the store")

	end, endAt := waitVerifiedTimestamp(t, sys.Lokahi, nodes[0], target.Timestamp)
	logRate(t, "resumed to the head before the cold start", resumed-1, end.Timestamp, endAt.Sub(restart), rollupCfg.BlockTime, len(nodes))
	logRate(t, "whole cold start, with the restart", activation, end.Timestamp, endAt.Sub(start), rollupCfg.BlockTime, len(nodes))
	t.Require().NoError(watch.stop(), "no safe head may move back while lokahi catches up")

	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 60))
	}
	dsl.CheckAll(t, checks...)
	logHeads(t, "after the cold start", nodes)
}

// waitVerifiedTimestamp polls lokahi until node's chain is verified at timestamp, also while
// lokahi has not caught up yet, returning the verified timestamp and when it was seen.
func waitVerifiedTimestamp(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode, timestamp uint64) (verifierHead, time.Time) {
	ctx, cancel := context.WithTimeout(t.Ctx(), 5*time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if next, _, err := lokahiProgress(ctx, client, node); err == nil && next > timestamp {
			return verifierHead{Timestamp: next - 1, Source: "verified"}, time.Now()
		}
		select {
		case <-ctx.Done():
			t.Require().NoError(ctx.Err(), "lokahi did not verify timestamp %d", timestamp)
		case <-ticker.C:
		}
	}
}

// logRate logs how fast lokahi verified the timestamps after from up to to.
func logRate(t devtest.T, leg string, from, to uint64, elapsed time.Duration, blockTime uint64, chains int) {
	timestamps := float64(to - from)
	perSecond := timestamps / elapsed.Seconds()
	t.Logger().Info("Lokahi verification rate", "leg", leg, "timestamps", to-from, "elapsed", elapsed,
		"timestamps_per_s", fmt.Sprintf("%.1f", perSecond),
		"blocks_per_s", fmt.Sprintf("%.1f", perSecond*float64(chains)/float64(blockTime)))
}

// firstProgress returns the next timestamp lokahi verifies for node's chain, as soon as it
// answers.
func firstProgress(t devtest.T, lokahi *sysgo.Lokahi, node *dsl.L2CLNode) uint64 {
	ctx, cancel := context.WithTimeout(t.Ctx(), time.Minute)
	defer cancel()
	client, err := rpc.DialContext(ctx, lokahi.UserRPC())
	t.Require().NoError(err, "dial lokahi")
	defer client.Close()
	var next uint64
	err = wait.For(ctx, 20*time.Millisecond, func() (bool, error) {
		n, _, err := lokahiProgress(ctx, client, node)
		next = n
		return err == nil && n > 0, nil
	})
	t.Require().NoError(err, "lokahi did not answer")
	return next
}
