package sources

import (
	"context"
	crand "crypto/rand"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockRPC struct {
	mock.Mock
}

func (m *mockRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return m.MethodCalled("BatchCallContext", ctx, b).Get(0).([]error)[0]
}

func (m *mockRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return m.MethodCalled("CallContext", ctx, result, method, args).Get(0).([]error)[0]
}

func (m *mockRPC) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	called := m.MethodCalled("Subscribe", namespace, channel, args)
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
	PayloadsCacheSize:     10,
	MaxRequestsPerBatch:   20,
	MaxConcurrentRequests: 10,
	TrustRPC:              false,
	MustBePostMerge:       false,
	RPCProviderKind:       RPCKindStandard,
}

func randHash() (out common.Hash) {
	_, _ = crand.Read(out[:])
	return out
}

func randHeader() (*types.Header, *RPCHeader) {
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
	rhdr := &RPCHeader{
		ParentHash:  hdr.ParentHash,
		UncleHash:   hdr.UncleHash,
		Coinbase:    hdr.Coinbase,
		Root:        hdr.Root,
		TxHash:      hdr.TxHash,
		ReceiptHash: hdr.ReceiptHash,
		Bloom:       eth.Bytes256(hdr.Bloom),
		Difficulty:  *(*hexutil.Big)(hdr.Difficulty),
		Number:      hexutil.Uint64(bigs.Uint64Strict(hdr.Number)),
		GasLimit:    hexutil.Uint64(hdr.GasLimit),
		GasUsed:     hexutil.Uint64(hdr.GasUsed),
		Time:        hexutil.Uint64(hdr.Time),
		Extra:       hdr.Extra,
		MixDigest:   hdr.MixDigest,
		Nonce:       hdr.Nonce,
		BaseFee:     (*hexutil.Big)(hdr.BaseFee),
		Hash:        hdr.Hash(),
	}
	return hdr, rhdr
}

func TestEthClient_InfoByHash(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	expectedInfo, _ := rhdr.Info(true, false)
	ctx := context.Background()
	m.On("CallContext", ctx, new(*RPCHeader),
		"eth_getBlockByHash", []any{rhdr.Hash, false}).Run(func(args mock.Arguments) {
		*args[1].(**RPCHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewEthClient(m, nil, nil, testEthClientConfig)
	require.NoError(t, err)
	info, err := s.InfoByHash(ctx, rhdr.Hash)
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
	// Again, without expecting any calls from the mock, the cache will return the block
	info, err = s.InfoByHash(ctx, rhdr.Hash)
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}

func TestEthClient_InfoByNumber(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	expectedInfo, _ := rhdr.Info(true, false)
	n := rhdr.Number
	ctx := context.Background()
	m.On("CallContext", ctx, new(*RPCHeader),
		"eth_getBlockByNumber", []any{n.String(), false}).Run(func(args mock.Arguments) {
		*args[1].(**RPCHeader) = rhdr
	}).Return([]error{nil})
	s, err := NewL1Client(m, nil, nil, L1ClientDefaultConfig(&rollup.Config{SeqWindowSize: 10}, true, RPCKindStandard))
	require.NoError(t, err)
	info, err := s.InfoByNumber(ctx, uint64(n))
	require.NoError(t, err)
	require.Equal(t, info, expectedInfo)
	m.Mock.AssertExpectations(t)
}

func TestEthClient_WrongInfoByNumber(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	rhdr2 := *rhdr
	rhdr2.Number += 1
	n := rhdr.Number
	ctx := context.Background()
	m.On("CallContext", ctx, new(*RPCHeader),
		"eth_getBlockByNumber", []any{n.String(), false}).Run(func(args mock.Arguments) {
		*args[1].(**RPCHeader) = &rhdr2
	}).Return([]error{nil})
	s, err := NewL1Client(m, nil, nil, L1ClientDefaultConfig(&rollup.Config{SeqWindowSize: 10}, true, RPCKindStandard))
	require.NoError(t, err)
	_, err = s.InfoByNumber(ctx, uint64(n))
	require.Error(t, err, "cannot accept the wrong block")
	m.Mock.AssertExpectations(t)
}

func TestEthClient_WrongInfoByHash(t *testing.T) {
	m := new(mockRPC)
	_, rhdr := randHeader()
	rhdr2 := *rhdr
	rhdr2.Root[0] += 1
	rhdr2.Hash = rhdr2.computeBlockHash()
	k := rhdr.Hash
	ctx := context.Background()
	m.On("CallContext", ctx, new(*RPCHeader),
		"eth_getBlockByHash", []any{k, false}).Run(func(args mock.Arguments) {
		*args[1].(**RPCHeader) = &rhdr2
	}).Return([]error{nil})
	s, err := NewL1Client(m, nil, nil, L1ClientDefaultConfig(&rollup.Config{SeqWindowSize: 10}, true, RPCKindStandard))
	require.NoError(t, err)
	_, err = s.InfoByHash(ctx, k)
	require.Error(t, err, "cannot accept the wrong block")
	m.Mock.AssertExpectations(t)
}

func newEthClientWithCaches(metrics caching.Metrics, cacheSize int) *EthClient {
	return &EthClient{
		transactionsCache: caching.NewLRUCache[common.Hash, types.Transactions](metrics, "txs", cacheSize),
		headersCache:      caching.NewLRUCache[common.Hash, *types.Header](metrics, "headers", cacheSize),
		payloadsCache:     caching.NewLRUCache[common.Hash, *eth.ExecutionPayloadEnvelope](metrics, "payloads", cacheSize),
	}
}

// makeDepositTx returns a minimal deposit-type transaction for tests.
func makeDepositTx(seq uint64) *types.Transaction {
	return types.NewTx(&types.DepositTx{
		SourceHash: common.Hash{0xde, byte(seq)},
		From:       common.HexToAddress("0xdead"),
		To:         &common.Address{0x42},
		Mint:       big.NewInt(0),
		Value:      big.NewInt(0),
		Gas:        21000,
		Data:       []byte{0xca, 0xfe},
	})
}

// makeNonDepositTx returns a non-deposit tx for testing rejection paths.
func makeNonDepositTx() *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{0x42},
		Value:    big.NewInt(0),
		Gas:      21000,
		GasPrice: big.NewInt(1),
	})
}

