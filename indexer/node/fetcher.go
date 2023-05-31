package node

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// Max number of headers that's bee returned by the Fetcher at once. This will
	// eventually be configurable
	maxHeaderBatchSize = 50
)

type Fetcher struct {
	ethClient EthClient

	// TODO: Store the last header block hash to ensure
	// the next batch of headers builds on top
	nextStartingBlockHeight *big.Int
}

// NewFetcher instantiates a new instance of Fetcher against the supplied rpc client.
// The Fetcher will start retrieving blocks starting at `fromBlockHeight`.
func NewFetcher(ethClient EthClient, fromBlockHeight *big.Int) (*Fetcher, error) {
	fetcher := &Fetcher{
		ethClient:               ethClient,
		nextStartingBlockHeight: fromBlockHeight,
	}

	return fetcher, nil
}

// NextConfirmedHeaders retrives the next set of headers that have been
// marked as finalized by the connected client
func (f *Fetcher) NextFinalizedHeaders() ([]*types.Header, error) {
	finalizedBlockHeight, err := f.ethClient.FinalizedBlockHeight()
	if err != nil {
		return nil, err
	}

	// TODO:
	//  - (unlikely) What do we do if our connected node is suddently behind by many blocks?
	if f.nextStartingBlockHeight.Cmp(finalizedBlockHeight) >= 0 {
		return nil, nil
	}

	// clamp to the max batch size. the range is inclusive so +1 when computing the count
	endHeight := finalizedBlockHeight
	count := new(big.Int).Sub(endHeight, f.nextStartingBlockHeight).Uint64() + 1
	if count > maxHeaderBatchSize {
		endHeight = new(big.Int).Add(f.nextStartingBlockHeight, big.NewInt(maxHeaderBatchSize-1))
	}

	headers, err := f.ethClient.BlockHeadersByRange(f.nextStartingBlockHeight, endHeight)
	if err != nil {
		return nil, err
	}

	numHeaders := int64(len(headers))
	f.nextStartingBlockHeight = endHeight.Add(f.nextStartingBlockHeight, big.NewInt(numHeaders))
	return headers, nil
}
