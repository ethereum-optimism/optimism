package db

import (
	"errors"
	"io"
	"math/rand" // nosemgrep
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestChainsDB_AddLog(t *testing.T) {
	t.Run("UnknownChain", func(t *testing.T) {
		db := NewChainsDB(nil, &stubHeadStorage{}, testlog.Logger(t, log.LevelDebug))
		err := db.AddLog(types.ChainIDFromUInt64(2), common.Hash{}, eth.BlockID{}, 33, nil)
		require.ErrorIs(t, err, ErrUnknownChain)
	})

	t.Run("KnownChain", func(t *testing.T) {
		chainID := types.ChainIDFromUInt64(1)
		logDB := &stubLogDB{}
		db := NewChainsDB(map[types.ChainID]LogStorage{
			chainID: logDB,
		}, &stubHeadStorage{}, testlog.Logger(t, log.LevelDebug))
		bl10 := eth.BlockID{Hash: common.Hash{0x10}, Number: 10}
		err := db.SealBlock(chainID, common.Hash{0x9}, bl10, 1234)
		require.NoError(t, err, err)
		err = db.AddLog(chainID, common.Hash{}, bl10, 0, nil)
		require.NoError(t, err, err)
		require.Equal(t, 1, logDB.addLogCalls)
		require.Equal(t, 1, logDB.sealBlockCalls)
	})
}

func TestChainsDB_Rewind(t *testing.T) {
	t.Run("UnknownChain", func(t *testing.T) {
		db := NewChainsDB(nil, &stubHeadStorage{}, testlog.Logger(t, log.LevelDebug))
		err := db.Rewind(types.ChainIDFromUInt64(2), 42)
		require.ErrorIs(t, err, ErrUnknownChain)
	})

	t.Run("KnownChain", func(t *testing.T) {
		chainID := types.ChainIDFromUInt64(1)
		logDB := &stubLogDB{}
		db := NewChainsDB(map[types.ChainID]LogStorage{
			chainID: logDB,
		}, &stubHeadStorage{},
			testlog.Logger(t, log.LevelDebug))
		err := db.Rewind(chainID, 23)
		require.NoError(t, err, err)
		require.EqualValues(t, 23, logDB.headBlockNum)
	})
}

func TestChainsDB_UpdateCrossHeads(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)

	checker.numSafe = 1
	xSafe := checker.crossHeadForChain

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h},
		testlog.Logger(t, log.LevelDebug))

	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	// found a safe executing message, and no new initiating messages
	require.Equal(t, xSafe+1, checker.updated)
}

func TestChainsDB_UpdateCrossHeadsBeyondLocal(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the safety checker to pass 99 times, effectively allowing all messages to be safe
	checker.numSafe = 99

	startLocalSafe := checker.localHeadForChain

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h},
		testlog.Logger(t, log.LevelDebug))

	// Update cross-heads is expected to:
	// 1. get a last checkpoint iterator from the logDB (stubbed to be at 15)
	// 2. progress the iterator to repeatedly, as the safety check will pass 99 times.
	// 3. exceed the local head, and update the cross-head to the local head (40)
	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	require.Equal(t, startLocalSafe, checker.updated)
}

func TestChainsDB_UpdateCrossHeadsEOF(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the log DB to return an EOF error when trying to get the next executing message
	// after processing 10 message (with more messages available to be safe)
	logDB.nextLogs = logDB.nextLogs[:checker.crossHeadForChain+11]
	// This is a legacy test, the local head is further than the DB content...

	checker.numSafe = 99

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h},
		testlog.Logger(t, log.LevelDebug))

	// Update cross-heads is expected to:
	// - process 10 logs as safe, 5 of which execute something
	// - update cross-safe to what was there
	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	require.Equal(t, checker.crossHeadForChain+11, checker.updated)
}