// expectBatchHeaderAndTx wires the mockRPC to fulfill a HeaderAndFirstTx
// batch (eth_getBlockBy* + eth_getTransactionByBlock*AndIndex). The setup fn
// receives pointers it must populate.
func expectBatchHeaderAndTx(m *mockRPC, populate func(hdr *RPCHeaderWithTxHashes, firstTx **types.Transaction)) {
	m.On("BatchCallContext", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		b := args.Get(1).([]rpc.BatchElem)
		hdrPtr := b[0].Result.(*RPCHeaderWithTxHashes)
		txPtrPtr := b[1].Result.(**types.Transaction)
		populate(hdrPtr, txPtrPtr)
	}).Return([]error{nil})
}

// rpcHeaderWithTxs builds an RPCHeaderWithTxHashes from a real header and the
// list of tx hashes the EL would have included.
func rpcHeaderWithTxs(hdr *types.Header, txHashes []common.Hash) *RPCHeaderWithTxHashes {
	rh := &RPCHeader{
		ParentHash:  hdr.ParentHash,
		UncleHash:   hdr.UncleHash,
		Coinbase:    hdr.Coinbase,
		Root:        hdr.Root,
		TxHash:      hdr.TxHash,
		ReceiptHash: hdr.ReceiptHash,
		Bloom:       eth.Bytes256(hdr.Bloom),
		Difficulty:  *(*hexutil.Big)(hdr.Difficulty),
		Number:      hexutil.Uint64(bigs.Uint64Strict(hdr.Number)),
		GasLimit:    hexutil.Uint64(hdr.GasLimit),
		GasUsed:     hexutil.Uint64(hdr.GasUsed),
		Time:        hexutil.Uint64(hdr.Time),
		Extra:       hdr.Extra,
		MixDigest:   hdr.MixDigest,
		Nonce:       hdr.Nonce,
		BaseFee:     (*hexutil.Big)(hdr.BaseFee),
		Hash:        hdr.Hash(),
	}
	return &RPCHeaderWithTxHashes{RPCHeader: *rh, Transactions: txHashes}
}

func TestEthClient_HeaderAndFirstTx_ByHash_Happy(t *testing.T) {
	hdr, _ := randHeader()
	deposit := makeDepositTx(1)
	rh := rpcHeaderWithTxs(hdr, []common.Hash{deposit.Hash()})

	m := new(mockRPC)
	expectBatchHeaderAndTx(m, func(hdrOut *RPCHeaderWithTxHashes, txOut **types.Transaction) {
		*hdrOut = *rh
		*txOut = deposit
	})
	s, err := NewEthClient(m, testlog.Logger(t, log.LevelError), nil, testEthClientConfig)
	require.NoError(t, err)

	ctx := context.Background()
	gotHdr, gotTx, err := s.HeaderAndFirstTx(ctx, hashID(rh.Hash))
	require.NoError(t, err)
	require.Equal(t, rh.Hash, gotHdr.Hash())
	require.Equal(t, deposit.Hash(), gotTx.Hash())
	m.Mock.AssertExpectations(t)
}

func TestEthClient_HeaderAndFirstTx_ByHash_CacheHit(t *testing.T) {
	hdr, _ := randHeader()
	deposit := makeDepositTx(2)
	rh := rpcHeaderWithTxs(hdr, []common.Hash{deposit.Hash()})

	m := new(mockRPC)
	expectBatchHeaderAndTx(m, func(hdrOut *RPCHeaderWithTxHashes, txOut **types.Transaction) {
		*hdrOut = *rh
		*txOut = deposit
	})
	s, err := NewEthClient(m, testlog.Logger(t, log.LevelError), nil, testEthClientConfig)
	require.NoError(t, err)

	ctx := context.Background()
	_, _, err = s.HeaderAndFirstTx(ctx, hashID(rh.Hash))
	require.NoError(t, err)
	// Second call hits cache; no further RPC expected.
	gotHdr, gotTx, err := s.HeaderAndFirstTx(ctx, hashID(rh.Hash))
	require.NoError(t, err)
	require.Equal(t, rh.Hash, gotHdr.Hash())
	require.Equal(t, deposit.Hash(), gotTx.Hash())
	m.Mock.AssertExpectations(t)
}

