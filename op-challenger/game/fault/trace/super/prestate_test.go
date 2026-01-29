package super

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAbsolutePreState(t *testing.T) {
	t.Run("FailedToFetchOutput", func(t *testing.T) {
		rootProvider := &stubRootProvider{}
		provider := NewSuperRootPrestateProvider(rootProvider, 100)

		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)

		_, err = provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)
	})

	t.Run("ReturnsSuperRootForTimestamp", func(t *testing.T) {
		response := eth.SuperRootResponse{
			Timestamp: 100,
			SuperRoot: eth.Bytes32{0x11},
			Version:   eth.SuperRootVersionV1,
			Chains: []eth.ChainRootInfo{
				{
					ChainID:   eth.ChainID{2987},
					Canonical: eth.Bytes32{0x88},
					Pending:   []byte{1, 2, 3, 4, 5},
				},
				{
					ChainID:   eth.ChainID{100},
					Canonical: eth.Bytes32{0x10},
					Pending:   []byte{1, 2, 3, 4, 5},
				},
			},
		}
		expectedPreimage, err := response.ToSuper()
		require.NoError(t, err)
		rootProvider := &stubRootProvider{
			rootsByTimestamp: map[uint64]eth.SuperRootResponse{
				100: response,
			},
		}
		provider := NewSuperRootPrestateProvider(rootProvider, 100)

		preimage, err := provider.AbsolutePreState(context.Background())
		require.NoError(t, err)
		require.Equal(t, expectedPreimage, preimage)

		commitment, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedPreimage)), commitment)
	})
}

func TestSuperNodeAbsolutePreState(t *testing.T) {
	t.Run("FailedToFetchOutput", func(t *testing.T) {
		rootProvider := &stubSuperNodePrestateRootProvider{}
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)

		_, err = provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)
	})

	t.Run("NilDataReturnsNotFound", func(t *testing.T) {
		rootProvider := &stubSuperNodePrestateRootProvider{
			rootsByTimestamp: map[uint64]eth.SuperRootAtTimestampResponse{
				100: {
					Data: nil, // No data available
				},
			},
		}
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)
	})

	t.Run("ReturnsSuperRootForTimestamp", func(t *testing.T) {
		expectedPreimage := eth.NewSuperV1(100,
			eth.ChainIDAndOutput{ChainID: eth.ChainID{100}, Output: eth.Bytes32{0x10}},
			eth.ChainIDAndOutput{ChainID: eth.ChainID{2987}, Output: eth.Bytes32{0x88}},
		)
		response := eth.SuperRootAtTimestampResponse{
			Data: &eth.SuperRootResponseData{
				Super:     expectedPreimage,
				SuperRoot: eth.SuperRoot(expectedPreimage),
			},
		}
		rootProvider := &stubSuperNodePrestateRootProvider{
			rootsByTimestamp: map[uint64]eth.SuperRootAtTimestampResponse{
				100: response,
			},
		}
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		preimage, err := provider.AbsolutePreState(context.Background())
		require.NoError(t, err)
		require.Equal(t, expectedPreimage, preimage)

		commitment, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedPreimage)), commitment)
	})
}

type stubSuperNodePrestateRootProvider struct {
	rootsByTimestamp map[uint64]eth.SuperRootAtTimestampResponse
}

func (s *stubSuperNodePrestateRootProvider) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	root, ok := s.rootsByTimestamp[timestamp]
	if !ok {
		return eth.SuperRootAtTimestampResponse{}, ethereum.NotFound
	}
	return root, nil
}
