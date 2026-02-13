package super

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAbsolutePreState_SuperNode(t *testing.T) {
	t.Run("FailedToFetchOutput", func(t *testing.T) {
		rootProvider := &stubSuperNodeRootProvider{}
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		_, err := provider.AbsolutePreState(context.Background())
		require.Error(t, err)
		require.NotErrorIs(t, err, ethereum.NotFound, "The API shouldn't return not found, it returns a response with no data instead")

		_, err = provider.AbsolutePreStateCommitment(context.Background())
		require.Error(t, err)
		require.NotErrorIs(t, err, ethereum.NotFound, "The API shouldn't return not found, it returns a response with no data instead")
	})

	t.Run("NoDataResponse", func(t *testing.T) {
		rootProvider := &stubSuperNodeRootProvider{}
		rootProvider.AddAtTimestamp(100, eth.SuperRootAtTimestampResponse{
			Data: nil,
		})
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		_, err := provider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)

		_, err = provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, ethereum.NotFound)
	})

	t.Run("ReturnsSuperRootForTimestamp", func(t *testing.T) {
		expectedPreimage := eth.NewSuperV1(100,
			eth.ChainIDAndOutput{
				ChainID: eth.ChainID{2987},
				Output:  eth.Bytes32{0x88},
			}, eth.ChainIDAndOutput{
				ChainID: eth.ChainID{100},
				Output:  eth.Bytes32{0x10},
			})
		rootProvider := &stubSuperNodeRootProvider{}
		rootProvider.AddAtTimestamp(100, eth.SuperRootAtTimestampResponse{
			Data: &eth.SuperRootResponseData{
				Super:     expectedPreimage,
				SuperRoot: eth.SuperRoot(expectedPreimage),
			},
		})
		provider := NewSuperNodePrestateProvider(rootProvider, 100)

		preimage, err := provider.AbsolutePreState(context.Background())
		require.NoError(t, err)
		require.Equal(t, expectedPreimage, preimage)

		commitment, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedPreimage)), commitment)
	})
}
