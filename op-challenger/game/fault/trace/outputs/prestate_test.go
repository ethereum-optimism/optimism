package outputs

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func newOutputPrestateProvider(t *testing.T, prestateBlock uint64) (*OutputPrestateProvider, *stubRollupClient) {
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
	}
	return &OutputPrestateProvider{
		rollupClient:  rollupClient,
		prestateBlock: prestateBlock,
	}, rollupClient
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	var prestateBlock = uint64(100)

	t.Run("FailedToFetchOutput", func(t *testing.T) {
		provider, rollupClient := newOutputPrestateProvider(t, prestateBlock)
		rollupClient.errorsOnPrestateFetch = true
		_, err := provider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, errNoOutputAtBlock)
	})

	t.Run("ReturnsCorrectPrestateOutput", func(t *testing.T) {
		provider, _ := newOutputPrestateProvider(t, prestateBlock)
		value, err := provider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, value, prestateOutputRoot)
	})
}
