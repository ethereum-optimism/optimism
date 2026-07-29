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
// This is a sanity check integration test that verifies minimum delay timing is honored
func TestChallengerResponseDelay(t *testing.T) {
	op_e2e.InitParallel(t)

	// Test with different delay configurations
	testCases := []struct {
		name    string
		delay   time.Duration
		minTime time.Duration // Minimum expected time for challenger response
	}{
		{
			name:    "NoDelay",
			delay:   0,
			minTime: 0, // No minimum delay expected
		},
		{
			name:    "ShortDelay",
			delay:   2 * time.Second,
			minTime: 2 * time.Second, // Must take at least the configured delay
		},
		{
			name:    "MediumDelay",
			delay:   5 * time.Second,
			minTime: 5 * time.Second, // Must take at least the configured delay
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

			// Sanity check: verify minimum delay is respected (includes polling time and system overhead)
			require.GreaterOrEqualf(t, responseTime, tc.minTime,
				"Challenger responded too quickly (expected >= %v, got %v)", tc.minTime, responseTime)

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

	// Sanity check: verify each response took at least the minimum delay
	for i, responseTime := range responseTimes {
		require.GreaterOrEqualf(t, responseTime, responseDelay,
			"Response %d was too fast (expected >= %v, got %v)", i+1, responseDelay, responseTime)
	}

	require.Len(t, responseTimes, 2, "Should have measured 2 response times")
}
