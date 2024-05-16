package node

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
)

// make a set of headers which chain correctly
func makeHeaders(numHeaders uint64, prevHeader *types.Header) []types.Header {
	headers := make([]types.Header, numHeaders)
	for i := range headers {
		if i == 0 {
			if prevHeader == nil {
				// genesis
				headers[i] = types.Header{Number: big.NewInt(0)}
			} else {
				// chain onto the previous header
				headers[i] = types.Header{Number: big.NewInt(prevHeader.Number.Int64() + 1)}
				headers[i].ParentHash = prevHeader.Hash()
			}
		} else {
			headers[i] = types.Header{Number: big.NewInt(headers[i-1].Number.Int64() + 1)}
			headers[i].ParentHash = headers[i-1].Hash()
		}
	}

	return headers
}

func TestHeaderTraversalNextHeadersNoOp(t *testing.T) {
	client := &testutils.MockClient{}
	t.Cleanup(func() { client.AssertExpectations(t) })

	// start from block 10 as the latest fetched block
	LastTraversedHeader := &types.Header{Number: big.NewInt(10)}
	headerTraversal := NewHeaderTraversal(client, LastTraversedHeader, bigint.Zero)

	require.Nil(t, headerTraversal.LatestHeader())
	require.NotNil(t, headerTraversal.LastTraversedHeader())

	// no new headers when matched with head
	client.ExpectHeaderByNumber(nil, LastTraversedHeader, nil)
	headers, err := headerTraversal.NextHeaders(100)
	require.NoError(t, err)
	require.Empty(t, headers)

	require.NotNil(t, headerTraversal.LatestHeader())
	require.NotNil(t, headerTraversal.LastTraversedHeader())
	require.Equal(t, LastTraversedHeader.Number.Uint64(), headerTraversal.LatestHeader().Number.Uint64())
}

func TestHeaderTraversalNextHeadersCursored(t *testing.T) {
	client := &testutils.MockClient{}
	t.Cleanup(func() { client.AssertExpectations(t) })

	rpc := &testutils.MockRPC{}
	client.Mock.On("RPC").Return(rpc)
	t.Cleanup(func() { rpc.AssertExpectations(t) })

	// start from genesis, 7 available headers
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)
	client.ExpectHeaderByNumber(nil, &types.Header{Number: big.NewInt(7)}, nil)

	headers := makeHeaders(10, nil)
	rpcElems := makeHeaderRpcElems(headers[0].Number, headers[9].Number)
	for i := 0; i < len(rpcElems); i++ {
		rpcElems[i].Result = &headers[i]
	}

	// traverse blocks [0..4]. Latest reported is 7
	rpc.ExpectBatchCallContext(rpcElems[:5], nil)
	_, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)

	require.Equal(t, uint64(7), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(4), headerTraversal.LastTraversedHeader().Number.Uint64())

	// blocks [5..9]. Latest Reported is 9
	client.ExpectHeaderByNumber(nil, &headers[9], nil)
	rpc.ExpectBatchCallContext(rpcElems[5:], nil)
	_, err = headerTraversal.NextHeaders(5)
	require.NoError(t, err)

	require.Equal(t, uint64(9), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(9), headerTraversal.LastTraversedHeader().Number.Uint64())
}

func TestHeaderTraversalNextHeadersMaxSize(t *testing.T) {
	client := &testutils.MockClient{}
	t.Cleanup(func() { client.AssertExpectations(t) })

	rpc := &testutils.MockRPC{}
	client.Mock.On("RPC").Return(rpc)
	t.Cleanup(func() { rpc.AssertExpectations(t) })

	// start from genesis, 100 available headers
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)
	client.ExpectHeaderByNumber(nil, &types.Header{Number: big.NewInt(100)}, nil)

	headers := makeHeaders(5, nil)
	rpcElems := makeHeaderRpcElems(headers[0].Number, headers[4].Number)
	for i := 0; i < len(rpcElems); i++ {
		rpcElems[i].Result = &headers[i]
	}

	// traverse only 5 headers [0..4]
	rpc.ExpectBatchCallContext(rpcElems, nil)
	headers, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	require.Equal(t, uint64(100), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(4), headerTraversal.LastTraversedHeader().Number.Uint64())

	// clamped by the supplied size. FinalizedHeight == 100
	client.ExpectHeaderByNumber(nil, &types.Header{Number: big.NewInt(100)}, nil)
	headers = makeHeaders(10, &headers[len(headers)-1])
	rpcElems = makeHeaderRpcElems(headers[0].Number, headers[9].Number)
	for i := 0; i < len(rpcElems); i++ {
		rpcElems[i].Result = &headers[i]
	}

	rpc.ExpectBatchCallContext(rpcElems, nil)
	headers, err = headerTraversal.NextHeaders(10)
	require.NoError(t, err)
	require.Len(t, headers, 10)

	require.Equal(t, uint64(100), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(14), headerTraversal.LastTraversedHeader().Number.Uint64())
}

func TestHeaderTraversalMismatchedProviderStateError(t *testing.T) {
	client := &testutils.MockClient{}
	t.Cleanup(func() { client.AssertExpectations(t) })

	rpc := &testutils.MockRPC{}
	client.Mock.On("RPC").Return(rpc)
	t.Cleanup(func() { rpc.AssertExpectations(t) })

	// start from genesis
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)

	// blocks [0..4]
	headers := makeHeaders(5, nil)
	rpcElems := makeHeaderRpcElems(headers[0].Number, headers[4].Number)
	for i := 0; i < len(rpcElems); i++ {
		rpcElems[i].Result = &headers[i]
	}

	client.ExpectHeaderByNumber(nil, &headers[4], nil)
	rpc.ExpectBatchCallContext(rpcElems, nil)
	headers, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	// Build on the wrong previous header, corrupting hashes
	prevHeader := headers[len(headers)-2]
	prevHeader.Number = headers[len(headers)-1].Number
	headers = makeHeaders(5, &prevHeader)
	rpcElems = makeHeaderRpcElems(headers[0].Number, headers[4].Number)
	for i := 0; i < len(rpcElems); i++ {
		rpcElems[i].Result = &headers[i]
	}

	// More headers are available (Latest == 9), but the mismatches will the last
	// traversed header
	client.ExpectHeaderByNumber(nil, &types.Header{Number: big.NewInt(9)}, nil)
	rpc.ExpectBatchCallContext(rpcElems[:2], nil)
	headers, err = headerTraversal.NextHeaders(2)
	require.Nil(t, headers)
	require.Equal(t, ErrHeaderTraversalAndProviderMismatchedState, err)
}
