package positions

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type executionStub struct {
	ref        eth.BlockRef
	receipts   types.Receipts
	afterFetch func()
}

func (s *executionStub) BlockRefByNumber(context.Context, uint64) (eth.BlockRef, error) {
	return s.ref, nil
}
func (s *executionStub) FetchReceipts(context.Context, common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	if s.afterFetch != nil {
		s.afterFetch()
	}
	return nil, optypes.FromGethReceipts(s.receipts), nil
}

type safetyStub struct{ number uint64 }

func (s *safetyStub) SyncStatus(context.Context) (*eth.SyncStatus, error) {
	return &eth.SyncStatus{SafeL2: eth.L2BlockRef{Number: s.number}}, nil
}

type fixture struct {
	resolver            *Resolver
	private, projection *executionStub
	safety              *safetyStub
	receipt             *types.Receipt
	block               eth.BlockRef
}

func newFixture() *fixture {
	block := eth.BlockRef{Hash: common.Hash{1}, Number: 5, Time: 10}
	tx := common.Hash{2}
	logs := []*types.Log{
		{Address: common.Address{3}, Index: 0, TxHash: common.Hash{4}},
		{Address: predeploys.L2toL2CrossDomainMessengerAddr, Topics: []common.Hash{render.SentMessageEventTopic}, Data: []byte{1}, Index: 1, TxHash: tx},
		{Address: common.Address{3}, Index: 2, TxHash: tx},
		{Address: predeploys.L2toL2CrossDomainMessengerAddr, Topics: []common.Hash{render.SentMessageEventTopic}, Data: []byte{1}, Index: 3, TxHash: tx},
	}
	rec := &types.Receipt{TxHash: tx, BlockHash: block.Hash, BlockNumber: big.NewInt(5), Logs: logs[1:]}
	private := &executionStub{ref: block, receipts: types.Receipts{{Logs: logs[:1]}, rec}}
	p0, p1 := *logs[1], *logs[3]
	p0.Index, p1.Index = 0, 1
	projection := &executionStub{ref: eth.BlockRef{Hash: common.Hash{5}, Number: 5, Time: 10}, receipts: types.Receipts{{Logs: []*types.Log{&p0, &p1}}}}
	safety := &safetyStub{number: 5}
	return &fixture{New(private, projection, safety, render.NewEmitterSet(), time.Second), private, projection, safety, rec, block}
}

func TestDensePositionsPreserveDuplicateLogs(t *testing.T) {
	f := newFixture()
	out, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.NoError(t, err)
	require.Len(t, out, 3)
	require.True(t, out[0].Public)
	require.Equal(t, uint32(0), out[0].LogIndex)
	require.False(t, out[1].Public)
	require.True(t, out[2].Public)
	require.Equal(t, uint32(1), out[2].LogIndex)
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, out[2].Origin)
}
func TestResolverRejectsInconsistentProjection(t *testing.T) {
	cases := map[string]func(*fixture){
		"extra deposit log": func(f *fixture) {
			f.projection.receipts[0].Logs = append([]*types.Log{{Address: common.Address{9}}}, f.projection.receipts[0].Logs...)
		},
		"wrong index":                          func(f *fixture) { f.projection.receipts[0].Logs[0].Index = 7 },
		"wrong data":                           func(f *fixture) { f.projection.receipts[0].Logs[0].Data = []byte{9} },
		"wrong emitter":                        func(f *fixture) { f.projection.receipts[0].Logs[0].Address = common.Address{9} },
		"wrong timestamp":                      func(f *fixture) { f.projection.ref.Time++ },
		"private orphan":                       func(f *fixture) { f.private.ref.Hash = common.Hash{9} },
		"projection reorg":                     func(f *fixture) { f.projection.afterFetch = func() { f.projection.ref.Hash = common.Hash{9} } },
		"private reorg during projection read": func(f *fixture) { f.projection.afterFetch = func() { f.private.ref.Hash = common.Hash{9} } },
		"receipt names another transaction": func(f *fixture) {
			copyLog := *f.receipt.Logs[0]
			copyLog.TxHash = common.Hash{9}
			f.receipt.Logs = append([]*types.Log{&copyLog}, f.receipt.Logs[1:]...)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			change(f)
			_, err := f.resolver.ResolvePublishedPositions(t.Context(), f.receipt, f.block)
			require.Error(t, err)
		})
	}
}
func TestResolverExtraEmitterUsesReplayer(t *testing.T) {
	f := newFixture()
	addr := common.Address{8}
	f.receipt.Logs[0].Address = addr
	f.projection.receipts[0].Logs[0].Address = predeploys.EventReplayerAddr
	f.resolver.emitters = render.NewEmitterSet(addr)
	out, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.NoError(t, err)
	require.Equal(t, predeploys.EventReplayerAddr, out[0].Origin)
}
func TestResolverPublishedPositionWaitIsBounded(t *testing.T) {
	f := newFixture()
	f.safety.number = 0
	f.resolver.timeout = time.Millisecond
	_, err := f.resolver.ResolvePublishedPositions(t.Context(), f.receipt, f.block)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestResolverPositionsBeforePublication(t *testing.T) {
	f := newFixture()
	want, err := f.resolver.ResolvePublishedPositions(t.Context(), f.receipt, f.block)
	require.NoError(t, err)
	// Neither the projection nor its safety RPC is needed for immediate resolution.
	f.resolver.projection = nil
	f.resolver.safety = nil
	got, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestResolverUnpublishedPositionsRejectPrivateReorg(t *testing.T) {
	f := newFixture()
	f.resolver.projection = nil
	f.resolver.safety = nil
	f.private.afterFetch = func() { f.private.ref.Hash = common.Hash{9} }
	_, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.ErrorContains(t, err, "private chain changed")
}

func TestResolverUnpublishedPositionsRejectMismatchedReceipt(t *testing.T) {
	f := newFixture()
	f.resolver.projection = nil
	f.resolver.safety = nil
	copyLog := *f.receipt.Logs[0]
	copyLog.TxHash = common.Hash{9}
	f.receipt.Logs = append([]*types.Log{&copyLog}, f.receipt.Logs[1:]...)
	_, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.ErrorContains(t, err, "does not match its private block")
}
func TestResolverPrivateOnlyReceiptDoesNotWait(t *testing.T) {
	f := newFixture()
	f.safety.number = 0
	rec := *f.receipt
	rec.Logs = rec.Logs[1:2]
	f.receipt = &rec
	out, err := f.resolver.ResolvePositions(t.Context(), f.receipt, f.block)
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.False(t, out[0].Public)
}
