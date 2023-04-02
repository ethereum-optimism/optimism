package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ethBackend struct {
	*mock.Mock
}

func (b *ethBackend) GetBlockByHash(id common.Hash, fullTxs bool) (*rpcBlock, error) {
	out := b.Mock.MethodCalled("eth_getBlockByHash", id, fullTxs)
	return out[0].(*rpcBlock), nil
}

func (b *ethBackend) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, error) {
	out := b.Mock.MethodCalled("eth_getTransactionReceipt", txHash)
	return out[0].(*types.Receipt), *out[1].(*error)
}

func (b *ethBackend) GetBlockReceipts(id string) ([]*types.Receipt, error) {
	out := b.Mock.MethodCalled("eth_getBlockReceipts", id)
	return out[0].([]*types.Receipt), *out[1].(*error)
}

type alchemyBackend struct {
	*mock.Mock
}

func (b *alchemyBackend) GetTransactionReceipts(p blockHashParameter) (*receiptsWrapper, error) {
	out := b.Mock.MethodCalled("alchemy_getTransactionReceipts", p.BlockHash.String())
	return &receiptsWrapper{Receipts: out[0].([]*types.Receipt)}, *out[1].(*error)
}

type debugBackend struct {
	*mock.Mock
}

func (b *debugBackend) GetRawReceipts(id string) ([]hexutil.Bytes, error) {
	out := b.Mock.MethodCalled("debug_getRawReceipts", id)
	return out[0].([]hexutil.Bytes), *out[1].(*error)
}

type parityBackend struct {
	*mock.Mock
}

func (b *parityBackend) GetBlockReceipts(id string) ([]*types.Receipt, error) {
	out := b.Mock.MethodCalled("parity_getBlockReceipts", id)
	return out[0].([]*types.Receipt), *out[1].(*error)
}

type ReceiptsRequest struct {
	method ReceiptsFetchingMethod
	result []*types.Receipt
	err    error
}

type methodNotFoundError struct{ method string }

func (e *methodNotFoundError) ErrorCode() int { return -32601 }

func (e *methodNotFoundError) Error() string {
	return fmt.Sprintf("the method %s does not exist/is not available", e.method)
}

// ReceiptsTestCase runs through a series of receipt fetching RPC requests with mocked results
// to test the prioritization/fallback logic of the receipt fetching in the EthClient.
type ReceiptsTestCase struct {
	name         string
	providerKind RPCProviderKind
	staticMethod bool
	setup        func(t *testing.T) (*rpcBlock, []ReceiptsRequest)
}

func (tc *ReceiptsTestCase) Run(t *testing.T) {
	srv := rpc.NewServer()
	defer srv.Stop()
	m := &mock.Mock{}

	require.NoError(t, srv.RegisterName("eth", &ethBackend{Mock: m}))
	require.NoError(t, srv.RegisterName("alchemy", &alchemyBackend{Mock: m}))
	require.NoError(t, srv.RegisterName("debug", &debugBackend{Mock: m}))
	require.NoError(t, srv.RegisterName("parity", &parityBackend{Mock: m}))

	block, requests := tc.setup(t)

	// always expect a block request to fetch txs and receipts root hash etc.
	m.On("eth_getBlockByHash", block.Hash, true).Once().Return(block)

	for _, reqData := range requests {
		req := reqData
		// depending on the method, expect to serve receipts by request(s)
		switch req.method {
		case EthGetTransactionReceiptBatch:
			for i, tx := range block.Transactions {
				m.On("eth_getTransactionReceipt", tx.Hash()).Once().Return(req.result[i], &req.err)
			}
		case AlchemyGetTransactionReceipts:
			m.On("alchemy_getTransactionReceipts", block.Hash.String()).Once().Return(req.result, &req.err)
		case DebugGetRawReceipts:
			var raw []hexutil.Bytes
			for _, r := range req.result {
				data, err := r.MarshalBinary()
				require.NoError(t, err)
				raw = append(raw, data)
			}
			m.On("debug_getRawReceipts", block.Hash.String()).Once().Return(raw, &req.err)
		case ParityGetBlockReceipts:
			m.On("parity_getBlockReceipts", block.Hash.String()).Once().Return(req.result, &req.err)
		case EthGetBlockReceipts:
			m.On("eth_getBlockReceipts", block.Hash.String()).Once().Return(req.result, &req.err)
		default:
			t.Fatalf("unrecognized request method: %d", uint64(req.method))
		}
	}

	cl := rpc.DialInProc(srv)
	testCfg := &EthClientConfig{
		// receipts and transactions are cached per block
		ReceiptsCacheSize:     1000,
		TransactionsCacheSize: 1000,
		HeadersCacheSize:      1000,
		PayloadsCacheSize:     1000,
		MaxRequestsPerBatch:   20,
		MaxConcurrentRequests: 10,
		TrustRPC:              false,
		MustBePostMerge:       false,
		RPCProviderKind:       tc.providerKind,
		MethodResetDuration:   time.Minute,
	}
	if tc.staticMethod { // if static, instantly reset, for fast clock-independent testing
		testCfg.MethodResetDuration = 0
	}
	logger := testlog.Logger(t, log.LvlError)
	ethCl, err := NewEthClient(client.NewBaseRPCClient(cl), logger, nil, testCfg)
	require.NoError(t, err)
	defer ethCl.Close()

	for i, req := range requests {
		info, result, err := ethCl.FetchReceipts(context.Background(), block.Hash)
		if err == nil {
			require.Nil(t, req.err, "error")
			require.Equal(t, block.Hash, info.Hash(), fmt.Sprintf("req %d blockhash", i))
			expectedJson, err := json.MarshalIndent(req.result, "", "  ")
			require.NoError(t, err)
			gotJson, err := json.MarshalIndent(result, "", "  ")
			require.NoError(t, err)
			require.Equal(t, string(expectedJson), string(gotJson), fmt.Sprintf("req %d result", i))
		} else {
			require.NotNil(t, req.err, "error")
			require.Equal(t, req.err.Error(), err.Error(), fmt.Sprintf("req %d err", i))
		}
	}

	m.AssertExpectations(t)
}

