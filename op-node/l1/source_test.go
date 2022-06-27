package l1

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRPC struct {
	mock.Mock
}

// we catch the optimized version, instead of mocking a lot of split/parallel calls
func (m *mockRPC) batchCall(ctx context.Context, b []rpc.BatchElem) error {
	return m.MethodCalled("batchCall", ctx, b).Get(0).([]error)[0]
}

func (m *mockRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return m.MethodCalled("BatchCallContext", ctx, b).Get(0).([]error)[0]
}

func (m *mockRPC) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return m.MethodCalled("CallContext", ctx, result, method, args).Get(0).([]error)[0]
}

func (m *mockRPC) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	called := m.MethodCalled("EthSubscribe", channel, args)
	return called.Get(0).(*rpc.ClientSubscription), called.Get(1).([]error)[0]
}

func (m *mockRPC) Close() {
	m.MethodCalled("Close")
}

var _ client.RPC = (*mockRPC)(nil)

func randHash() (out common.Hash) {
	rand.Read(out[:])
	return out
}

func randHeader() *types.Header {
	return &types.Header{
		ParentHash:  randHash(),
		UncleHash:   randHash(),
		Coinbase:    common.Address{},
		Root:        randHash(),
		TxHash:      randHash(),
		ReceiptHash: randHash(),
		Number:      big.NewInt(1234),
		Time:        123456,
		MixDigest:   randHash(),
		BaseFee:     big.NewInt(100),
	}
}

func randTransaction(i uint64) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(999),
		Nonce:     i,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(100),
		Gas:       21000,
		To:        &common.Address{0x42},
		Value:     big.NewInt(0),
	})
}

func randTxs(offset uint64, count uint64) types.Transactions {
	out := make(types.Transactions, count)
	for i := uint64(0); i < count; i++ {
		out[i] = randTransaction(offset + i)
	}
	return out
}

func TestSource_InfoByHash(t *testing.T) {
	log := testlog.Logger(t, log.LvlError)
	m := new(mockRPC)
	hdr := randHeader()
	rhdr := &rpcHeader{
		cache:  rpcHeaderCacheInfo{Hash: hdr.Hash()},
		header: *hdr,
	}
	expectedInfo, _ := rhdr.Info(true)
	h := rhdr.header.Hash()
	ctx := context.Background()
	m.On("CallContext", ctx, new(*rpcHeader), "eth_getBlockByHash", []interface{}{h, false}).Run(func(args mock.Arguments) {
		*args[1].(**rpcHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewSource(m, log, DefaultConfig(&rollup.Config{SeqWindowSize: 10}, true))
	assert.NoError(t, err)
	info, err := s.InfoByHash(ctx, h)
	assert.NoError(t, err)
	assert.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
	// Again, without expecting any calls from the mock, the cache will return the block
	info, err = s.InfoByHash(ctx, h)
	assert.NoError(t, err)
	assert.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}

func TestSource_InfoByNumber(t *testing.T) {
	log := testlog.Logger(t, log.LvlError)
	m := new(mockRPC)
	hdr := randHeader()
	rhdr := &rpcHeader{
		cache:  rpcHeaderCacheInfo{Hash: hdr.Hash()},
		header: *hdr,
	}
	expectedInfo, _ := rhdr.Info(true)
	n := hdr.Number.Uint64()
	ctx := context.Background()
	m.On("CallContext", ctx, new(*rpcHeader), "eth_getBlockByNumber", []interface{}{hexutil.EncodeUint64(n), false}).Run(func(args mock.Arguments) {
		*args[1].(**rpcHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewSource(m, log, DefaultConfig(&rollup.Config{SeqWindowSize: 10}, true))
	assert.NoError(t, err)
	info, err := s.InfoByNumber(ctx, n)
	assert.NoError(t, err)
	assert.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}

func TestSource_FetchAllTransactions(t *testing.T) {
	log := testlog.Logger(t, log.LvlError)
	m := new(mockRPC)

	ctx := context.Background()
	a, b := randHeader(), randHeader()
	blocks := []*rpcBlock{
		{
			header: rpcHeader{
				cache: rpcHeaderCacheInfo{
					Hash: a.Hash(),
				},
				header: *a,
			},
			extra: rpcBlockCacheInfo{
				Transactions: randTxs(0, 4),
			},
		},
		{
			header: rpcHeader{
				cache: rpcHeaderCacheInfo{
					Hash: b.Hash(),
				},
				header: *b,
			},
			extra: rpcBlockCacheInfo{
				Transactions: randTxs(4, 3),
			},
		},
	}
	expectedRequest := make([]rpc.BatchElem, len(blocks))
	expectedTxLists := make([]types.Transactions, len(blocks))
	for i, b := range blocks {
		expectedRequest[i] = rpc.BatchElem{Method: "eth_getBlockByHash", Args: []interface{}{b.header.header.Hash(), true}, Result: new(rpcBlock)}
		expectedTxLists[i] = b.extra.Transactions
	}

	m.On("batchCall", ctx, expectedRequest).Run(func(args mock.Arguments) {
		batch := args[1].([]rpc.BatchElem)
		for i, b := range blocks {
			*batch[i].Result.(*rpcBlock) = *b
		}
	}).Return([]error{nil})

	s, err := NewSource(m, log, DefaultConfig(&rollup.Config{SeqWindowSize: 10}, true))
	assert.NoError(t, err)
	s.batchCall = m.batchCall // override the optimized batch call

	id := func(i int) eth.BlockID {
		return eth.BlockID{Hash: blocks[i].header.header.Hash(), Number: blocks[i].header.header.Number.Uint64()}
	}

	txLists, err := s.FetchAllTransactions(ctx, []eth.BlockID{id(0), id(1)})
	assert.NoError(t, err)
	assert.Equal(t, txLists, expectedTxLists)
	m.Mock.AssertExpectations(t)

	// again, but now without expecting any calls (transactions were cached)
	txLists, err = s.FetchAllTransactions(ctx, []eth.BlockID{id(0), id(1)})
	assert.NoError(t, err)
	assert.Equal(t, txLists, expectedTxLists)
	m.Mock.AssertExpectations(t)
}
