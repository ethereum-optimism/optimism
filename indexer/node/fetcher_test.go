package node

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/core/types"
)

func bigIntMatcher(num int64) func(*big.Int) bool {
	return func(bi *big.Int) bool { return bi.Int64() == num }
}

func TestNextFinalizedHeadersNoOp(t *testing.T) {
	client := new(MockEthClient)
	fetcher, err := NewFetcher(client, big.NewInt(1))
	assert.NoError(t, err)

	// no new headers
	client.On("FinalizedBlockHeight").Return(big.NewInt(1), nil)
	headers, err := fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Empty(t, headers)
}

func TestNextFinalizedHeadersCursor(t *testing.T) {
	client := new(MockEthClient)
	fetcher, err := NewFetcher(client, big.NewInt(1))
	assert.NoError(t, err)

	// 5 available headers [1..5]
	client.On("FinalizedBlockHeight").Return(big.NewInt(5), nil)

	headers := make([]*types.Header, 5)
	for i := range headers {
		headers[i] = new(types.Header)
	}

	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(1)), mock.MatchedBy(bigIntMatcher(5))).Return(headers, nil)

	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, 5)

	// [1.. 5] nextHeight == 6
	assert.Equal(t, fetcher.nextStartingBlockHeight.Int64(), int64(6))
}

func TestNextFinalizedHeadersMaxHeaderBatch(t *testing.T) {
	client := new(MockEthClient)
	fetcher, err := NewFetcher(client, big.NewInt(1))
	assert.NoError(t, err)

	client.On("FinalizedBlockHeight").Return(big.NewInt(2*maxHeaderBatchSize), nil)

	headers := make([]*types.Header, maxHeaderBatchSize)
	for i := range headers {
		headers[i] = new(types.Header)
	}

	// clamped by the max batch size
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(1)), mock.MatchedBy(bigIntMatcher(maxHeaderBatchSize))).Return(headers, nil)

	headers, err = fetcher.NextFinalizedHeaders()
	assert.NoError(t, err)
	assert.Len(t, headers, maxHeaderBatchSize)

	// [1..maxHeaderBatchSize], nextHeight == 1+maxHeaderBatchSize
	assert.Equal(t, fetcher.nextStartingBlockHeight.Int64(), int64(1+maxHeaderBatchSize))
}
