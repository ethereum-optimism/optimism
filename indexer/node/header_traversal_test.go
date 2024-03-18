package node

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/stretchr/testify/mock"
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
	client := new(MockEthClient)

	// start from block 10 as the latest fetched block
	LastTraversedHeader := &types.Header{Number: big.NewInt(10)}
	headerTraversal := NewHeaderTraversal(client, LastTraversedHeader, bigint.Zero)

	require.Nil(t, headerTraversal.LatestHeader())
	require.NotNil(t, headerTraversal.LastTraversedHeader())

	// no new headers when matched with head
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(LastTraversedHeader, nil)
	headers, err := headerTraversal.NextHeaders(100)
	require.NoError(t, err)
	require.Empty(t, headers)

	require.NotNil(t, headerTraversal.LatestHeader())
	require.NotNil(t, headerTraversal.LastTraversedHeader())
	require.Equal(t, LastTraversedHeader.Number.Uint64(), headerTraversal.LatestHeader().Number.Uint64())
}

func TestHeaderTraversalNextHeadersCursored(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)

	headers := makeHeaders(10, nil)

	// blocks [0..4]. Latest reported is 7
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(&headers[7], nil).Times(1) // Times so that we can override next
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(0)), mock.MatchedBy(bigint.Matcher(4))).Return(headers[:5], nil)
	_, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)

	require.Equal(t, uint64(7), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(4), headerTraversal.LastTraversedHeader().Number.Uint64())

	// blocks [5..9]. Latest Reported is 9
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(&headers[9], nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(5)), mock.MatchedBy(bigint.Matcher(9))).Return(headers[5:], nil)
	_, err = headerTraversal.NextHeaders(5)
	require.NoError(t, err)

	require.Equal(t, uint64(9), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(9), headerTraversal.LastTraversedHeader().Number.Uint64())
}

func TestHeaderTraversalNextHeadersMaxSize(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)

	// 100 "available" headers
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(&types.Header{Number: big.NewInt(100)}, nil)

	// clamped by the supplied size
	headers := makeHeaders(5, nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(0)), mock.MatchedBy(bigint.Matcher(4))).Return(headers, nil)
	headers, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	require.Equal(t, uint64(100), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(4), headerTraversal.LastTraversedHeader().Number.Uint64())

	// clamped by the supplied size. FinalizedHeight == 100
	headers = makeHeaders(10, &headers[len(headers)-1])
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(5)), mock.MatchedBy(bigint.Matcher(14))).Return(headers, nil)
	headers, err = headerTraversal.NextHeaders(10)
	require.NoError(t, err)
	require.Len(t, headers, 10)

	require.Equal(t, uint64(100), headerTraversal.LatestHeader().Number.Uint64())
	require.Equal(t, uint64(14), headerTraversal.LastTraversedHeader().Number.Uint64())
}

func TestHeaderTraversalMismatchedProviderStateError(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	headerTraversal := NewHeaderTraversal(client, nil, bigint.Zero)

	// blocks [0..4]
	headers := makeHeaders(5, nil)
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(&headers[4], nil).Times(1) // Times so that we can override next
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(0)), mock.MatchedBy(bigint.Matcher(4))).Return(headers, nil)
	headers, err := headerTraversal.NextHeaders(5)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	// blocks [5..9]. Next batch is not chained correctly (starts again from genesis)
	headers = makeHeaders(5, nil)
	client.On("BlockHeaderByNumber", (*big.Int)(nil)).Return(&types.Header{Number: big.NewInt(9)}, nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigint.Matcher(5)), mock.MatchedBy(bigint.Matcher(9))).Return(headers, nil)
	headers, err = headerTraversal.NextHeaders(5)
	require.Nil(t, headers)
	require.Equal(t, ErrHeaderTraversalAndProviderMismatchedState, err)
}