func TestEthClient_HeaderAndFirstTx_ByNumber_RaceRetry(t *testing.T) {
	hdr, _ := randHeader()
	headerSideTx := makeDepositTx(3)    // hash that the header references
	conflictingTx := makeNonDepositTx() // wrong hash + wrong type — but we'll treat it as the "stale" tx the batch returned
	require.NotEqual(t, headerSideTx.Hash(), conflictingTx.Hash())
	rh := rpcHeaderWithTxs(hdr, []common.Hash{headerSideTx.Hash()})

	m := new(mockRPC)
	// 1) batched call returns header with one tx hash and a tx whose hash does NOT match
	expectBatchHeaderAndTx(m, func(hdrOut *RPCHeaderWithTxHashes, txOut **types.Transaction) {
		*hdrOut = *rh
		*txOut = conflictingTx
	})
	// 2) follow-up tx-by-hash-and-index returns the correct tx
	m.On("CallContext", mock.Anything, mock.Anything,
		"eth_getTransactionByBlockHashAndIndex",
		[]any{rh.Hash, hexutil.EncodeUint64(0)}).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(**types.Transaction)
			*out = headerSideTx
		}).Return([]error{nil})

	s, err := NewEthClient(m, testlog.Logger(t, log.LevelError), nil, testEthClientConfig)
	require.NoError(t, err)

	ctx := context.Background()
	gotHdr, gotTx, err := s.HeaderAndFirstTx(ctx, numberID(uint64(rh.Number)))
	require.NoError(t, err)
	require.Equal(t, rh.Hash, gotHdr.Hash())
	require.Equal(t, headerSideTx.Hash(), gotTx.Hash())
	m.Mock.AssertExpectations(t)
}

func TestEthClient_HeaderAndFirstTx_FirstTxNotDeposit(t *testing.T) {
	hdr, _ := randHeader()
	tx := makeNonDepositTx()
	rh := rpcHeaderWithTxs(hdr, []common.Hash{tx.Hash()})

	m := new(mockRPC)
	expectBatchHeaderAndTx(m, func(hdrOut *RPCHeaderWithTxHashes, txOut **types.Transaction) {
		*hdrOut = *rh
		*txOut = tx
	})
	s, err := NewEthClient(m, testlog.Logger(t, log.LevelError), nil, testEthClientConfig)
	require.NoError(t, err)

	ctx := context.Background()
	_, _, err = s.HeaderAndFirstTx(ctx, hashID(rh.Hash))
	require.Error(t, err, "expected first-tx-not-deposit rejection")
	require.Contains(t, err.Error(), "unexpected tx type")
}

func TestEthClient_HeaderAndFirstTx_EmptyBlock(t *testing.T) {
	hdr, _ := randHeader()
	rh := rpcHeaderWithTxs(hdr, nil)

	m := new(mockRPC)
	expectBatchHeaderAndTx(m, func(hdrOut *RPCHeaderWithTxHashes, txOut **types.Transaction) {
		*hdrOut = *rh
		*txOut = nil
	})
	s, err := NewEthClient(m, testlog.Logger(t, log.LevelError), nil, testEthClientConfig)
	require.NoError(t, err)

	ctx := context.Background()
	_, _, err = s.HeaderAndFirstTx(ctx, hashID(rh.Hash))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no transactions")
}

// TestReceiptValidation tests that the receipt validation is performed by the underlying RPCReceiptsFetcher
func TestReceiptValidation(t *testing.T) {
	require := require.New(t)
	mrpc := new(mockRPC)
	rp := NewRPCReceiptsFetcher(mrpc, nil, RPCReceiptsConfig{})
	const numTxs = 1
	block, _ := randomRpcBlockAndReceipts(rand.New(rand.NewSource(420)), numTxs)
	//txHashes := receiptTxHashes(receipts)
	ctx := context.Background()

	mrpc.On("CallContext",
		ctx,
		mock.Anything,
		"eth_getTransactionReceipt",
		mock.Anything).
		Run(func(args mock.Arguments) {
		}).
		Return([]error{nil})

	// when the block is requested, the block is returned
	mrpc.On("CallContext",
		ctx,
		mock.Anything,
		"eth_getBlockByHash",
		mock.Anything).
		Run(func(args mock.Arguments) {
			*(args[1].(**RPCBlock)) = block
		}).
		Return([]error{nil})

	ethcl := newEthClientWithCaches(nil, numTxs)
	ethcl.client = mrpc
	ethcl.recProvider = rp
	ethcl.trustRPC = true

	_, _, err := ethcl.FetchReceipts(ctx, block.Hash)
	require.ErrorContains(err, "unexpected nil block number")
}
