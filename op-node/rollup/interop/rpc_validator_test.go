package interop

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// ---- mocks ----

type fakeL2 struct {
	receipts types.Receipts
	err      error
}

func (f *fakeL2) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return nil, f.receipts, f.err
}

type fakeRemote struct {
	logs       []types.Log
	headers    map[common.Hash]*types.Header
	filterErr  error
	failUntil  int32 // atomic: number of FilterLogs calls that should fail
	callCount  int32
	headerErr  error
}

func (f *fakeRemote) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	atomic.AddInt32(&f.callCount, 1)
	if cur := atomic.LoadInt32(&f.failUntil); cur > 0 {
		atomic.AddInt32(&f.failUntil, -1)
		return nil, errors.New("transient rpc error")
	}
	if f.filterErr != nil {
		return nil, f.filterErr
	}
	out := []types.Log{}
	for _, l := range f.logs {
		if q.FromBlock != nil && l.BlockNumber < q.FromBlock.Uint64() {
			continue
		}
		if q.ToBlock != nil && l.BlockNumber > q.ToBlock.Uint64() {
			continue
		}
		if len(q.Addresses) > 0 {
			match := false
			for _, a := range q.Addresses {
				if l.Address == a {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeRemote) HeaderByHash(ctx context.Context, h common.Hash) (*types.Header, error) {
	if f.headerErr != nil {
		return nil, f.headerErr
	}
	hdr, ok := f.headers[h]
	if !ok {
		return nil, errors.New("not found")
	}
	return hdr, nil
}

// ---- helpers ----

// makeExecutingMessageLog builds a well-formed ExecutingMessage event log as
// it would be emitted by CrossL2Inbox.validateMessage, pointing at the given
// remote initiating log.
func makeExecutingMessageLog(t *testing.T, initiating *types.Log, remoteChain eth.ChainID, blockTime uint64) *types.Log {
	t.Helper()
	payload := supervisortypes.LogToMessagePayload(initiating)
	payloadHash := crypto.Keccak256Hash(payload)

	// Identifier: origin (32b, left-padded), blockNumber (32b big-endian),
	// logIndex (32b big-endian), timestamp (32b big-endian), chainID (32b).
	data := make([]byte, 0, 32*5)
	addrPad := make([]byte, 12)
	data = append(data, addrPad...)
	data = append(data, initiating.Address.Bytes()...)

	bnPad := make([]byte, 24)
	data = append(data, bnPad...)
	data = binary.BigEndian.AppendUint64(data, initiating.BlockNumber)

	liPad := make([]byte, 28)
	data = append(data, liPad...)
	data = binary.BigEndian.AppendUint32(data, uint32(initiating.Index))

	tsPad := make([]byte, 24)
	data = append(data, tsPad...)
	data = binary.BigEndian.AppendUint64(data, blockTime)

	chainIDBytes := remoteChain.Bytes32()
	data = append(data, chainIDBytes[:]...)

	return &types.Log{
		Address: gethparams.InteropCrossL2InboxAddress,
		Topics:  []common.Hash{supervisortypes.ExecutingMessageEventTopic, payloadHash},
		Data:    data,
	}
}

func newEnvelope(blockTime uint64) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash: common.HexToHash("0xdead"),
			Timestamp: eth.Uint64Quantity(blockTime),
		},
	}
}

func newValidator(t *testing.T, l2 L2ReceiptsSource, clients map[eth.ChainID]remoteClient) *RPCExecMsgValidator {
	t.Helper()
	interopTime := uint64(0)
	cfg := &rollup.Config{InteropTime: &interopTime}
	return &RPCExecMsgValidator{
		log:        testlog.Logger(t, log.LvlDebug),
		rollupCfg:  cfg,
		l2:         l2,
		clients:    clients,
		timeout:    2 * time.Second,
		retryDelay: 5 * time.Millisecond,
	}
}