func TestChainsDB_UpdateCrossHeadsError(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the log DB to return an error when trying to get the next executing message
	// after processing 3 messages as safe (with more messages available to be safe)

	executed := 0
	for i, e := range logDB.nextLogs {
		if executed == 3 {
			logDB.nextLogs[i].err = errors.New("some error")
		}
		if entrydb.EntryIdx(i) > checker.crossHeadForChain && e.execIdx >= 0 {
			executed++
		}
	}

	// everything is safe until error
	checker.numSafe = 99

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h},
		testlog.Logger(t, log.LevelDebug))

	// Update cross-heads is expected to:
	// 1. get a last checkpoint iterator from the logDB (stubbed to be at 10)
	// 2. fail during execution, even after processing 3 messages as safe
	// 3. exit without updating, returning the error
	err := db.UpdateCrossHeads(checker)
	require.Error(t, err)
	// the update was never set (aka 0-value)
	require.Equal(t, entrydb.EntryIdx(0), checker.updated)
}

// setupStubbedForUpdateHeads sets up stubbed components for testing the UpdateCrossHeads method
// it returns stubbed structs which are suitable for their interfaces, and can be modified before testing
// TODO: the variables at the top of this function should be configurable by the caller.
// this isn't an issue for now, as all tests can modify the stubbed components directly after calling this function.
// but readability and maintainability would be improved by making this function more configurable.
func setupStubbedForUpdateHeads(chainID types.ChainID) (*stubLogDB, *stubChecker, *heads.Heads) {
	// the last known cross-safe head is at 20
	cross := entrydb.EntryIdx(20)
	// the local head (the limit of the update) is at 40
	local := entrydb.EntryIdx(40)
	// the number of executing messages to make available (this should be more than the number of safety checks performed)
	numExecutingMessages := 30
	// number of safety checks that will pass before returning false
	numSafe := 1

	// set up stubbed logDB
	logDB := &stubLogDB{}

	// set up stubbed executing messages that the ChainsDB can pass to the checker
	logDB.executingMessages = []*types.ExecutingMessage{}
	for i := 0; i < numExecutingMessages; i++ {
		// executing messages are packed in groups of 3, with block numbers increasing by 1
		logDB.executingMessages = append(logDB.executingMessages, &types.ExecutingMessage{
			BlockNum: uint64(100 + int(i/3)),
			LogIdx:   uint32(i),
			Hash:     common.Hash{},
		})
	}

	rng := rand.New(rand.NewSource(123))
	blockNum := uint64(100)
	logIndex := uint32(0)
	executedCount := 0
	for i := entrydb.EntryIdx(0); i <= local; i++ {
		var logHash common.Hash
		rng.Read(logHash[:])

		execIndex := -1
		// All the even messages have an executing message
		if i%2 == 0 {
			execIndex = rng.Intn(len(logDB.executingMessages))
			executedCount += 1
		}
		var msgErr error

		logDB.nextLogs = append(logDB.nextLogs, nextLogResponse{
			blockNum: blockNum,
			logIdx:   logIndex,
			evtHash:  logHash,
			err:      msgErr,
			execIdx:  execIndex,
		})
	}

	// set up stubbed checker
	checker := &stubChecker{
		localHeadForChain: local,
		crossHeadForChain: cross,
		// the first safety check will return true, the second false
		numSafe: numSafe,
	}

	// set up stubbed heads with sample values
	h := heads.NewHeads()
	h.Chains[chainID] = heads.ChainHeads{}

	return logDB, checker, h
}

type stubChecker struct {
	localHeadForChain entrydb.EntryIdx
	crossHeadForChain entrydb.EntryIdx
	numSafe           int
	checkCalls        int
	updated           entrydb.EntryIdx
}

func (s *stubChecker) LocalHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	return s.localHeadForChain
}

func (s *stubChecker) Name() string {
	return "stubChecker"
}

func (s *stubChecker) CrossHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	return s.crossHeadForChain
}

// stubbed Check returns true for the first numSafe calls, and false thereafter
func (s *stubChecker) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) bool {
	if s.checkCalls >= s.numSafe {
		return false
	}
	s.checkCalls++
	return true
}

