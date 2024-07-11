package backend

import (
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/stretchr/testify/require"
)

func TestRecover(t *testing.T) {
	tests := []struct {
		name             string
		stubDB           *stubLogStore
		expectedBlockNum uint64
		expectRewoundTo  uint64
	}{
		{
			name:             "emptydb",
			stubDB:           &stubLogStore{closestBlockErr: fmt.Errorf("no entries: %w", io.EOF)},
			expectedBlockNum: 0,
			expectRewoundTo:  0,
		},
		{
			name:             "genesis",
			stubDB:           &stubLogStore{},
			expectedBlockNum: 0,
			expectRewoundTo:  0,
		},
		{
			name:             "with_blocks",
			stubDB:           &stubLogStore{closestBlockNumber: 15},
			expectedBlockNum: 14,
			expectRewoundTo:  14,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			block, err := Resume(test.stubDB)
			require.NoError(t, err)
			require.Equal(t, test.expectedBlockNum, block)
			require.Equal(t, test.expectRewoundTo, test.stubDB.rewoundTo)
		})
	}
}

type stubLogStore struct {
	closestBlockNumber uint64
	closestBlockErr    error
	rewoundTo          uint64
}

func (s *stubLogStore) Close() error {
	return nil
}

func (s *stubLogStore) ClosestBlockInfo(blockNum uint64) (uint64, types.TruncatedHash, error) {
	if s.closestBlockErr != nil {
		return 0, types.TruncatedHash{}, s.closestBlockErr
	}
	return s.closestBlockNumber, types.TruncatedHash{}, nil
}

func (s *stubLogStore) Rewind(headBlockNum uint64) error {
	s.rewoundTo = headBlockNum
	return nil
}
