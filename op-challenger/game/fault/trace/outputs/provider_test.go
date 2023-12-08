package outputs

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	prestateBlock       = uint64(100)
	poststateBlock      = uint64(200)
	gameDepth           = uint64(7) // 128 leaf nodes
	prestateOutputRoot  = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	firstOutputRoot     = common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	poststateOutputRoot = common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	errNoOutputAtBlock  = errors.New("no output at block")
)

func TestGet(t *testing.T) {
	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := uint64(164)
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.Get(context.Background(), pos)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})

	t.Run("FirstBlockAfterPrestate", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPosition(int(gameDepth), big.NewInt(0)))
		require.NoError(t, err)
		require.Equal(t, firstOutputRoot, value)
	})

	t.Run("MissingOutputAtBlock", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		_, err := provider.Get(context.Background(), types.NewPosition(int(gameDepth), big.NewInt(1)))
		require.ErrorIs(t, err, errNoOutputAtBlock)
	})

	t.Run("PostStateBlock", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(228)))
		require.NoError(t, err)
		require.Equal(t, value, poststateOutputRoot)
	})

	t.Run("AfterPostStateBlock", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(229)))
		require.NoError(t, err)
		require.Equal(t, value, poststateOutputRoot)
	})
}

func TestGetBlockNumber(t *testing.T) {
	tests := []struct {
		name     string
		pos      types.Position
		expected uint64
	}{
		{"FirstBlockAfterPrestate", types.NewPosition(int(gameDepth), big.NewInt(0)), prestateBlock + 1},
		{"PostStateBlock", types.NewPositionFromGIndex(big.NewInt(228)), poststateBlock},
		{"AfterPostStateBlock", types.NewPositionFromGIndex(big.NewInt(229)), poststateBlock},
		{"Root", types.NewPositionFromGIndex(big.NewInt(1)), poststateBlock},
		{"MiddleNode1", types.NewPosition(int(gameDepth-1), big.NewInt(2)), 106},
		{"MiddleNode2", types.NewPosition(int(gameDepth-1), big.NewInt(3)), 108},
		{"Leaf1", types.NewPosition(int(gameDepth), big.NewInt(1)), prestateBlock + 2},
		{"Leaf2", types.NewPosition(int(gameDepth), big.NewInt(2)), prestateBlock + 3},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
			actual, err := provider.BlockNumber(test.pos)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := uint64(164)
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.BlockNumber(pos)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})
}

func TestGetStepData(t *testing.T) {
	provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
	_, _, _, err := provider.GetStepData(context.Background(), types.NewPosition(1, common.Big0))
	require.ErrorIs(t, err, ErrGetStepData)
}

func setupWithTestData(t *testing.T, prestateBlock, poststateBlock uint64, customGameDepth ...uint64) (*OutputTraceProvider, *stubRollupClient) {
	rollupClient := stubRollupClient{
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
	}
	inputGameDepth := gameDepth
	if len(customGameDepth) > 0 {
		inputGameDepth = customGameDepth[0]
	}
	return &OutputTraceProvider{
		logger:         testlog.Logger(t, log.LvlInfo),
		rollupClient:   &rollupClient,
		prestateBlock:  prestateBlock,
		poststateBlock: poststateBlock,
		gameDepth:      inputGameDepth,
	}, &rollupClient
}

type stubRollupClient struct {
	errorsOnPrestateFetch bool
	outputs               map[uint64]*eth.OutputResponse
}

func (s *stubRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	output, ok := s.outputs[blockNum]
	if !ok || s.errorsOnPrestateFetch {
		return nil, fmt.Errorf("%w: %d", errNoOutputAtBlock, blockNum)
	}
	return output, nil
}
