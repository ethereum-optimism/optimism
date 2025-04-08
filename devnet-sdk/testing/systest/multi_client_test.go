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

func TestRequireNoChainFork(t *testing.T) {

	mockA := mockGethClient{latestBlockNum: 0, headersByNum: map[int]types.Header{
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

	mockB := mockGethClient{latestBlockNum: 0, headersByNum: map[int]types.Header{
		0: {
			Number: big.NewInt(0),
			TxHash: common.HexToHash("0x0"), // in sync with mockA at this block
		},
		1: {
			Number: big.NewInt(1),
			TxHash: common.HexToHash("0xb"), // forks off from mockA at this block
		},
	},
	}

	secondCheck, firstErr := checkForChainFork(context.Background(), []HeaderProvider{mockA, mockB}, testlog.Logger(t, log.LevelDebug))

	require.NoError(t, firstErr)
	mockA.latestBlockNum = 1
	mockB.latestBlockNum = 1

	require.Error(t, secondCheck(), "expected chain split error")
}
