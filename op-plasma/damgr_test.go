package plasma

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func RandomData(rng *rand.Rand, size int) []byte {
	out := make([]byte, size)
	rng.Read(out)
	return out
}

// TestDAChallengeState is a simple test with small values to verify the finalized head logic
func TestDAChallengeState(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)

	rng := rand.New(rand.NewSource(1234))
	state := NewState(logger, &NoopMetrics{})

	i := uint64(1)

	challengeWindow := uint64(6)
	resolveWindow := uint64(6)

	// track commitments in the first 10 blocks
	for ; i < 10; i++ {
		// this is akin to stepping the derivation pipeline through a range a blocks each with a commitment
		state.SetInputCommitment(RandomData(rng, 32), i, challengeWindow)
	}

	// blocks are finalized after the challenge window expires
	bn, err := state.ExpireChallenges(10)
	require.NoError(t, err)
	// finalized head = 10 - 6 = 4
	require.Equal(t, uint64(4), bn)

	// track the next commitment and mark it as challenged
	c := RandomData(rng, 32)
	// add input commitment at block i = 10
	state.SetInputCommitment(c, 10, challengeWindow)
	// i+4 is the block at which it was challenged
	state.SetActiveChallenge(c, 14, resolveWindow)

	for j := i + 1; j < 18; j++ {
		// continue walking the pipeline through some more blocks with commitments
		state.SetInputCommitment(RandomData(rng, 32), j, challengeWindow)
	}

	// finalized l1 origin should not extend past the resolve window
	bn, err = state.ExpireChallenges(18)
	require.NoError(t, err)
	// finalized is active_challenge_block - 1 = 10 - 1 and cannot move until the challenge expires
	require.Equal(t, uint64(9), bn)

	// walk past the resolve window
	for j := uint64(18); j < 22; j++ {
		state.SetInputCommitment(RandomData(rng, 32), j, challengeWindow)
	}

	// no more active challenges, the finalized head can catch up to the challenge window
	bn, err = state.ExpireChallenges(22)
	require.ErrorIs(t, err, ErrReorgRequired)
	// finalized head is now 22 - 6 = 16
	require.Equal(t, uint64(16), bn)

	// cleanup state we don't need anymore
	state.Prune(22)
	// now if we expire the challenges again, it won't request a reorg again
	bn, err = state.ExpireChallenges(22)
	require.NoError(t, err)
	// finalized head hasn't moved
	require.Equal(t, uint64(16), bn)

	i = 22
	// add one more commitment and challenge it
	c = RandomData(rng, 32)
	state.SetInputCommitment(c, 22, challengeWindow)
	// challenge 3 blocks after
	state.SetActiveChallenge(c, 25, resolveWindow)

	// exceed the challenge window with more commitments
	for j := uint64(23); j < 30; j++ {
		state.SetInputCommitment(RandomData(rng, 32), j, challengeWindow)
	}

	// finalized head should not extend past the resolve window
	bn, err = state.ExpireChallenges(30)
	require.NoError(t, err)
	// finalized head is stuck waiting for resolve window
	require.Equal(t, uint64(21), bn)

	input := RandomData(rng, 100)
	// resolve the challenge
	state.SetResolvedChallenge(c, input, 30)

	// finalized head catches up
	bn, err = state.ExpireChallenges(31)
	require.NoError(t, err)
	// finalized head is now 31 - 6 = 25
	require.Equal(t, uint64(25), bn)

	// the resolved input is also stored
	storedInput, err := state.GetResolvedInput(c)
	require.NoError(t, err)
	require.Equal(t, input, storedInput)
}

// TestExpireChallenges expires challenges and prunes the state for longer windows
// with commitments every 6 blocks.
func TestExpireChallenges(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)

	rng := rand.New(rand.NewSource(1234))
	state := NewState(logger, &NoopMetrics{})

	comms := make(map[uint64][]byte)

	i := uint64(3713854)

	var finalized uint64

	challengeWindow := uint64(90)
	resolveWindow := uint64(90)

	// increment new commitments every 6 blocks
	for ; i < 3713948; i += 6 {
		comm := RandomData(rng, 32)
		comms[i] = comm
		logger.Info("set commitment", "block", i)
		cm := state.GetOrTrackChallenge(comm, i, challengeWindow)
		require.NotNil(t, cm)

		bn, err := state.ExpireChallenges(i)
		logger.Info("expire challenges", "finalized head", bn, "err", err)

		// only update finalized head if it has moved
		if bn > finalized {
			finalized = bn
			// prune unused state
			state.Prune(bn)
		}
	}

	// activate a couple of subsquent challenges
	state.SetActiveChallenge(comms[3713926], 3713948, resolveWindow)

	state.SetActiveChallenge(comms[3713932], 3713950, resolveWindow)

	// continue incrementing commitments
	for ; i < 3714038; i += 6 {
		comm := RandomData(rng, 32)
		comms[i] = comm
		logger.Info("set commitment", "block", i)
		cm := state.GetOrTrackChallenge(comm, i, challengeWindow)
		require.NotNil(t, cm)

		bn, err := state.ExpireChallenges(i)
		logger.Info("expire challenges", "expired", bn, "err", err)

		if bn > finalized {
			finalized = bn
			state.Prune(bn)
		}

	}

	// finalized head does not move as it expires previously seen blocks
	bn, err := state.ExpireChallenges(3714034)
	require.NoError(t, err)
	require.Equal(t, uint64(3713920), bn)

	bn, err = state.ExpireChallenges(3714035)
	require.NoError(t, err)
	require.Equal(t, uint64(3713920), bn)

	bn, err = state.ExpireChallenges(3714036)
	require.NoError(t, err)
	require.Equal(t, uint64(3713920), bn)

	bn, err = state.ExpireChallenges(3714037)
	require.NoError(t, err)
	require.Equal(t, uint64(3713920), bn)

	// lastly we get to the resolve window and trigger a reorg
	bn, err = state.ExpireChallenges(3714038)
	require.ErrorIs(t, err, ErrReorgRequired)

	// this is simulating a pipeline reset where it walks back challenge + resolve window
	for i := uint64(3713854); i < 3714044; i += 6 {
		cm := state.GetOrTrackChallenge(comms[i], i, challengeWindow)
		require.NotNil(t, cm)

		// check that the challenge status was updated to expired
		if i == 3713926 {
			require.Equal(t, ChallengeExpired, cm.challengeStatus)
		}
	}

	bn, err = state.ExpireChallenges(3714038)
	require.NoError(t, err)

	// finalized at last
	require.Equal(t, uint64(3713926), bn)
}
