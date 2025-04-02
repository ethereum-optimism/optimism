package systest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockGethClient struct {
	latestBlockNum int
	headersByNum   map[int]types.Header
}

func (m mockGethClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	idx := int(0)
	if number == nil {
		idx = m.latestBlockNum
	} else {
		idx = int(number.Int64())
	}
	h := m.headersByNum[idx]
	return &h, nil
}
func (mockGethClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	panic("unimplemented")
}
func (mockGethClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	panic("unimplemented")
}
func (mockGethClient) Close() {}

var _ HeaderProvider = mockGethClient{}

func TestCheckForChainFork(t *testing.T) {

	leader := mockGethClient{latestBlockNum: 0, headersByNum: map[int]types.Header{
		0: {
			Number: big.NewInt(0),
			TxHash: common.HexToHash("0x0"),
		},
		1: {
			Number: big.NewInt(1),
			TxHash: common.HexToHash("0x1"),
		},
	},
	}

	followerA := mockGethClient{latestBlockNum: 0, headersByNum: map[int]types.Header{
		0: {
			Number: big.NewInt(0),
			TxHash: common.HexToHash("0x0"), // in sync with mockA at this block
		},
		1: {
			Number: big.NewInt(1),
			TxHash: common.HexToHash("0xb"), // forks off from leader at this block
		},
	},
	}

	followerB := mockGethClient{latestBlockNum: 0, headersByNum: map[int]types.Header{
		0: {
			Number: big.NewInt(0),
			TxHash: common.HexToHash("0x0"), // forks off from leader at this block
		},
		1: {
			Number: big.NewInt(1),
			TxHash: common.HexToHash("0xb"), // forks off from leader at this block
		},
	},
	}

	// First scenario is that the leader and follower are in sync initially, but then split:
	secondCheck, firstErr := checkForChainFork(context.Background(), []HeaderProvider{leader, followerA}, testlog.Logger(t, log.LevelDebug))
	require.NoError(t, firstErr)
	leader.latestBlockNum = 1    // advance the chain head
	followerA.latestBlockNum = 1 // advance the chain head
	require.Error(t, secondCheck(), "expected chain split error")

	// Second scenario is that the leader and follower are forked immediately:
	_, firstErr = checkForChainFork(context.Background(), []HeaderProvider{leader, followerB}, testlog.Logger(t, log.LevelDebug))
	require.Error(t, firstErr, "expected chain split error")

}
