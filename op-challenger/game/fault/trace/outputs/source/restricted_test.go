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
			l1Head := eth.BlockID{Number: 3428}
			rollupClient := &stubOutputRollupClient{
				safeHead: test.maxSafeHead,
			}
			loader := NewRestrictedOutputSource(rollupClient, l1Head)
			result, err := loader.OutputAtBlock(context.Background(), test.blockNum)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, common.Hash{byte(test.blockNum)}, result)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
			}
			require.Equal(t, l1Head.Number, rollupClient.requestedL1BlockNum)
		})
	}
}

func TestRestrictedOutputLoader_GetOutputRootErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	client := &stubOutputRollupClient{outputErr: expectedErr, safeHead: 884}
	loader := NewRestrictedOutputSource(client, eth.BlockID{Number: 1234})
	_, err := loader.OutputAtBlock(context.Background(), 4)
	require.ErrorIs(t, err, expectedErr)
}

func TestRestrictedOutputLoader_SafeHeadAtL1BlockErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	client := &stubOutputRollupClient{safeHeadErr: expectedErr, safeHead: 884}
	loader := NewRestrictedOutputSource(client, eth.BlockID{Number: 1234})
	_, err := loader.OutputAtBlock(context.Background(), 4)
	require.ErrorIs(t, err, expectedErr)
}

type stubOutputRollupClient struct {
	outputErr           error
	safeHeadErr         error
	safeHead            uint64
	requestedL1BlockNum uint64
}

func (s *stubOutputRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	if s.outputErr != nil {
		return nil, s.outputErr
	}
	return &eth.OutputResponse{
		OutputRoot: eth.Bytes32{byte(blockNum)},
	}, nil
}

func (s *stubOutputRollupClient) SafeHeadAtL1Block(_ context.Context, l1BlockNum uint64) (*eth.SafeHeadResponse, error) {
	s.requestedL1BlockNum = l1BlockNum
	if s.safeHeadErr != nil {
		return nil, s.safeHeadErr
	}
	return &eth.SafeHeadResponse{
		L1Block: eth.BlockID{
			Hash:   common.Hash{0x11},
			Number: 4824,
		},
		SafeHead: eth.BlockID{
			Hash:   common.Hash{0x22},
			Number: s.safeHead,
		},
	}, nil
}
