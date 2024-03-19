package source

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestRestrictedOutputLoader(t *testing.T) {
	tests := []struct {
		name        string
		maxSafeHead uint64
		blockNum    uint64
		expectedErr error
	}{
		{
			name:        "GenesisNotRestricted",
			maxSafeHead: 1000,
			blockNum:    0,
			expectedErr: nil,
		},
		{
			name:        "BothAtGenesis",
			maxSafeHead: 0,
			blockNum:    0,
			expectedErr: nil,
		},
		{
			name:        "RestrictedToGenesis",
			maxSafeHead: 0,
			blockNum:    1,
			expectedErr: ErrExceedsL1Head,
		},
		{
			name:        "JustBelowMaxHead",
			maxSafeHead: 1000,
			blockNum:    999,
			expectedErr: nil,
		},
		{
			name:        "EqualMaxHead",
			maxSafeHead: 1000,
			blockNum:    1000,
			expectedErr: nil,
		},
		{
			name:        "JustAboveMaxHead",
			maxSafeHead: 1000,
			blockNum:    1001,
			expectedErr: ErrExceedsL1Head,
		},
		{
			name:        "WellAboveMaxHead",
			maxSafeHead: 1000,
			blockNum:    99001,
			expectedErr: ErrExceedsL1Head,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			loader := NewRestrictedOutputSource(&stubOutputRollupClient{}, test.maxSafeHead)
			result, err := loader.OutputAtBlock(context.Background(), test.blockNum)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, common.Hash{byte(test.blockNum)}, result)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
			}
		})
	}
}

func TestRestrictedOutputLoader_ReturnsError(t *testing.T) {
	expectedErr := errors.New("boom")
	loader := NewRestrictedOutputSource(&stubOutputRollupClient{err: expectedErr}, 6)
	_, err := loader.OutputAtBlock(context.Background(), 4)
	require.ErrorIs(t, err, expectedErr)
}

type stubOutputRollupClient struct {
	err error
}

func (s *stubOutputRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &eth.OutputResponse{
		OutputRoot: eth.Bytes32{byte(blockNum)},
	}, nil
}
