package db

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestChainsDB_AddLog(t *testing.T) {
	t.Run("UnknownChain", func(t *testing.T) {
		db := NewChainsDB(nil, &stubHeadStorage{})
		err := db.AddLog(types.ChainIDFromUInt64(2), backendTypes.TruncatedHash{}, eth.BlockID{}, 1234, 33, nil)
		require.ErrorIs(t, err, ErrUnknownChain)
	})

	t.Run("KnownChain", func(t *testing.T) {
		chainID := types.ChainIDFromUInt64(1)
		logDB := &stubLogDB{}
		db := NewChainsDB(map[types.ChainID]LogStorage{
			chainID: logDB,
		}, &stubHeadStorage{})
		err := db.AddLog(chainID, backendTypes.TruncatedHash{}, eth.BlockID{}, 1234, 33, nil)
		require.NoError(t, err, err)
		require.Equal(t, 1, logDB.addLogCalls)
	})
}

func TestChainsDB_Rewind(t *testing.T) {
	t.Run("UnknownChain", func(t *testing.T) {
		db := NewChainsDB(nil, &stubHeadStorage{})
		err := db.Rewind(types.ChainIDFromUInt64(2), 42)
		require.ErrorIs(t, err, ErrUnknownChain)
	})

	t.Run("KnownChain", func(t *testing.T) {
		chainID := types.ChainIDFromUInt64(1)
		logDB := &stubLogDB{}
		db := NewChainsDB(map[types.ChainID]LogStorage{
			chainID: logDB,
		}, &stubHeadStorage{})
		err := db.Rewind(chainID, 23)
		require.NoError(t, err, err)
		require.EqualValues(t, 23, logDB.headBlockNum)
	})
}

type stubHeadStorage struct{}

type stubLogDB struct {
	addLogCalls  int
	headBlockNum uint64
}

func (s *stubLogDB) ClosestBlockInfo(_ uint64) (uint64, backendTypes.TruncatedHash, error) {
	panic("not implemented")
}

func (s *stubLogDB) AddLog(logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error {
	s.addLogCalls++
	return nil
}

func (s *stubLogDB) Rewind(newHeadBlockNum uint64) error {
	s.headBlockNum = newHeadBlockNum
	return nil
}

func (s *stubLogDB) LatestBlockNum() uint64 {
	return s.headBlockNum
}

func (s *stubLogDB) Close() error {
	return nil
}
