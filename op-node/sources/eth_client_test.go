package sources

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRPC struct {
	mock.Mock
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

var testEthClientConfig = &EthClientConfig{
	ReceiptsCacheSize:     10,
	TransactionsCacheSize: 10,
	HeadersCacheSize:      10,
	MaxRequestsPerBatch:   20,
	MaxConcurrentRequests: 10,
	TrustRPC:              false,
}

func randHash() (out common.Hash) {
	rand.Read(out[:])
	return out
}

func randHeader() (*types.Header, *rpcHeader) {
	hdr := &types.Header{
		ParentHash:  randHash(),
		UncleHash:   randHash(),
		Coinbase:    common.Address{},
		Root:        randHash(),
		TxHash:      randHash(),
		ReceiptHash: randHash(),
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(42),
		Number:      big.NewInt(1234),
		GasLimit:    0,
		GasUsed:     0,
		Time:        123456,
		Extra:       make([]byte, 0),
		MixDigest:   randHash(),
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(100),
	}
	rhdr := &rpcHeader{
		cache:  rpcHeaderCacheInfo{Hash: hdr.Hash()},
		header: *hdr,
	}
	return hdr, rhdr
}

func TestEthClient_InfoByHash(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	expectedInfo, _ := rhdr.Info(true)
	ctx := context.Background()
	m.On("CallContext", ctx, new(*rpcHeader),
		"eth_getBlockByHash", []interface{}{rhdr.cache.Hash, false}).Run(func(args mock.Arguments) {
		*args[1].(**rpcHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewEthClient(m, nil, nil, testEthClientConfig)
	require.NoError(t, err)
	info, err := s.InfoByHash(ctx, rhdr.cache.Hash)
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
	// Again, without expecting any calls from the mock, the cache will return the block
	info, err = s.InfoByHash(ctx, rhdr.cache.Hash)
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}

func TestEthClient_InfoByNumber(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	expectedInfo, _ := rhdr.Info(true)
	n := rhdr.header.Number
	ctx := context.Background()
	m.On("CallContext", ctx, new(*rpcHeader),
		"eth_getBlockByNumber", []interface{}{hexutil.EncodeBig(n), false}).Run(func(args mock.Arguments) {
		*args[1].(**rpcHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewL1Client(m, nil, nil, L1ClientDefaultConfig(&rollup.Config{SeqWindowSize: 10}, true))
	require.NoError(t, err)
	info, err := s.InfoByNumber(ctx, n.Uint64())
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}
