package source_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/source"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	supTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestHeadPointerProcessor_OnNewHead(t *testing.T) {
	chain := supTypes.ChainIDFromUInt64(1111)
	// create a mock storage which stubs out lastLogInBlock
	store := &MockStorage{
		lastLogInBlock: 1234,
	}
	// set up the block to be processed
	block := eth.L1BlockRef{
		Number: 1234,
	}

	t.Run("Unsafe", func(t *testing.T) {
		safety := supTypes.Unsafe
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was updated correctly
		expectedHeads := heads.Heads{
			Chains: map[supTypes.ChainID]heads.ChainHeads{
				chain: {
					Unsafe: 1234,
				},
			},
		}
		require.Equal(t, expectedHeads, store.heads)
	})

	t.Run("Safe", func(t *testing.T) {
		safety := supTypes.Safe
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was updated correctly
		expectedHeads := heads.Heads{
			Chains: map[supTypes.ChainID]heads.ChainHeads{
				chain: {
					LocalSafe: 1234,
				},
			},
		}
		require.Equal(t, expectedHeads, store.heads)
	})

	t.Run("finalized", func(t *testing.T) {
		safety := supTypes.Finalized
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was updated correctly
		expectedHeads := heads.Heads{
			Chains: map[supTypes.ChainID]heads.ChainHeads{
				chain: {
					LocalFinalized: 1234,
				},
			},
		}
		require.Equal(t, expectedHeads, store.heads)
	})
}

// test that the head pointer processor does not update the head storage if there is an error
func TestHeadPointerProcessor_OnNewHeadError(t *testing.T) {
	chain := supTypes.ChainIDFromUInt64(1111)
	// create a mock storage which stubs out lastLogInBlock
	store := &MockStorage{
		lastLogInBlock:    1234,
		lastLogInBlockErr: fmt.Errorf("error"),
	}
	// set up the block to be processed
	block := eth.L1BlockRef{
		Number: 1234,
	}

	t.Run("Unsafe", func(t *testing.T) {
		safety := supTypes.Unsafe
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was not updated
		expectedHeads := heads.Heads{}
		require.Equal(t, expectedHeads, store.heads)
	})

	t.Run("Safe", func(t *testing.T) {
		safety := supTypes.Safe
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was not updated
		expectedHeads := heads.Heads{}
		require.Equal(t, expectedHeads, store.heads)
	})

	t.Run("finalized", func(t *testing.T) {
		safety := supTypes.Finalized
		processor := source.NewHeadPointerProcessor(chain, store, safety)
		ctx := context.Background()
		processor.OnNewHead(ctx, block)
		// check that the head storage was not updated
		expectedHeads := heads.Heads{}
		require.Equal(t, expectedHeads, store.heads)
	})

}

type MockStorage struct {
	lastLogInBlock    entrydb.EntryIdx
	lastLogInBlockErr error
	heads             heads.Heads
}

func (m *MockStorage) LastLogInBlock(chain supTypes.ChainID, blockNumber uint64) (entrydb.EntryIdx, error) {
	if m.lastLogInBlockErr != nil {
		return 0, m.lastLogInBlockErr
	}
	// return the stored value
	return m.lastLogInBlock, nil
}
func (m *MockStorage) Apply(f heads.OperationFn) error {
	empty := heads.Heads{
		Chains: make(map[supTypes.ChainID]heads.ChainHeads),
	}
	f(&empty)
	m.heads = empty
	return nil
}
func (m *MockStorage) AddLog(chain supTypes.ChainID, logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error {
	return nil
}
func (m *MockStorage) LastEntryIdx(chain supTypes.ChainID) entrydb.EntryIdx {
	return 0
}
func (m *MockStorage) LatestBlockNum(chain supTypes.ChainID) uint64 {
	return 0
}
func (m *MockStorage) Rewind(chain supTypes.ChainID, x uint64) error {
	return nil
}
