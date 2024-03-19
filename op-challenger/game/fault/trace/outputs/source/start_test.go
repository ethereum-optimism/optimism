package source

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestFindGuaranteedSafeHead_ErrorWhenSafeHeadNotAvailable(t *testing.T) {
	cfg := &rollup.Config{
		SeqWindowSize: 100,
		Genesis: rollup.Genesis{
			L2: eth.BlockID{
				Hash:   common.Hash{0x1},
				Number: 1343,
			},
		},
	}
	expectedErr := errors.New("boom")
	l2Source := &stubL2Source{byLabelError: expectedErr}
	_, err := FindGuaranteedSafeHead(context.Background(), cfg, 248249, l2Source)
	require.Error(t, err)
}

func TestFindGuaranteedSafeHead_L2GenesisWhenL1HeadNotPastSequenceWindow(t *testing.T) {
	cfg := &rollup.Config{
		SeqWindowSize: 100,
		Genesis: rollup.Genesis{
			L2: eth.BlockID{
				Hash:   common.Hash{0x1},
				Number: 1343,
			},
		},
	}
	l2Source := &stubL2Source{}
	actual, err := FindGuaranteedSafeHead(context.Background(), cfg, 99, l2Source)
	require.NoError(t, err)
	require.Equal(t, cfg.Genesis.L2, actual)
}

func TestFindGuaranteedSafeHead_L2GenesisWhenL1HeadEqualToSequenceWindow(t *testing.T) {
	cfg := &rollup.Config{
		SeqWindowSize: 100,
		Genesis: rollup.Genesis{
			L2: eth.BlockID{
				Hash:   common.Hash{0x1},
				Number: 1343,
			},
		},
	}
	l2Source := &stubL2Source{}
	actual, err := FindGuaranteedSafeHead(context.Background(), cfg, 100, l2Source)
	require.NoError(t, err)
	require.Equal(t, cfg.Genesis.L2, actual)
}

func TestFindGuaranteedSafeHead_SafeHeadIsGuaranteedSafe(t *testing.T) {
	cfg := &rollup.Config{
		SeqWindowSize: 100,
		Genesis: rollup.Genesis{
			L2: eth.BlockID{
				Hash:   common.Hash{0x1},
				Number: 1343,
			},
		},
	}
	safeHead := eth.L2BlockRef{
		Hash:   common.Hash{0xaa},
		Number: 1000,
		L1Origin: eth.BlockID{
			Number: 499,
		},
	}
	l2Source := &stubL2Source{
		safe: safeHead,
	}
	actual, err := FindGuaranteedSafeHead(context.Background(), cfg, 500, l2Source)
	require.NoError(t, err)
	require.Equal(t, cfg.Genesis.L2, actual)
}

func TestFindGuaranteedSafeHead_SearchBackwardFromSafeHead(t *testing.T) {
	cfg := &rollup.Config{
		SeqWindowSize: 100,
		Genesis: rollup.Genesis{
			L2: eth.BlockID{
				Hash:   common.Hash{0x1},
				Number: 500,
			},
		},
	}
	safeHead := eth.L2BlockRef{
		Hash:   common.Hash{0xaa},
		Number: 1500,
		L1Origin: eth.BlockID{
			Number: 5000,
		},
	}

	l2Source := &stubL2Source{
		safe:   safeHead,
		blocks: make(map[uint64]eth.L2BlockRef),
	}
	for i := cfg.Genesis.L2.Number + 1; i < safeHead.Number; i++ {
		block := eth.L2BlockRef{
			Hash:   common.Hash{byte(i)},
			Number: i,
			L1Origin: eth.BlockID{
				Number: 2000 + i, // Make it different from L2 block number
			},
		}
		l2Source.blocks[block.Number] = block
	}
	expected := l2Source.blocks[1260]
	actual, err := FindGuaranteedSafeHead(context.Background(), cfg, expected.L1Origin.Number+cfg.SeqWindowSize+1, l2Source)
	require.NoError(t, err)
	require.Equal(t, expected.ID(), actual)
	maxQueries := int(math.Log2(float64(len(l2Source.blocks))) + 1)
	require.LessOrEqual(t, l2Source.byNumCount, maxQueries, "Should use an efficient search")
}

type stubL2Source struct {
	safe         eth.L2BlockRef
	byLabelError error
	blocks       map[uint64]eth.L2BlockRef
	byNumCount   int
}

func (s *stubL2Source) L2BlockRefByLabel(_ context.Context, _ eth.BlockLabel) (eth.L2BlockRef, error) {
	return s.safe, s.byLabelError
}

func (s *stubL2Source) L2BlockRefByNumber(_ context.Context, blockNum uint64) (eth.L2BlockRef, error) {
	s.byNumCount++
	ref, ok := s.blocks[blockNum]
	if !ok {
		return eth.L2BlockRef{}, errors.New("not found")
	}
	return ref, nil
}