// ---- config parser tests ----

func TestParseRPCOverrides(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantLen int
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"whitespace", "   ", 0, false},
		{"one", "10=https://a", 1, false},
		{"two", "10=https://a,8453=https://b", 2, false},
		{"spaces", " 10 = https://a , 8453 = https://b ", 2, false},
		{"bad pair", "10", 0, true},
		{"bad chainID", "abc=https://a", 0, true},
		{"empty URL", "10=", 0, true},
		{"dup chain", "10=a,10=b", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ParseRPCOverrides(c.in)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, out, c.wantLen)
		})
	}
}

// ---- validation tests ----

func TestValidate_HappyPath(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	remoteBlockTime := uint64(1000)
	execBlockTime := remoteBlockTime + 5

	initLog := &types.Log{
		Address:     common.HexToAddress("0xabc"),
		Topics:      []common.Hash{common.HexToHash("0x1")},
		Data:        []byte("hello"),
		BlockNumber: 42,
		BlockHash:   common.HexToHash("0xbeef"),
		Index:       7,
	}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, remoteBlockTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{
		logs:    []types.Log{*initLog},
		headers: map[common.Hash]*types.Header{initLog.BlockHash: {Time: remoteBlockTime, Number: big.NewInt(42)}},
	}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	require.NoError(t, v.Validate(context.Background(), newEnvelope(execBlockTime)))
}

func TestValidate_NoExecutingMessages(t *testing.T) {
	// Receipts with logs that don't match the Inbox or the topic: validator is a no-op.
	unrelated := &types.Log{Address: common.HexToAddress("0xcafe"), Topics: []common.Hash{common.HexToHash("0xbeef")}}
	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{unrelated}}}}
	rem := &fakeRemote{}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{eth.ChainIDFromUInt64(10): rem})

	require.NoError(t, v.Validate(context.Background(), newEnvelope(1000)))
	require.EqualValues(t, 0, atomic.LoadInt32(&rem.callCount), "no RPC call expected on fast-path")
}

func TestValidate_PreInteropSkip(t *testing.T) {
	// Interop activates at time 2000; block timestamp 1500 → skip entirely.
	interopAt := uint64(2000)
	cfg := &rollup.Config{InteropTime: &interopAt}
	v := &RPCExecMsgValidator{
		log:       testlog.Logger(t, log.LvlDebug),
		rollupCfg: cfg,
		// Deliberately nil l2 and clients: we must never touch them.
		timeout: time.Second,
	}
	require.NoError(t, v.Validate(context.Background(), newEnvelope(1500)))
}

func TestValidate_MissingRemoteLog(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, 1000)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	// Remote returns no matching logs at all.
	rem := &fakeRemote{logs: nil}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	err := v.Validate(context.Background(), newEnvelope(1005))
	require.Error(t, err)
	require.Contains(t, err.Error(), "remote log not found")
}

func TestValidate_UnknownChain(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, 1000)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	// Configure a *different* chain's RPC than the one the message references.
	rem := &fakeRemote{}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{eth.ChainIDFromUInt64(8453): rem})

	err := v.Validate(context.Background(), newEnvelope(1005))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no RPC configured")
}

func TestValidate_TimestampInversion(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	// Initiator timestamp > executor block timestamp — invariant violation.
	initiatorTime := uint64(2000)
	execTime := uint64(1000)

	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, initiatorTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	err := v.Validate(context.Background(), newEnvelope(execTime))
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds exec block timestamp")
	require.EqualValues(t, 0, atomic.LoadInt32(&rem.callCount), "remote RPC must not be called on obvious violations")
}

func TestValidate_Expired(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	initiatorTime := uint64(1000)
	// Way past the 7-day expiry window.
	execTime := initiatorTime + 8*24*3600

	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, initiatorTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	err := v.Validate(context.Background(), newEnvelope(execTime))
	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")
}

