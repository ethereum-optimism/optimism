package interopsafety

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const (
	chainA = uint64(901)
	chainB = uint64(902)
)

type fakeBlock struct {
	hash  common.Hash
	time  uint64
	logs  []*ethTypes.Log
	types []uint64
}

type fakeChain struct {
	ss       eth.SyncStatus
	blocks   map[uint64]fakeBlock
	receipts map[common.Hash]bool
}

func (f *fakeChain) SyncStatus(context.Context) (*eth.SyncStatus, error) { return &f.ss, nil }
func (f *fakeChain) Header(_ context.Context, n uint64) (common.Hash, uint64, error) {
	b, ok := f.blocks[n]
	if !ok {
		return common.Hash{}, 0, errors.New("not found")
	}
	return b.hash, b.time, nil
}
func (f *fakeChain) BlockLogs(_ context.Context, n uint64) ([]*ethTypes.Log, error) {
	return f.blocks[n].logs, nil
}
func (f *fakeChain) ExecutingMessageLogs(_ context.Context, from, to uint64) ([]*ethTypes.Log, error) {
	var out []*ethTypes.Log
	for n := from; n <= to; n++ {
		for _, l := range f.blocks[n].logs {
			if msg, _ := messages.MessageFromLog(l); msg != nil {
				out = append(out, l)
			}
		}
	}
	return out, nil
}
func (f *fakeChain) HasReceipt(_ context.Context, tx common.Hash) (bool, error) {
	return f.receipts[tx], nil
}
func (f *fakeChain) TxTypes(_ context.Context, n uint64) ([]uint64, error) {
	return f.blocks[n].types, nil
}

var emitter = common.HexToAddress("0x1234")

// initLog is the initiating log: block 10 on chain A, log index 1.
func initLog() *ethTypes.Log {
	return &ethTypes.Log{
		Address: emitter,
		Topics:  []common.Hash{crypto.Keccak256Hash([]byte("topic"))},
		Data:    []byte("payload"),
		Index:   1,
	}
}

func validMessage() *messages.Message {
	return &messages.Message{
		Identifier: messages.Identifier{
			Origin:      emitter,
			BlockNumber: 10,
			LogIndex:    1,
			Timestamp:   1000,
			ChainID:     eth.ChainIDFromUInt64(chainA),
		},
		PayloadHash: crypto.Keccak256Hash(messages.LogToMessagePayload(initLog())),
	}
}

func readers() map[uint64]ChainReader {
	a := &fakeChain{blocks: map[uint64]fakeBlock{
		10: {hash: common.HexToHash("0xa10"), time: 1000, logs: []*ethTypes.Log{{Address: emitter, Index: 0}, initLog()}},
	}}
	b := &fakeChain{blocks: map[uint64]fakeBlock{}}
	return map[uint64]ChainReader{chainA: a, chainB: b}
}

func TestValidateMessage(t *testing.T) {
	crossSafe := map[uint64]uint64{chainA: 20, chainB: 20}
	ctx := context.Background()
	cases := []struct {
		name   string
		mutate func(m *messages.Message)
		time   uint64
		errMsg string
	}{
		{name: "valid", mutate: func(*messages.Message) {}, time: 1002},
		{name: "wrong log index", mutate: func(m *messages.Message) { m.Identifier.LogIndex = 2 }, time: 1002, errMsg: "no log 2"},
		{name: "payload hash mismatch", mutate: func(m *messages.Message) { m.PayloadHash = common.HexToHash("0xbad") }, time: 1002, errMsg: "payload hash"},
		{name: "initiated after execution", mutate: func(*messages.Message) {}, time: 999, errMsg: "after executing block timestamp"},
		{name: "unknown source chain", mutate: func(m *messages.Message) { m.Identifier.ChainID = eth.ChainIDFromUInt64(777) }, time: 1002, errMsg: "not in the dependency set"},
		{name: "wrong origin", mutate: func(m *messages.Message) { m.Identifier.Origin = common.HexToAddress("0x9") }, time: 1002, errMsg: "identifier says"},
		{name: "identifier timestamp mismatch", mutate: func(m *messages.Message) { m.Identifier.Timestamp = 1001 }, time: 1002, errMsg: "does not match block"},
		{name: "beyond cross-safe", mutate: func(m *messages.Message) { m.Identifier.BlockNumber = 30 }, time: 1002, errMsg: "beyond the cross-safe head"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := validMessage()
			tc.mutate(msg)
			err := ValidateMessage(ctx, readers(), crossSafe, tc.time, msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.errMsg)
		})
	}
}

