package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// PlasmaDataSource is a data source that fetches inputs from a plasma DA provider given
// their onchain commitments. Same as CalldataSource it will keep attempting to fetch.
type PlasmaDataSource struct {
	log     log.Logger
	src     DataIter
	fetcher PlasmaInputFetcher
	id      eth.BlockID
	// keep track of a pending commitment so we can keep trying to fetch the input.
	comm []byte
}

func NewPlasmaDataSource(log log.Logger, src DataIter, fetcher PlasmaInputFetcher, id eth.BlockID) *PlasmaDataSource {
	return &PlasmaDataSource{
		log:     log,
		src:     src,
		fetcher: fetcher,
		id:      id,
	}
}

func (s *PlasmaDataSource) Next(ctx context.Context) (eth.Data, error) {
	if s.comm == nil {
		var err error
		// the l1 source returns the input commitment for the batch.
		s.comm, err = s.src.Next(ctx)
		if err != nil {
			return nil, err
		}
	}
	// use the commitment to fetch the input from the plasma DA provider.
	resp, err := s.fetcher.GetInput(ctx, s.comm, s.id.Number)
	if err != nil {
		// return temporary error so we can keep retrying.
		return nil, NewTemporaryError(fmt.Errorf("failed to fetch input data with comm %x from da service: %w", s.comm, err))
	}
	// reset the commitment so we can fetch the next one from the source at the next iteration.
	s.comm = nil
	return resp.Data, nil
}