func TestValidate_PayloadHashMismatch(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	remoteBlockTime := uint64(1000)

	initLog := &types.Log{
		Address:     common.HexToAddress("0xabc"),
		Topics:      []common.Hash{common.HexToHash("0x1")},
		Data:        []byte("hello"),
		BlockNumber: 42, BlockHash: common.HexToHash("0xbeef"), Index: 7,
	}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, remoteBlockTime)

	// Remote returns a *different* log at the same (origin, block, index).
	tamperedLog := *initLog
	tamperedLog.Data = []byte("tampered")

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{
		logs:    []types.Log{tamperedLog},
		headers: map[common.Hash]*types.Header{initLog.BlockHash: {Time: remoteBlockTime, Number: big.NewInt(42)}},
	}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	err := v.Validate(context.Background(), newEnvelope(remoteBlockTime+5))
	require.Error(t, err)
	require.Contains(t, err.Error(), "payload hash mismatch")
}

func TestValidate_RemoteBlockTimestampMismatch(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	remoteBlockTime := uint64(1000)

	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, BlockHash: common.HexToHash("0xbeef"), Index: 7}
	// Claim the identifier timestamp is remoteBlockTime, but the remote header returns a different time.
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, remoteBlockTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{
		logs:    []types.Log{*initLog},
		headers: map[common.Hash]*types.Header{initLog.BlockHash: {Time: remoteBlockTime + 99, Number: big.NewInt(42)}},
	}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	err := v.Validate(context.Background(), newEnvelope(remoteBlockTime+5))
	require.Error(t, err)
	require.Contains(t, err.Error(), "remote block timestamp")
}

func TestValidate_RetryThenSucceed(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	remoteBlockTime := uint64(1000)

	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, Data: []byte("hi"), BlockNumber: 42, BlockHash: common.HexToHash("0xbeef"), Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, remoteBlockTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	rem := &fakeRemote{
		logs:    []types.Log{*initLog},
		headers: map[common.Hash]*types.Header{initLog.BlockHash: {Time: remoteBlockTime, Number: big.NewInt(42)}},
		// First 3 calls fail, 4th succeeds.
		failUntil: 3,
	}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})

	require.NoError(t, v.Validate(context.Background(), newEnvelope(remoteBlockTime+5)))
	require.GreaterOrEqual(t, atomic.LoadInt32(&rem.callCount), int32(4))
}

func TestValidate_TimeoutExceeded(t *testing.T) {
	remoteChain := eth.ChainIDFromUInt64(10)
	remoteBlockTime := uint64(1000)

	initLog := &types.Log{Address: common.HexToAddress("0xabc"), Topics: []common.Hash{common.HexToHash("0x1")}, BlockNumber: 42, BlockHash: common.HexToHash("0xbeef"), Index: 7}
	execLog := makeExecutingMessageLog(t, initLog, remoteChain, remoteBlockTime)

	l2 := &fakeL2{receipts: types.Receipts{&types.Receipt{Logs: []*types.Log{execLog}}}}
	// Remote permanently errors; validator should give up after timeout.
	rem := &fakeRemote{filterErr: errors.New("permanent outage")}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{remoteChain: rem})
	v.timeout = 50 * time.Millisecond
	v.retryDelay = 5 * time.Millisecond

	start := time.Now()
	err := v.Validate(context.Background(), newEnvelope(remoteBlockTime+5))
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Contains(t, err.Error(), "validation budget")
	require.Less(t, elapsed, 500*time.Millisecond, "validator must bail once ctx times out")
}

func TestValidate_ReceiptsFetchError(t *testing.T) {
	l2 := &fakeL2{err: errors.New("engine unavailable")}
	v := newValidator(t, l2, map[eth.ChainID]remoteClient{eth.ChainIDFromUInt64(10): &fakeRemote{}})

	err := v.Validate(context.Background(), newEnvelope(1000))
	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch receipts")
}
