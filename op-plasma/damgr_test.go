package plasma

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
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

	// activate a couple of subsequent challenges
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
	_, err = state.ExpireChallenges(3714038)
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

func TestDAChallengeDetached(t *testing.T) {
	logger := testlog.Logger(t, log.LvlDebug)

	rng := rand.New(rand.NewSource(1234))
	state := NewState(logger, &NoopMetrics{})

	challengeWindow := uint64(6)
	resolveWindow := uint64(6)

	c1 := RandomData(rng, 32)
	c2 := RandomData(rng, 32)

	// c1 at bn1 is missing, pipeline stalls
	state.GetOrTrackChallenge(c1, 1, challengeWindow)

	// c2 at bn2 is challenged at bn3
	require.True(t, state.IsTracking(c2, 2))
	state.SetActiveChallenge(c2, 3, resolveWindow)

	// c1 is finally challenged at bn5
	state.SetActiveChallenge(c1, 5, resolveWindow)

	// c2 expires but should not trigger a reset because we don't know if it's valid yet
	bn, err := state.ExpireChallenges(10)
	require.NoError(t, err)
	require.Equal(t, uint64(0), bn)

	// c1 expires finally
	bn, err = state.ExpireChallenges(11)
	require.ErrorIs(t, err, ErrReorgRequired)
	require.Equal(t, uint64(1), bn)

	// pruning finalized block is safe
	state.Prune(bn)

	// pipeline discovers c2
	comm := state.GetOrTrackChallenge(c2, 2, challengeWindow)
	// it is already marked as expired so it will be skipped without needing a reorg
	require.Equal(t, ChallengeExpired, comm.challengeStatus)

	// later when we get to finalizing block 10 + margin, the pending challenge is safely pruned
	state.Prune(210)
	require.Equal(t, 0, len(state.expiredComms))
}

// cannot import from testutils at this time because of import cycle
type mockL1Fetcher struct {
	mock.Mock
}

func (m *mockL1Fetcher) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	out := m.Mock.Called(hash)
	return out.Get(0).(eth.BlockInfo), out.Get(1).(types.Transactions), out.Error(2)
}

func (m *mockL1Fetcher) ExpectInfoAndTxsByHash(hash common.Hash, info eth.BlockInfo, transactions types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByHash", hash).Once().Return(info, transactions, err)
}

func (m *mockL1Fetcher) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	out := m.Mock.Called(blockHash)
	return *out.Get(0).(*eth.BlockInfo), out.Get(1).(types.Receipts), out.Error(2)
}

func (m *mockL1Fetcher) ExpectFetchReceipts(hash common.Hash, info eth.BlockInfo, receipts types.Receipts, err error) {
	m.Mock.On("FetchReceipts", hash).Once().Return(&info, receipts, err)
}

func (m *mockL1Fetcher) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	out := m.Mock.Called(num)
	return out.Get(0).(eth.L1BlockRef), out.Error(1)
}

func (m *mockL1Fetcher) ExpectL1BlockRefByNumber(num uint64, ref eth.L1BlockRef, err error) {
	m.Mock.On("L1BlockRefByNumber", num).Once().Return(ref, err)
}

func TestFilterInvalidBlockNumber(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	ctx := context.Background()

	l1F := &mockL1Fetcher{}

	storage := NewMockDAClient(logger)

	daddr := common.HexToAddress("0x978e3286eb805934215a88694d80b09aded68d90")
	pcfg := Config{
		ChallengeWindow: 90, ResolveWindow: 90, DAChallengeContractAddress: daddr,
	}

	bn := uint64(19)
	bhash := common.HexToHash("0xd438144ffab918b1349e7cd06889c26800c26d8edc34d64f750e3e097166a09c")

	state := NewState(logger, &NoopMetrics{})

	da := NewPlasmaDAWithState(logger, pcfg, storage, &NoopMetrics{}, state)

	receipts := types.Receipts{&types.Receipt{
		Type:   2,
		Status: 1,
		Logs: []*types.Log{
			{
				BlockNumber: bn,
				Address:     daddr,
				Topics: []common.Hash{
					common.HexToHash("0xa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05"),
				},
			},
			{
				BlockNumber: bn,
				Address:     daddr,
				Topics: []common.Hash{
					common.HexToHash("0xc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad"),
					common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000e"),
				},
				Data: common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002100eed82c1026bdd0f23461dd6ca515ef677624e63e6fc0ff91e3672af8eddf579d00000000000000000000000000000000000000000000000000000000000000"),
			},
		},
		BlockNumber: big.NewInt(int64(bn)),
	}}
	id := eth.BlockID{
		Number: bn,
		Hash:   bhash,
	}
	l1F.ExpectFetchReceipts(bhash, nil, receipts, nil)

	// we get 1 log successfully filtered as valid status updated contract event
	logs, err := da.fetchChallengeLogs(ctx, l1F, id)
	require.NoError(t, err)
	require.Equal(t, len(logs), 1)

	// commitment is tracked but not canonical
	status, comm, err := da.decodeChallengeStatus(logs[0])
	require.NoError(t, err)

	c, has := state.commsByKey[string(comm.Encode())]
	require.True(t, has)
	require.False(t, c.canonical)

	require.Equal(t, ChallengeActive, status)
	// once tracked, set as active based on decoded status
	state.SetActiveChallenge(comm.Encode(), bn, pcfg.ResolveWindow)

	// once we request it during derivation it becomes canonical
	tracked := state.GetOrTrackChallenge(comm.Encode(), 14, pcfg.ChallengeWindow)
	require.True(t, tracked.canonical)

	require.Equal(t, ChallengeActive, tracked.challengeStatus)
	require.Equal(t, uint64(14), tracked.blockNumber)
	require.Equal(t, bn+pcfg.ResolveWindow, tracked.expiresAt)
}
