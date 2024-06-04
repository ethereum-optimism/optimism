package outputs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	prestateBlock       = uint64(100)
	poststateBlock      = uint64(200)
	gameDepth           = types.Depth(7) // 128 leaf nodes
	prestateOutputRoot  = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	firstOutputRoot     = common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	poststateOutputRoot = common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	errNoOutputAtBlock  = errors.New("no output at block")
)

func TestGet(t *testing.T) {
	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := types.Depth(164)
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.Get(context.Background(), pos)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})

	t.Run("FirstBlockAfterPrestate", func(t *testing.T) {
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(0)))
		require.NoError(t, err)
		require.Equal(t, firstOutputRoot, value)
	})

	t.Run("MissingOutputAtBlock", func(t *testing.T) {
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		_, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(1)))
		require.ErrorIs(t, err, errNoOutputAtBlock)
	})

	t.Run("PostStateBlock", func(t *testing.T) {
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(228)))
		require.NoError(t, err)
		require.Equal(t, value, poststateOutputRoot)
	})

	t.Run("AfterPostStateBlock", func(t *testing.T) {
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(229)))
		require.NoError(t, err)
		require.Equal(t, value, poststateOutputRoot)
	})
}

func TestHonestBlockNumber(t *testing.T) {
	tests := []struct {
		name        string
		pos         types.Position
		expected    uint64
		maxSafeHead uint64
	}{
		{"FirstBlockAfterPrestate", types.NewPosition(gameDepth, big.NewInt(0)), prestateBlock + 1, math.MaxUint64},
		{"PostStateBlock", types.NewPositionFromGIndex(big.NewInt(228)), poststateBlock, math.MaxUint64},
		{"AfterPostStateBlock", types.NewPositionFromGIndex(big.NewInt(229)), poststateBlock, math.MaxUint64},
		{"Root", types.NewPositionFromGIndex(big.NewInt(1)), poststateBlock, math.MaxUint64},
		{"MiddleNode1", types.NewPosition(gameDepth-1, big.NewInt(2)), 106, math.MaxUint64},
		{"MiddleNode2", types.NewPosition(gameDepth-1, big.NewInt(3)), 108, math.MaxUint64},
		{"Leaf1", types.NewPosition(gameDepth, big.NewInt(1)), prestateBlock + 2, math.MaxUint64},
		{"Leaf2", types.NewPosition(gameDepth, big.NewInt(2)), prestateBlock + 3, math.MaxUint64},

		{"RestrictedHead-UnderLimit", types.NewPosition(gameDepth, big.NewInt(48)), prestateBlock + 49, prestateBlock + 50},
		{"RestrictedHead-EqualLimit", types.NewPosition(gameDepth, big.NewInt(49)), prestateBlock + 50, prestateBlock + 50},
		{"RestrictedHead-OverLimit", types.NewPosition(gameDepth, big.NewInt(50)), prestateBlock + 50, prestateBlock + 50},
		{"RestrictedHead-PastPostState", types.NewPosition(gameDepth, big.NewInt(1000)), prestateBlock + 50, prestateBlock + 50},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider, stubRollupClient, _ := setupWithTestData(t, prestateBlock, poststateBlock)
			stubRollupClient.maxSafeHead = test.maxSafeHead
			actual, err := provider.HonestBlockNumber(context.Background(), test.pos)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := types.Depth(164)
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.HonestBlockNumber(context.Background(), pos)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})
}

func TestGetL2BlockNumberChallenge(t *testing.T) {
	tests := []struct {
		name            string
		maxSafeHead     uint64
		expectChallenge bool
	}{
		{"NoChallengeWhenMaxHeadNotLimited", math.MaxUint64, false},
		{"NoChallengeWhenBeforeMaxHead", poststateBlock + 1, false},
		{"NoChallengeWhenAtMaxHead", poststateBlock, false},
		{"ChallengeWhenBeforeMaxHead", poststateBlock - 1, true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider, stubRollupClient, stubL2Client := setupWithTestData(t, prestateBlock, poststateBlock)
			stubRollupClient.maxSafeHead = test.maxSafeHead
			if test.expectChallenge {
				stubRollupClient.outputs[test.maxSafeHead] = &eth.OutputResponse{
					OutputRoot: eth.Bytes32{0xaa},
					BlockRef: eth.L2BlockRef{
						Number: test.maxSafeHead,
					},
				}
				stubL2Client.headers[test.maxSafeHead] = &ethTypes.Header{
					Number: new(big.Int).SetUint64(test.maxSafeHead),
					Root:   common.Hash{0xcc},
				}
			}
			actual, err := provider.GetL2BlockNumberChallenge(context.Background())
			if test.expectChallenge {
				require.NoError(t, err)
				require.Equal(t, &types.InvalidL2BlockNumberChallenge{
					Output: stubRollupClient.outputs[test.maxSafeHead],
					Header: stubL2Client.headers[test.maxSafeHead],
				}, actual)
			} else {
				require.ErrorIs(t, err, types.ErrL2BlockNumberValid)
			}
		})
	}
}

