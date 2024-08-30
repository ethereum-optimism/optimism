package db

import (
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
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

func TestChainsDB_LastLogInBlock(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, _, h := setupStubbedForUpdateHeads(chainID)
	logDB.nextLogs = []nextLogResponse{
		{10, 1, backendTypes.TruncatedHash{}, nil},
		{10, 2, backendTypes.TruncatedHash{}, nil},
		{10, 3, backendTypes.TruncatedHash{}, nil},
		{10, 4, backendTypes.TruncatedHash{}, nil},
		{11, 5, backendTypes.TruncatedHash{}, nil},
	}

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// LastLogInBlock is expected to:
	// 1. get a block iterator for block 10 (stubbed)
	// 2. scan through the iterator until the block number exceeds the target (10)
	// 3. return the index of the last log in the block (4)
	index, err := db.LastLogInBlock(chainID, 10)
	require.NoError(t, err)
	require.Equal(t, entrydb.EntryIdx(4), index)
}

func TestChainsDB_LastLogInBlockEOF(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, _, h := setupStubbedForUpdateHeads(chainID)
	logDB.nextLogs = []nextLogResponse{
		{10, 5, backendTypes.TruncatedHash{}, nil},
		{10, 6, backendTypes.TruncatedHash{}, nil},
		{10, 7, backendTypes.TruncatedHash{}, nil},
		{10, 8, backendTypes.TruncatedHash{}, nil},
		{10, 9, backendTypes.TruncatedHash{}, nil},
		{10, 10, backendTypes.TruncatedHash{}, nil},
	}

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// LastLogInBlock is expected to:
	// 1. get a block iterator for block 10 (stubbed)
	// 2. scan through the iterator and never find the target block
	// return an error
	index, err := db.LastLogInBlock(chainID, 10)
	require.NoError(t, err)
	require.Equal(t, entrydb.EntryIdx(10), index)
}

func TestChainsDB_LastLogInBlockNotFound(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, _, h := setupStubbedForUpdateHeads(chainID)
	logDB.nextLogs = []nextLogResponse{
		{100, 5, backendTypes.TruncatedHash{}, nil},
		{100, 6, backendTypes.TruncatedHash{}, nil},
		{100, 7, backendTypes.TruncatedHash{}, nil},
		{101, 8, backendTypes.TruncatedHash{}, nil},
		{101, 9, backendTypes.TruncatedHash{}, nil},
		{101, 10, backendTypes.TruncatedHash{}, nil},
	}

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// LastLogInBlock is expected to:
	// 1. get a block iterator for block 10 (stubbed)
	// 2. scan through the iterator and never find the target block
	// return an error
	_, err := db.LastLogInBlock(chainID, 10)
	require.ErrorContains(t, err, "block 10 not found")
}

func TestChainsDB_LastLogInBlockError(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, _, h := setupStubbedForUpdateHeads(chainID)
	logDB.nextLogs = []nextLogResponse{
		{10, 1, backendTypes.TruncatedHash{}, nil},
		{10, 2, backendTypes.TruncatedHash{}, nil},
		{10, 3, backendTypes.TruncatedHash{}, nil},
		{0, 0, backendTypes.TruncatedHash{}, fmt.Errorf("some error")},
		{11, 5, backendTypes.TruncatedHash{}, nil},
	}

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// LastLogInBlock is expected to:
	// 1. get a block iterator for block 10 (stubbed)
	// 2. scan through the iterator and encounter an error
	// return an error
	_, err := db.LastLogInBlock(chainID, 10)
	require.ErrorContains(t, err, "some error")
}

func TestChainsDB_UpdateCrossHeads(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// Update cross-heads is expected to:
	// 1. get a last checkpoint iterator from the logDB (stubbed to be at 15)
	// 2. progress the iterator to the next log (16) because the first safety check will pass
	// 3. fail the second safety check
	// 4. update the cross-heads to the last successful safety check (16)
	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	require.Equal(t, entrydb.EntryIdx(16), checker.updated)
}

func TestChainsDB_UpdateCrossHeadsBeyondLocal(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the safety checker to pass 99 times, effeciively allowing all messages to be safe
	checker.numSafe = 99

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// Update cross-heads is expected to:
	// 1. get a last checkpoint iterator from the logDB (stubbed to be at 15)
	// 2. progress the iterator to repeatedly, as the safety check will pass 99 times.
	// 3. exceed the local head, and update the cross-head to the local head (40)
	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	require.Equal(t, entrydb.EntryIdx(40), checker.updated)
}

func TestChainsDB_UpdateCrossHeadsEOF(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the log DB to return an EOF error when trying to get the next executing message
	// after processing 10 messages as safe (with more messages available to be safe)
	logDB.errOverload = io.EOF
	logDB.errAfter = 10
	checker.numSafe = 99

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

	// Update cross-heads is expected to:
	// 1. get a last checkpoint iterator from the logDB (stubbed to be at 15)
	// 2. after processing 10 messages as safe, fail to find any executing messages (EOF)
	// 3. update to the last successful safety check (25) without returning an error
	err := db.UpdateCrossHeads(checker)
	require.NoError(t, err)
	require.Equal(t, entrydb.EntryIdx(25), checker.updated)
}

func TestChainsDB_UpdateCrossHeadsError(t *testing.T) {
	// using a chainID of 1 for simplicity
	chainID := types.ChainIDFromUInt64(1)
	// get default stubbed components
	logDB, checker, h := setupStubbedForUpdateHeads(chainID)
	// set the log DB to return an error when trying to get the next executing message
	// after processing 3 messages as safe (with more messages available to be safe)
	logDB.errOverload = fmt.Errorf("some error")
	logDB.errAfter = 3
	checker.numSafe = 99

	// The ChainsDB is real, but uses only stubbed components
	db := NewChainsDB(
		map[types.ChainID]LogStorage{
			chainID: logDB},
		&stubHeadStorage{h})

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
	// the checkpoint starts somewhere behind the last known cross-safe head
	checkpoint := entrydb.EntryIdx(15)
	// the last known cross-safe head is at 20
	cross := entrydb.EntryIdx(20)
	// the local head (the limit of the update) is at 40
	local := entrydb.EntryIdx(40)
	// the number of executing messages to make available (this should be more than the number of safety checks performed)
	numExecutingMessages := 30
	// number of safety checks that will pass before returning false
	numSafe := 1
	// number of calls to nextExecutingMessage before potentially returning an error
	errAfter := 4

	// set up stubbed logDB
	logDB := &stubLogDB{}
	// the log DB will start the iterator at the checkpoint index
	logDB.lastCheckpointBehind = &stubIterator{checkpoint, 0, nil}
	// rig the log DB to return an error after a certain number of calls to NextExecutingMessage
	logDB.errAfter = errAfter
	// set up stubbed executing messages that the ChainsDB can pass to the checker
	logDB.executingMessages = []*backendTypes.ExecutingMessage{}
	for i := 0; i < numExecutingMessages; i++ {
		// executing messages are packed in groups of 3, with block numbers increasing by 1
		logDB.executingMessages = append(logDB.executingMessages, &backendTypes.ExecutingMessage{
			BlockNum: uint64(100 + int(i/3)),
			LogIdx:   uint32(i),
			Hash:     backendTypes.TruncatedHash{},
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
func (s *stubChecker) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) bool {
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
	logIdx   uint32
	evtHash  backendTypes.TruncatedHash
	err      error
}
type stubIterator struct {
	index        entrydb.EntryIdx
	nextLogIndex int
	nextLogs     []nextLogResponse
}

func (s *stubIterator) NextLog() (uint64, uint32, backendTypes.TruncatedHash, error) {
	if s.nextLogIndex >= len(s.nextLogs) {
		return 0, 0, backendTypes.TruncatedHash{}, io.EOF
	}
	r := s.nextLogs[s.nextLogIndex]
	s.nextLogIndex++
	return r.blockNum, r.logIdx, r.evtHash, r.err
}

func (s *stubIterator) Index() entrydb.EntryIdx {
	return s.index
}
func (s *stubIterator) ExecMessage() (backendTypes.ExecutingMessage, error) {
	panic("not implemented")
}

type stubLogDB struct {
	addLogCalls          int
	headBlockNum         uint64
	emIndex              int
	executingMessages    []*backendTypes.ExecutingMessage
	nextLogs             []nextLogResponse
	lastCheckpointBehind *stubIterator
	errOverload          error
	errAfter             int
	containsResponse     containsResponse
}

// stubbed LastCheckpointBehind returns a stubbed iterator which was passed in to the struct
func (s *stubLogDB) LastCheckpointBehind(entrydb.EntryIdx) (logs.Iterator, error) {
	return s.lastCheckpointBehind, nil
}

func (s *stubLogDB) ClosestBlockIterator(blockNum uint64) (logs.Iterator, error) {
	return &stubIterator{
		index:    entrydb.EntryIdx(99),
		nextLogs: s.nextLogs,
	}, nil
}

func (s *stubLogDB) NextExecutingMessage(i logs.Iterator) (backendTypes.ExecutingMessage, error) {
	// if error overload is set, return it to simulate a failure condition
	if s.errOverload != nil && s.emIndex >= s.errAfter {
		return backendTypes.ExecutingMessage{}, s.errOverload
	}
	// increment the iterator to mark advancement
	i.(*stubIterator).index += 1
	// return the next executing message
	m := *s.executingMessages[s.emIndex]
	// and increment to the next message for the next call
	s.emIndex++
	return m, nil
}

func (s *stubLogDB) ClosestBlockInfo(_ uint64) (uint64, backendTypes.TruncatedHash, error) {
	panic("not implemented")
}

func (s *stubLogDB) AddLog(logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error {
	s.addLogCalls++
	return nil
}

type containsResponse struct {
	contains bool
	index    entrydb.EntryIdx
	err      error
}

// stubbed Contains records the arguments passed to it
// it returns the response set in the struct, or an empty response
func (s *stubLogDB) Contains(blockNum uint64, logIdx uint32, loghash backendTypes.TruncatedHash) (bool, entrydb.EntryIdx, error) {
	return s.containsResponse.contains, s.containsResponse.index, s.containsResponse.err
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
