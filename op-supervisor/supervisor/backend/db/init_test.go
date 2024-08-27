package db

import (
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/stretchr/testify/require"
)

func TestRecover(t *testing.T) {
	tests := []struct {
		name            string
		stubDB          *stubLogStore
		expectRewoundTo uint64
	}{
		{
			name:            "emptydb",
			stubDB:          &stubLogStore{closestBlockErr: fmt.Errorf("no entries: %w", io.EOF)},
			expectRewoundTo: 0,
		},
		{
			name:            "genesis",
			stubDB:          &stubLogStore{},
			expectRewoundTo: 0,
		},
		{
			name:            "with_blocks",
			stubDB:          &stubLogStore{closestBlockNumber: 15},
			expectRewoundTo: 14,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := Resume(test.stubDB)
			require.NoError(t, err)
			require.Equal(t, test.expectRewoundTo, test.stubDB.rewoundTo)
		})
	}
}

type stubLogStore struct {
	closestBlockNumber uint64
	closestBlockErr    error
	rewoundTo          uint64
}

func (s *stubLogStore) Contains(blockNum uint64, logIdx uint32, loghash types.TruncatedHash) (bool, entrydb.EntryIdx, error) {
	panic("not supported")
}

func (s *stubLogStore) ClosestBlockIterator(blockNum uint64) (logs.Iterator, error) {
	panic("not supported")
}

func (s *stubLogStore) LastCheckpointBehind(entrydb.EntryIdx) (logs.Iterator, error) {
	panic("not supported")
}

func (s *stubLogStore) ClosestBlockInfo(blockNum uint64) (uint64, types.TruncatedHash, error) {
	if s.closestBlockErr != nil {
		return 0, types.TruncatedHash{}, s.closestBlockErr
	}
	return s.closestBlockNumber, types.TruncatedHash{}, nil
}

func (s *stubLogStore) NextExecutingMessage(logs.Iterator) (types.ExecutingMessage, error) {
	panic("not supported")
}

func (s *stubLogStore) Rewind(headBlockNum uint64) error {
	s.rewoundTo = headBlockNum
	return nil
}

func (s *stubLogStore) AddLog(logHash types.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *types.ExecutingMessage) error {
	panic("not supported")
}

func (s *stubLogStore) LatestBlockNum() uint64 {
	panic("not supported")
}

func (s *stubLogStore) Close() error {
	return nil
}
