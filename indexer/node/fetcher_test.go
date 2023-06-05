package node

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/core/types"
)

// make a set of headers which chain correctly
func makeHeaders(numHeaders uint64, prevHeader *types.Header) []*types.Header {
	headers := make([]*types.Header, numHeaders)
	for i := range headers {
		if i == 0 {
			if prevHeader == nil {
				// genesis
				headers[i] = &types.Header{Number: big.NewInt(0)}
			} else {
				// chain onto the previous header
				headers[i] = &types.Header{Number: big.NewInt(prevHeader.Number.Int64() + 1)}
				headers[i].ParentHash = prevHeader.Hash()
			}
		} else {
			prevHeader = headers[i-1]
			headers[i] = &types.Header{Number: big.NewInt(prevHeader.Number.Int64() + 1)}
			headers[i].ParentHash = prevHeader.Hash()
		}
	}

	return headers
}

func TestFetcherNextFinalizedHeadersNoOp(t *testing.T) {
	client := new(MockEthClient)

	// start from block 0 as the latest fetched block
	lastHeader := &types.Header{Number: bigZero}
	fetcher, err := NewFetcher(client, lastHeader)
	assert.NoError(t, err)

	// no new headers when matched with head
	client.On("FinalizedBlockHeight").Return(big.NewInt(0), nil)
	headers, err := fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Empty(t, headers)
}

func TestFetcherNextFinalizedHeadersCursored(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	fetcher, err := NewFetcher(client, nil)
	assert.NoError(t, err)

	// blocks [0..4]
	headers := makeHeaders(5, nil)
	client.On("FinalizedBlockHeight").Return(big.NewInt(4), nil).Times(1) // Times so that we can override next
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(4))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, 5)

	// blocks [5..9]
	headers = makeHeaders(5, headers[len(headers)-1])
	client.On("FinalizedBlockHeight").Return(big.NewInt(9), nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(5)), mock.MatchedBy(bigIntMatcher(9))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, 5)
}

func TestFetcherNextFinalizedHeadersMaxHeaderBatch(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	fetcher, err := NewFetcher(client, nil)
	assert.NoError(t, err)

	// blocks [0..maxBatchSize] size == maxBatchSize = 1
	headers := makeHeaders(maxHeaderBatchSize, nil)
	client.On("FinalizedBlockHeight").Return(big.NewInt(maxHeaderBatchSize), nil)

	// clamped by the max batch size
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(maxHeaderBatchSize-1))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, maxHeaderBatchSize)

	// blocks [maxBatchSize..maxBatchSize]
	headers = makeHeaders(1, headers[len(headers)-1])
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(maxHeaderBatchSize)), mock.MatchedBy(bigIntMatcher(maxHeaderBatchSize))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, 1)
}

func TestFetcherMismatchedProviderStateError(t *testing.T) {
	client := new(MockEthClient)

	// start from genesis
	fetcher, err := NewFetcher(client, nil)
	assert.NoError(t, err)

	// blocks [0..4]
	headers := makeHeaders(5, nil)
	client.On("FinalizedBlockHeight").Return(big.NewInt(4), nil).Times(1) // Times so that we can override next
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(4))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, 5)

	// blocks [5..9]. Next batch is not chained correctly (starts again from genesis)
	headers = makeHeaders(5, nil)
	client.On("FinalizedBlockHeight").Return(big.NewInt(9), nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(5)), mock.MatchedBy(bigIntMatcher(9))).Return(headers, nil)
	headers, err = fetcher.NextFinalizedHeaders()
	assert.Nil(t, headers)
	assert.Equal(t, ErrFetcherAndProviderMismatchedState, err)
}
