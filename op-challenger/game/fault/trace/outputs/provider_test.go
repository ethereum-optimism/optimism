package outputs

import (
	"context"
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
)

func TestGet(t *testing.T) {
	t.Run("PrePrestateErrors", func(t *testing.T) {
		provider, _ := setupWithTestData(t, 0, poststateBlock)
		_, err := provider.Get(context.Background(), types.NewPosition(1, common.Big0))
		require.ErrorAs(t, fmt.Errorf("no output at block %d", 1), &err)
	})

	t.Run("ErrorsTraceIndexOutOfBounds", func(t *testing.T) {
		deepGame := uint64(64)
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock, deepGame)
		pos := types.NewPosition(0, big.NewInt(0))
		_, err := provider.Get(context.Background(), pos)
		require.ErrorAs(t, fmt.Errorf("trace index %v is greater than max uint64", pos.TraceIndex(int(deepGame))), &err)
	})

	t.Run("MisconfiguredPoststateErrors", func(t *testing.T) {
		provider, _ := setupWithTestData(t, 0, 0)
		_, err := provider.Get(context.Background(), types.NewPosition(1, common.Big0))
		require.ErrorAs(t, fmt.Errorf("no output at block %d", 0), &err)
	})

	t.Run("FirstBlockAfterPrestate", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(128)))
		require.NoError(t, err)
		require.Equal(t, firstOutputRoot, value)
	})

	t.Run("MissingOutputAtBlock", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		_, err := provider.Get(context.Background(), types.NewPositionFromGIndex(big.NewInt(129)))
		require.ErrorAs(t, fmt.Errorf("no output at block %d", prestateBlock+2), &err)
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

func TestAbsolutePreStateCommitment(t *testing.T) {
	t.Run("FailedToFetchOutput", func(t *testing.T) {
		provider, rollupClient := setupWithTestData(t, prestateBlock, poststateBlock)
		rollupClient.errorsOnPrestateFetch = true
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorAs(t, fmt.Errorf("no output at block %d", prestateBlock), &err)
	})

	t.Run("ReturnsCorrectPrestateOutput", func(t *testing.T) {
		provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
		value, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, value, prestateOutputRoot)
	})
}

func TestGetStepData(t *testing.T) {
	provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
	_, _, _, err := provider.GetStepData(context.Background(), types.NewPosition(1, common.Big0))
	require.ErrorIs(t, err, GetStepDataErr)
}

func TestAbsolutePreState(t *testing.T) {
	provider, _ := setupWithTestData(t, prestateBlock, poststateBlock)
	_, err := provider.AbsolutePreState(context.Background())
	require.ErrorIs(t, err, AbsolutePreStateErr)
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

func (s *stubRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	output, ok := s.outputs[blockNum]
	if !ok || s.errorsOnPrestateFetch {
		return nil, fmt.Errorf("no output at block %d", blockNum)
	}
	return output, nil
}
