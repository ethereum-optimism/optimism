package node

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBufferedHeaderTraversalNextFinalizedHeaders(t *testing.T) {
	client := new(MockEthClient)
	bufferedHeaderTraversal := NewBufferedHeaderTraversal(client, nil)

	// buffer 10 blocks
	headers := makeHeaders(10, nil)
	client.On("FinalizedBlockHeight").Return(big.NewInt(10), nil)
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(9))).Return(headers, nil)
	headers, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headers, 10)

	// next call returns the same headers
	sameHeaders, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.ElementsMatch(t, sameHeaders, headers)

	// subset reuses the same internal buffer
	subsetHeaders, err := bufferedHeaderTraversal.NextFinalizedHeaders(5)
	assert.NoError(t, err)
	assert.ElementsMatch(t, subsetHeaders[:5], headers[:5])
}

func TestBufferedHeaderTraversalExpandingBuffer(t *testing.T) {
	client := new(MockEthClient)
	bufferedHeaderTraversal := NewBufferedHeaderTraversal(client, nil)

	client.On("FinalizedBlockHeight").Return(big.NewInt(100), nil)
	headers := makeHeaders(20, nil)

	// buffer 10 blocks
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(9))).Return(headers[:10], nil)
	headerBatch, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)

	// expand buffer to 20 blocks
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(10)), mock.MatchedBy(bigIntMatcher(19))).Return(headers[10:], nil)
	headerBatch, err = bufferedHeaderTraversal.NextFinalizedHeaders(20)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 20)

	// covers the full list of headers
	assert.ElementsMatch(t, headers, headerBatch)
}

func TestBufferedHeaderTraversalExpandingBufferFailures(t *testing.T) {
	client := new(MockEthClient)
	bufferedHeaderTraversal := NewBufferedHeaderTraversal(client, nil)

	client.On("FinalizedBlockHeight").Return(big.NewInt(100), nil)
	headers := makeHeaders(20, nil)

	// buffer 10 blocks
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(9))).Return(headers[:10], nil)
	headerBatch, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)

	// buffer expansion fails. Returns current buffer as is
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(10)), mock.MatchedBy(bigIntMatcher(19))).Return([]*types.Header{}, errors.New("boom"))
	headerBatch, err = bufferedHeaderTraversal.NextFinalizedHeaders(20)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)
}

func TestBufferedHeaderTraversalAdvance(t *testing.T) {
	client := new(MockEthClient)
	bufferedHeaderTraversal := NewBufferedHeaderTraversal(client, nil)

	client.On("FinalizedBlockHeight").Return(big.NewInt(100), nil)
	headers := makeHeaders(20, nil)

	// observe & buffer 10 blocks
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(9))).Return(headers[:10], nil)
	headerBatch, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)
	assert.ElementsMatch(t, headers[:10], headerBatch)

	// advance past the first 5
	err = bufferedHeaderTraversal.Advance(headers[4])
	assert.NoError(t, err)

	// 5 remaining headers that are internally buffered
	headerBatch, err = bufferedHeaderTraversal.NextFinalizedHeaders(5)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 5)
	assert.ElementsMatch(t, headers[5:10], headerBatch)

	// empty the buffer by advancing to the last (10th) buffer
	err = bufferedHeaderTraversal.Advance(headers[9])
	assert.NoError(t, err)

	// Next 10 requires an expansion of an empty internal buffer
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(10)), mock.MatchedBy(bigIntMatcher(19))).Return(headers[10:], nil)
	headerBatch, err = bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)
	assert.ElementsMatch(t, headers[10:], headerBatch)
}

func TestBufferedHeaderTraversalInvalidAdvance(t *testing.T) {
	client := new(MockEthClient)
	bufferedHeaderTraversal := NewBufferedHeaderTraversal(client, nil)

	client.On("FinalizedBlockHeight").Return(big.NewInt(100), nil)
	headers := makeHeaders(20, nil)

	// observe & buffer 10 blocks
	client.On("BlockHeadersByRange", mock.MatchedBy(bigIntMatcher(0)), mock.MatchedBy(bigIntMatcher(9))).Return(headers[:10], nil)
	headerBatch, err := bufferedHeaderTraversal.NextFinalizedHeaders(10)
	assert.NoError(t, err)
	assert.Len(t, headerBatch, 10)
	assert.ElementsMatch(t, headers[:10], headerBatch)

	// advance past the first 5
	err = bufferedHeaderTraversal.Advance(headers[4])
	assert.NoError(t, err)

	// advance to the same header
	err = bufferedHeaderTraversal.Advance(headers[4])
	assert.Error(t, err)
	assert.Equal(t, ErrBufferedHeaderTraversalInvalidAdvance, err)

	// advance to a past header
	err = bufferedHeaderTraversal.Advance(headers[0])
	assert.Error(t, err)
	assert.Equal(t, ErrBufferedHeaderTraversalInvalidAdvance, err)

	// non-buffered header
	err = bufferedHeaderTraversal.Advance(headers[10])
	assert.Error(t, err)
	assert.Equal(t, ErrBufferedHeaderTraversalInvalidAdvance, err)

	// valid header number but a different block hash
	fakeHeader := &types.Header{Number: big.NewInt(6)}
	assert.Equal(t, headers[6].Number, fakeHeader.Number)
	assert.NotEqual(t, headers[6].Hash(), fakeHeader.Hash())
	err = bufferedHeaderTraversal.Advance(fakeHeader)
	assert.Error(t, err)
	assert.Equal(t, ErrBufferedHeaderTraversalInvalidAdvance, err)
}