// execLog is a CrossL2Inbox ExecutingMessage log for msg.
func execLog(t *testing.T, msg *messages.Message, block uint64) *ethTypes.Log {
	l := &ethTypes.Log{BlockNumber: block}
	l.Address, l.Topics, l.Data = encodeExecutingMessage(t, msg)
	return l
}

func TestCheckerFindsInvalidMessageInCrossSafe(t *testing.T) {
	r := readers()
	b := r[chainB].(*fakeChain)
	good := validMessage()
	bad := validMessage()
	bad.Identifier.LogIndex = 5
	b.blocks[12] = fakeBlock{hash: common.HexToHash("0xb12"), time: 1004, logs: []*ethTypes.Log{execLog(t, good, 12)}}
	b.blocks[13] = fakeBlock{hash: common.HexToHash("0xb13"), time: 1006, logs: []*ethTypes.Log{execLog(t, bad, 13)}}
	a := r[chainA].(*fakeChain)
	a.ss.SafeL2 = eth.L2BlockRef{Number: 10, Hash: common.HexToHash("0xa10")}
	b.ss.SafeL2 = eth.L2BlockRef{Number: 13, Hash: common.HexToHash("0xb13")}

	now := time.Unix(2000, 0)
	k := NewChecker(testlog.Logger(t, slog.LevelError), Config{From: map[uint64]uint64{chainA: 10, chainB: 12}, StallAfter: time.Minute},
		[]uint64{chainA, chainB}, []*Supernode{{Name: "sn0", Chains: r}}, func() time.Time { return now })
	k.Pass(context.Background())

	require.Equal(t, 2, k.State()["sn0"][chainB].ExecMsgs)
	require.Len(t, k.Failures(), 1)
	require.Equal(t, uint64(13), k.Failures()[0].Block)
	require.Contains(t, k.Failures()[0].Reason, "no log 5")
}

func TestCheckerReplacements(t *testing.T) {
	r := readers()
	gone := common.HexToHash("0xdead")
	a := r[chainA].(*fakeChain)
	a.receipts = map[common.Hash]bool{gone: true}
	a.blocks[11] = fakeBlock{hash: common.HexToHash("0xa11"), types: []uint64{depositTxType, 2}}
	k := NewChecker(testlog.Logger(t, slog.LevelError), Config{GoneTxs: []common.Hash{gone}, DepositOnly: map[uint64][]uint64{chainA: {11}}},
		[]uint64{chainA, chainB}, []*Supernode{{Name: "sn0", Chains: r}}, time.Now)
	k.CheckReplacements(context.Background())
	require.Len(t, k.Failures(), 2)
}

// encodeExecutingMessage builds the CrossL2Inbox ExecutingMessage log fields for msg,
// the inverse of messages.Message.DecodeEvent.
func encodeExecutingMessage(t *testing.T, msg *messages.Message) (common.Address, []common.Hash, []byte) {
	t.Helper()
	id := msg.Identifier
	data := make([]byte, 32*5)
	copy(data[12:32], id.Origin.Bytes())
	binary.BigEndian.PutUint64(data[32+24:64], id.BlockNumber)
	binary.BigEndian.PutUint32(data[64+28:96], id.LogIndex)
	binary.BigEndian.PutUint64(data[96+24:128], id.Timestamp)
	chainID := id.ChainID.Bytes32()
	copy(data[128:160], chainID[:])
	return predeploys.CrossL2InboxAddr, []common.Hash{messages.ExecutingMessageEventTopic, msg.PayloadHash}, data
}
