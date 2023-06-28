package node

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// Max number of headers that's bee returned by the Fetcher at once.
const maxHeaderBatchSize = 50

var ErrFetcherAndProviderMismatchedState = errors.New("the fetcher and provider have diverged in finalized state")

type Fetcher struct {
	ethClient  EthClient
	lastHeader *types.Header
}

// NewFetcher instantiates a new instance of Fetcher against the supplied rpc client.
// The Fetcher will start fetching blocks starting from the supplied header unless
// nil, indicating genesis.
func NewFetcher(ethClient EthClient, fromHeader *types.Header) *Fetcher {
	return &Fetcher{ethClient: ethClient, lastHeader: fromHeader}
}

// NextConfirmedHeaders retrives the next set of headers that have been
// marked as finalized by the connected client
func (f *Fetcher) NextFinalizedHeaders() ([]*types.Header, error) {
	finalizedBlockHeight, err := f.ethClient.FinalizedBlockHeight()
	if err != nil {
		return nil, err
	}

	if f.lastHeader != nil && f.lastHeader.Number.Cmp(finalizedBlockHeight) >= 0 {
		// Warn if our fetcher is ahead of the provider. The fetcher should always
		// be behind or at head with the provider.
		return nil, nil
	}

	nextHeight := bigZero
	if f.lastHeader != nil {
		nextHeight = new(big.Int).Add(f.lastHeader.Number, bigOne)
	}

	endHeight := clampBigInt(nextHeight, finalizedBlockHeight, maxHeaderBatchSize)
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
		return nil, ErrFetcherAndProviderMismatchedState
	}

	f.lastHeader = headers[numHeaders-1]
	return headers, nil
}
