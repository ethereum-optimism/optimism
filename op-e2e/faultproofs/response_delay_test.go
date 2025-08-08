package faultproofs

import (
	"context"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestChallengerResponseDelay tests that the challenger respects the configured response delay
// This is an integration test that verifies that the actual delay functionality works end-to-end
func TestChallengerResponseDelay(t *testing.T) {
	op_e2e.InitParallel(t)

	// Test with different delay configurations
	testCases := []struct {
		name    string
		delay   time.Duration
		minTime time.Duration // Minimum expected time for challenger response
		maxTime time.Duration // Maximum reasonable time (delay + some buffer)
	}{
		{
			name:    "NoDelay",
			delay:   0,
			minTime: 0,
			maxTime: 5 * time.Second, // Should respond quickly
		},
		{
			name:    "ShortDelay",
			delay:   2 * time.Second,
			minTime: 2 * time.Second,
			maxTime: 7 * time.Second, // 2s delay + 5s buffer
		},
		{
			name:    "MediumDelay",
			delay:   5 * time.Second,
			minTime: 5 * time.Second,
			maxTime: 10 * time.Second, // 5s delay + 5s buffer
		},
	}

	for _, tc := range testCases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			sys, _ := StartFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			// Create a dispute game with incorrect root to trigger challenger response
			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, common.Hash{0xaa, 0xbb, 0xcc})

			// Make an invalid claim that the honest challenger should counter
			invalidClaim := game.RootClaim(ctx)

			// Record time before starting challenger
			startTime := time.Now()

			// Start challenger with response delay
			game.StartChallenger(ctx, "sequencer", "DelayedChallenger",
				challenger.WithAlphabet(),
				challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
				challenger.WithResponseDelay(tc.delay),
				challenger.WithPollInterval(100*time.Millisecond), // Fast polling to ensure delay isn't from polling
			)

			// Wait for challenger to respond to the invalid root claim
			counterClaim := invalidClaim.WaitForCounterClaim(ctx)
			responseTime := time.Since(startTime)

			// Verify the response timing respects the configured delay
			require.GreaterOrEqualf(t, responseTime, tc.minTime,
				"Challenger responded too quickly (expected >= %v, got %v)", tc.minTime, responseTime)
			require.LessOrEqualf(t, responseTime, tc.maxTime,
				"Challenger took too long to respond (expected <= %v, got %v)", tc.maxTime, responseTime)

			// Verify the counter claim is valid (challenger actually responded correctly)
			require.NotNil(t, counterClaim, "Challenger should have posted a counter claim")
			counterClaim.RequireCorrectOutputRoot(ctx)
		})
	}
}

// TestChallengerResponseDelayWithMultipleActions tests that delay applies to each individual action
func TestChallengerResponseDelayWithMultipleActions(t *testing.T) {
	op_e2e.InitParallel(t)

	if testing.Short() {
		t.Skip("Skipping multi-action test during short run")
	}

	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	responseDelay := 2 * time.Second
	maxTime := responseDelay + 5*time.Second

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, common.Hash{0xaa, 0xbb, 0xcc})

	// Start challenger with response delay
	game.StartChallenger(ctx, "sequencer", "DelayedChallenger",
		challenger.WithAlphabet(),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
		challenger.WithResponseDelay(responseDelay),
		challenger.WithPollInterval(100*time.Millisecond),
	)

	// Track multiple challenger responses and their timing
	var responseTimes []time.Duration

	// First response to root claim
	claim := game.RootClaim(ctx)
	startTime := time.Now()
	claim = claim.WaitForCounterClaim(ctx)
	responseTimes = append(responseTimes, time.Since(startTime))

	// Second response - attack the challenger's claim to trigger another response
	startTime = time.Now()
	claim = claim.Attack(ctx, common.Hash{0x01})
	claim.WaitForCounterClaim(ctx)
	responseTimes = append(responseTimes, time.Since(startTime))

	// Verify each response respected the delay
	for i, responseTime := range responseTimes {
		require.GreaterOrEqualf(t, responseTime, responseDelay,
			"Response %d was too fast (expected >= %v, got %v)", i+1, responseDelay, responseTime)
		require.LessOrEqualf(t, responseTime, maxTime,
			"Response %d took too long (expected <= %v, got %v)", i+1, maxTime, responseTime)
	}

	require.Len(t, responseTimes, 2, "Should have measured 2 response times")
}

// TestChallengerResponseDelayRespectsCancellation tests that delay respects context cancellation
func TestChallengerResponseDelayRespectsCancellation(t *testing.T) {
	op_e2e.InitParallel(t)

	if testing.Short() {
		t.Skip("Skipping cancellation test during short run")
	}

	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	// Use a long delay that we'll interrupt with context cancellation
	longDelay := 30 * time.Second

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, common.Hash{0xaa, 0xbb, 0xcc})

	// Start challenger with long response delay
	challengerHelper := game.StartChallenger(ctx, "sequencer", "DelayedChallenger",
		challenger.WithAlphabet(),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
		challenger.WithResponseDelay(longDelay),
		challenger.WithPollInterval(100*time.Millisecond),
	)

	// Give challenger time to start and begin processing
	time.Sleep(1 * time.Second)

	// Stop the challenger (simulates context cancellation)
	startTime := time.Now()
	err := challengerHelper.Close()
	stopTime := time.Since(startTime)

	// Verify challenger stops quickly despite the long delay
	require.NoError(t, err, "Challenger should stop cleanly")
	require.Less(t, stopTime, 10*time.Second,
		"Challenger should stop quickly even with long delay (took %v)", stopTime)
}