func (s *stubChecker) Update(chain types.ChainID, index entrydb.EntryIdx) heads.OperationFn {
	s.updated = index
	return func(heads *heads.Heads) error {
		return nil
	}
}

func (s *stubChecker) SafetyLevel() types.SafetyLevel {
	return types.CrossSafe
}

type stubHeadStorage struct {
	heads *heads.Heads
}

func (s *stubHeadStorage) Apply(heads.Operation) error {
	return nil
}

func (s *stubHeadStorage) Current() *heads.Heads {
	if s.heads == nil {
		s.heads = heads.NewHeads()
	}
	return s.heads.Copy()
}

type nextLogResponse struct {
	blockNum uint64

	logIdx uint32

	evtHash common.Hash

	err error

	// -1 if not executing
	execIdx int
}

type stubIterator struct {
	index entrydb.EntryIdx

	db *stubLogDB
}

func (s *stubIterator) End() error {
	return nil // only used for DB-loading. The stub is already loaded
}

func (s *stubIterator) NextInitMsg() error {
	s.index += 1
	if s.index >= entrydb.EntryIdx(len(s.db.nextLogs)) {
		return io.EOF
	}
	e := s.db.nextLogs[s.index]
	return e.err
}

func (s *stubIterator) NextExecMsg() error {
	for {
		s.index += 1
		if s.index >= entrydb.EntryIdx(len(s.db.nextLogs)) {
			return io.EOF
		}
		e := s.db.nextLogs[s.index]
		if e.err != nil {
			return e.err
		}
		if e.execIdx >= 0 {
			return nil
		}
	}
}

func (s *stubIterator) NextBlock() error {
	panic("not yet supported")
}

func (s *stubIterator) NextIndex() entrydb.EntryIdx {
	return s.index + 1
}

func (s *stubIterator) SealedBlock() (hash common.Hash, num uint64, ok bool) {
	panic("not yet supported")
}

func (s *stubIterator) InitMessage() (hash common.Hash, logIndex uint32, ok bool) {
	if s.index < 0 {
		return common.Hash{}, 0, false
	}
	if s.index >= entrydb.EntryIdx(len(s.db.nextLogs)) {
		return common.Hash{}, 0, false
	}
	e := s.db.nextLogs[s.index]
	return e.evtHash, e.logIdx, true
}

func (s *stubIterator) ExecMessage() *types.ExecutingMessage {
	if s.index < 0 {
		return nil
	}
	if s.index >= entrydb.EntryIdx(len(s.db.nextLogs)) {
		return nil
	}
	e := s.db.nextLogs[s.index]
	if e.execIdx < 0 {
		return nil
	}
	return s.db.executingMessages[e.execIdx]
}

var _ logs.Iterator = (*stubIterator)(nil)

type stubLogDB struct {
	addLogCalls    int
	sealBlockCalls int
	headBlockNum   uint64

	executingMessages []*types.ExecutingMessage
	nextLogs          []nextLogResponse

	containsResponse containsResponse
}

func (s *stubLogDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	s.addLogCalls++
	return nil
}

func (s *stubLogDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	s.sealBlockCalls++
	return nil
}

func (s *stubLogDB) LatestSealedBlockNum() (n uint64, ok bool) {
	return s.headBlockNum, true
}

func (s *stubLogDB) FindSealedBlock(block eth.BlockID) (nextEntry entrydb.EntryIdx, err error) {
	panic("not implemented")
}

func (s *stubLogDB) IteratorStartingAt(i entrydb.EntryIdx) (logs.Iterator, error) {
	return &stubIterator{
		index: i - 1,
		db:    s,
	}, nil
}

var _ LogStorage = (*stubLogDB)(nil)

type containsResponse struct {
	index entrydb.EntryIdx
	err   error
}

// stubbed Contains records the arguments passed to it
// it returns the response set in the struct, or an empty response
func (s *stubLogDB) Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (nextIndex entrydb.EntryIdx, err error) {
	return s.containsResponse.index, s.containsResponse.err
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