func TestClaimedBlockNumber(t *testing.T) {
	tests := []struct {
		name        string
		pos         types.Position
		expected    uint64
		maxSafeHead uint64
	}{
		{"FirstBlockAfterPrestate", types.NewPosition(gameDepth, big.NewInt(0)), prestateBlock + 1, math.MaxUint64},
		{"PostStateBlock", types.NewPositionFromGIndex(big.NewInt(228)), poststateBlock, math.MaxUint64},
		{"AfterPostStateBlock", types.NewPositionFromGIndex(big.NewInt(229)), poststateBlock, math.MaxUint64},
		{"Root", types.NewPositionFromGIndex(big.NewInt(1)), poststateBlock, math.MaxUint64},
		{"MiddleNode1", types.NewPosition(gameDepth-1, big.NewInt(2)), 106, math.MaxUint64},
		{"MiddleNode2", types.NewPosition(gameDepth-1, big.NewInt(3)), 108, math.MaxUint64},
		{"Leaf1", types.NewPosition(gameDepth, big.NewInt(1)), prestateBlock + 2, math.MaxUint64},
		{"Leaf2", types.NewPosition(gameDepth, big.NewInt(2)), prestateBlock + 3, math.MaxUint64},

		{"RestrictedHead-UnderLimit", types.NewPosition(gameDepth, big.NewInt(48)), prestateBlock + 49, prestateBlock + 50},
		{"RestrictedHead-EqualLimit", types.NewPosition(gameDepth, big.NewInt(49)), prestateBlock + 50, prestateBlock + 50},
		{"RestrictedHead-OverLimit", types.NewPosition(gameDepth, big.NewInt(50)), prestateBlock + 51, prestateBlock + 50},
		{"RestrictedHead-PastPostState", types.NewPosition(gameDepth, big.NewInt(300)), poststateBlock, prestateBlock + 50},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider, stubRollupClient, _ := setupWithTestData(t, prestateBlock, poststateBlock)
			stubRollupClient.maxSafeHead = test.maxSafeHead
			actual, err := provider.ClaimedBlockNumber(test.pos)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := types.Depth(164)
		provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.ClaimedBlockNumber(pos)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})
}

func TestGetStepData(t *testing.T) {
	provider, _, _ := setupWithTestData(t, prestateBlock, poststateBlock)
	_, _, _, err := provider.GetStepData(context.Background(), types.NewPosition(1, common.Big0))
	require.ErrorIs(t, err, ErrGetStepData)
}

func setupWithTestData(t *testing.T, prestateBlock, poststateBlock uint64, customGameDepth ...types.Depth) (*OutputTraceProvider, *stubRollupClient, *stubL2HeaderSource) {
	rollupClient := &stubRollupClient{
		outputs: map[uint64]*eth.OutputResponse{
			prestateBlock: {
				OutputRoot: eth.Bytes32(prestateOutputRoot),
			},
			101: {
				OutputRoot: eth.Bytes32(firstOutputRoot),
			},
			poststateBlock: {
				OutputRoot: eth.Bytes32(poststateOutputRoot),
			},
		},
		maxSafeHead: math.MaxUint64,
	}
	l2Client := &stubL2HeaderSource{
		headers: make(map[uint64]*ethTypes.Header),
	}
	inputGameDepth := gameDepth
	if len(customGameDepth) > 0 {
		inputGameDepth = customGameDepth[0]
	}
	return &OutputTraceProvider{
		logger:         testlog.Logger(t, log.LevelInfo),
		rollupProvider: rollupClient,
		l2Client:       l2Client,
		prestateBlock:  prestateBlock,
		poststateBlock: poststateBlock,
		gameDepth:      inputGameDepth,
	}, rollupClient, l2Client
}

type stubRollupClient struct {
	errorsOnPrestateFetch bool
	outputs               map[uint64]*eth.OutputResponse
	maxSafeHead           uint64
}

func (s *stubRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	output, ok := s.outputs[blockNum]
	if !ok || s.errorsOnPrestateFetch {
		return nil, fmt.Errorf("%w: %d", errNoOutputAtBlock, blockNum)
	}
	return output, nil
}

func (s *stubRollupClient) SafeHeadAtL1Block(_ context.Context, l1BlockNum uint64) (*eth.SafeHeadResponse, error) {
	return &eth.SafeHeadResponse{
		SafeHead: eth.BlockID{
			Number: s.maxSafeHead,
			Hash:   common.Hash{0x11},
		},
	}, nil
}

type stubL2HeaderSource struct {
	headers map[uint64]*ethTypes.Header
}

func (s *stubL2HeaderSource) HeaderByNumber(_ context.Context, num *big.Int) (*ethTypes.Header, error) {
	header, ok := s.headers[num.Uint64()]
	if !ok {
		return nil, ethereum.NotFound
	}
	return header, nil
}