func randomRpcBlockAndReceipts(rng *rand.Rand, txCount uint64) (*rpcBlock, []*types.Receipt) {
	block, receipts := testutils.RandomBlock(rng, txCount)
	return &rpcBlock{
		rpcHeader: rpcHeader{
			ParentHash:  block.ParentHash(),
			UncleHash:   block.UncleHash(),
			Coinbase:    block.Coinbase(),
			Root:        block.Root(),
			TxHash:      block.TxHash(),
			ReceiptHash: block.ReceiptHash(),
			Bloom:       eth.Bytes256(block.Bloom()),
			Difficulty:  *(*hexutil.Big)(block.Difficulty()),
			Number:      hexutil.Uint64(block.NumberU64()),
			GasLimit:    hexutil.Uint64(block.GasLimit()),
			GasUsed:     hexutil.Uint64(block.GasUsed()),
			Time:        hexutil.Uint64(block.Time()),
			Extra:       hexutil.Bytes(block.Extra()),
			MixDigest:   block.MixDigest(),
			Nonce:       types.EncodeNonce(block.Nonce()),
			BaseFee:     (*hexutil.Big)(block.BaseFee()),
			Hash:        block.Hash(),
		},
		Transactions: block.Transactions(),
	}, receipts
}

func TestEthClient_FetchReceipts(t *testing.T) {
	// Helper to quickly define the test case requests scenario:
	// each method fails to fetch the receipts, except the last
	fallbackCase := func(txCount uint64, methods ...ReceiptsFetchingMethod) func(t *testing.T) (*rpcBlock, []ReceiptsRequest) {
		return func(t *testing.T) (*rpcBlock, []ReceiptsRequest) {
			block, receipts := randomRpcBlockAndReceipts(rand.New(rand.NewSource(123)), txCount)
			// zero out the data we don't want to verify
			for _, r := range receipts {
				r.ContractAddress = common.Address{}
			}
			var out []ReceiptsRequest
			for _, m := range methods {
				out = append(out, ReceiptsRequest{
					method: m,
				})
			}
			// all but the last request fail to fetch receipts
			for i := 0; i < len(out)-1; i++ {
				out[i].result = nil
				out[i].err = new(methodNotFoundError)
			}
			// last request fetches receipts
			out[len(out)-1].result = receipts
			return block, out
		}
	}

	testCases := []ReceiptsTestCase{
		{
			name:         "alchemy",
			providerKind: RPCKindAlchemy,
			setup:        fallbackCase(30, AlchemyGetTransactionReceipts),
		},
		{
			name:         "alchemy sticky",
			providerKind: RPCKindAlchemy,
			staticMethod: true,
			setup:        fallbackCase(30, AlchemyGetTransactionReceipts, AlchemyGetTransactionReceipts),
		},
		{
			name:         "alchemy fallback 1",
			providerKind: RPCKindAlchemy,
			setup:        fallbackCase(40, AlchemyGetTransactionReceipts, EthGetBlockReceipts),
		},
		{
			name:         "alchemy low tx count cost saving",
			providerKind: RPCKindAlchemy,
			// when it's cheaper to fetch individual receipts than the alchemy-bundled receipts we change methods.
			setup: fallbackCase(5, EthGetTransactionReceiptBatch),
		},
		{
			name:         "quicknode",
			providerKind: RPCKindQuickNode,
			setup:        fallbackCase(30, DebugGetRawReceipts),
		},
		{
			name:         "quicknode fallback 1",
			providerKind: RPCKindQuickNode,
			setup: fallbackCase(30,
				DebugGetRawReceipts,
				EthGetBlockReceipts,
			),
		},
		{
			name:         "quicknode low tx count cost saving",
			providerKind: RPCKindQuickNode,
			// when it's cheaper to fetch individual receipts than the alchemy-bundled receipts we change methods.
			setup: fallbackCase(5, DebugGetRawReceipts, EthGetTransactionReceiptBatch),
		},
		{
			name:         "infura",
			providerKind: RPCKindInfura,
			setup:        fallbackCase(4, EthGetTransactionReceiptBatch),
		},
		{
			name:         "nethermind",
			providerKind: RPCKindNethermind,
			setup:        fallbackCase(4, ParityGetBlockReceipts), // uses parity namespace method
		},
		{
			name:         "geth with debug rpc",
			providerKind: RPCKindDebugGeth,
			setup:        fallbackCase(4, DebugGetRawReceipts),
		},
		{
			name:         "erigon",
			providerKind: RPCKindErigon,
			setup:        fallbackCase(4, EthGetBlockReceipts),
		},
		{
			name:         "basic",
			providerKind: RPCKindBasic,
			setup:        fallbackCase(4, EthGetTransactionReceiptBatch),
		},
		{
			name:         "any discovers alchemy",
			providerKind: RPCKindAny,
			setup:        fallbackCase(4, AlchemyGetTransactionReceipts),
		},
		{
			name:         "any discovers parity",
			providerKind: RPCKindAny,
			// fallback through the least priority method: parity (nethermind supports this still)
			setup: fallbackCase(4,
				AlchemyGetTransactionReceipts,
				DebugGetRawReceipts,
				EthGetBlockReceipts,
				ParityGetBlockReceipts,
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}
