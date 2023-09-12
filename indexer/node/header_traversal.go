package node

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bigint"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrHeaderTraversalAheadOfProvider            = errors.New("the HeaderTraversal's internal state is ahead of the provider")
	ErrHeaderTraversalAndProviderMismatchedState = errors.New("the HeaderTraversal and provider have diverged in state")
)

type HeaderTraversal struct {
	ethClient EthClient

	lastHeader             *types.Header
	blockConfirmationDepth *big.Int
}

// NewHeaderTraversal instantiates a new instance of HeaderTraversal against the supplied rpc client.
// The HeaderTraversal will start fetching blocks starting from the supplied header unless nil, indicating genesis.
func NewHeaderTraversal(ethClient EthClient, fromHeader *types.Header, confDepth *big.Int) *HeaderTraversal {
	return &HeaderTraversal{ethClient: ethClient, lastHeader: fromHeader, blockConfirmationDepth: confDepth}
}

// LastHeader returns the last header that was fetched by the HeaderTraversal
// This is useful for testing the state of the HeaderTraversal
func (f *HeaderTraversal) LastHeader() *types.Header {
	return f.lastHeader
}

// NextFinalizedHeaders retrives the next set of headers that have been
// marked as finalized by the connected client, bounded by the supplied size
func (f *HeaderTraversal) NextFinalizedHeaders(maxSize uint64) ([]types.Header, error) {
	latestBlockHeader, err := f.ethClient.BlockHeaderByNumber(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to query latest block: %w", err)
	}

	endHeight := new(big.Int).Sub(latestBlockHeader.Number, f.blockConfirmationDepth)
	if endHeight.Sign() < 0 {
		// No blocks with the provided confirmation depth available
		return nil, nil
	}

	if f.lastHeader != nil {
		cmp := f.lastHeader.Number.Cmp(endHeight)
		if cmp == 0 {
			return nil, nil
		} else if cmp > 0 {
			return nil, ErrHeaderTraversalAheadOfProvider
		}
	}

	nextHeight := bigint.Zero
	if f.lastHeader != nil {
		nextHeight = new(big.Int).Add(f.lastHeader.Number, bigint.One)
	}

	endHeight = bigint.Clamp(nextHeight, endHeight, maxSize)
	headers, err := f.ethClient.BlockHeadersByRange(nextHeight, endHeight)
	if err != nil {
		return nil, fmt.Errorf("error querying blocks by range: %w", err)
	}

	numHeaders := len(headers)
	if numHeaders == 0 {
		return nil, nil
	} else if f.lastHeader != nil && headers[0].ParentHash != f.lastHeader.Hash() {
		// The indexer's state is in an irrecoverable state relative to the provider. This
		// should never happen since the indexer is dealing with only finalized blocks.
		return nil, ErrHeaderTraversalAndProviderMismatchedState
	}

	f.lastHeader = &headers[numHeaders-1]
	return headers, nil
}
