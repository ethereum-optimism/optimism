package interop

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type mockL1Source struct {
	blocks map[uint64]eth.L1BlockRef
}

func (m *mockL1Source) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	ref, ok := m.blocks[num]
	if !ok {
		return eth.L1BlockRef{}, fmt.Errorf("block %d not found", num)
	}
	return ref, nil
}

func TestByNumberConsistencyChecker_SameL1Chain(t *testing.T) {
	t.Parallel()

	hashA := common.HexToHash("0xaaaa")
	hashB := common.HexToHash("0xbbbb")

	source := &mockL1Source{
		blocks: map[uint64]eth.L1BlockRef{
			100: {Hash: hashA, Number: 100},
			200: {Hash: hashB, Number: 200},
		},
	}
	checker := newByNumberConsistencyChecker(source)

	t.Run("all match canonical", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{Hash: hashA, Number: 100},
			{Hash: hashB, Number: 200},
		})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("mismatch detected", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{Hash: hashA, Number: 100},
			{Hash: common.HexToHash("0xdead"), Number: 200}, // wrong hash
		})
		require.NoError(t, err)
		require.False(t, same)
	})

	t.Run("zero block IDs skipped", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{
			{},
			{Hash: hashA, Number: 100},
			{},
		})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("empty heads list", func(t *testing.T) {
		same, err := checker.SameL1Chain(context.Background(), []eth.BlockID{})
		require.NoError(t, err)
		require.True(t, same)
	})

	t.Run("nil checker returns nil", func(t *testing.T) {
		nilChecker := newByNumberConsistencyChecker(nil)
		require.Nil(t, nilChecker)
	})
}
