package node

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrHeaderTraversalAheadOfProvider            = errors.New("the HeaderTraversal's internal state is ahead of the provider")
	ErrHeaderTraversalAndProviderMismatchedState = errors.New("the HeaderTraversal and provider have diverged in state")
)

type HeaderTraversal struct {
	ethClient  EthClient
	lastHeader *types.Header
}

// NewHeaderTraversal instantiates a new instance of HeaderTraversal against the supplied rpc client.
// The HeaderTraversal will start fetching blocks starting from the supplied header unless
// nil, indicating genesis.
func NewHeaderTraversal(ethClient EthClient, fromHeader *types.Header) *HeaderTraversal {
	return &HeaderTraversal{ethClient: ethClient, lastHeader: fromHeader}
}

// NextFinalizedHeaders retrives the next set of headers that have been
// marked as finalized by the connected client, bounded by the supplied size
func (f *HeaderTraversal) NextFinalizedHeaders(maxSize uint64) ([]*types.Header, error) {
	finalizedBlockHeight, err := f.ethClient.FinalizedBlockHeight()
	if err != nil {
		return nil, err
	}

	if f.lastHeader != nil {
		cmp := f.lastHeader.Number.Cmp(finalizedBlockHeight)
		if cmp == 0 {
			return nil, nil
		} else if cmp > 0 {
			return nil, ErrHeaderTraversalAheadOfProvider
		}
	}

	nextHeight := bigZero
	if f.lastHeader != nil {
		nextHeight = new(big.Int).Add(f.lastHeader.Number, bigOne)
	}

	endHeight := clampBigInt(nextHeight, finalizedBlockHeight, maxSize)
	headers, err := f.ethClient.BlockHeadersByRange(nextHeight, endHeight)
	if err != nil {
		return nil, err
	}

	numHeaders := len(headers)
	if numHeaders == 0 {
		return nil, nil
	} else if f.lastHeader != nil && headers[0].ParentHash != f.lastHeader.Hash() {
		// The indexer's state is in an irrecoverable state relative to the provider. This
		// should never happen since the indexer is dealing with only finalized blocks.
		return nil, ErrHeaderTraversalAndProviderMismatchedState
	}

	f.lastHeader = headers[numHeaders-1]
	return headers, nil
}
