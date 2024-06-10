package derive

import (
	"context"
	"errors"
	"fmt"

	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// PlasmaDataSource is a data source that fetches inputs from a plasma DA provider given
// their onchain commitments. Same as CalldataSource it will keep attempting to fetch.
type PlasmaDataSource struct {
	log     log.Logger
	src     DataIter
	fetcher PlasmaInputFetcher
	l1      L1Fetcher
	id      eth.BlockID
	// keep track of a pending commitment so we can keep trying to fetch the input.
	comm plasma.CommitmentData
}

func NewPlasmaDataSource(log log.Logger, src DataIter, l1 L1Fetcher, fetcher PlasmaInputFetcher, id eth.BlockID) *PlasmaDataSource {
	return &PlasmaDataSource{
		log:     log,
		src:     src,
		fetcher: fetcher,
		l1:      l1,
		id:      id,
	}
}

func (s *PlasmaDataSource) Next(ctx context.Context) (eth.Data, error) {
	// Process origin syncs the challenge contract events and updates the local challenge states
	// before we can proceed to fetch the input data. This function can be called multiple times
	// for the same origin and noop if the origin was already processed. It is also called if
	// there is not commitment in the current origin.
	if err := s.fetcher.AdvanceL1Origin(ctx, s.l1, s.id); err != nil {
		if errors.Is(err, plasma.ErrReorgRequired) {
			return nil, NewResetError(fmt.Errorf("new expired challenge"))
		}
		return nil, NewTemporaryError(fmt.Errorf("failed to advance plasma L1 origin: %w", err))
	}

	if s.comm == nil {
		// the l1 source returns the input commitment for the batch.
		data, err := s.src.Next(ctx)
		if err != nil {
			return nil, err
		}

		if len(data) == 0 {
			return nil, NotEnoughData
		}
		// If the tx data type is not plasma, we forward it downstream to let the next
		// steps validate and potentially parse it as L1 DA inputs.
		if data[0] != plasma.TxDataVersion1 {
			return data, nil
		}

		// validate batcher inbox data is a commitment.
		// strip the transaction data version byte from the data before decoding.
		comm, err := plasma.DecodeCommitmentData(data[1:])
		if err != nil {
			s.log.Warn("invalid commitment", "commitment", data, "err", err)
			return nil, NotEnoughData
		}
		s.comm = comm
	}
	// use the commitment to fetch the input from the plasma DA provider.
	data, err := s.fetcher.GetInput(ctx, s.l1, s.comm, s.id)
	// GetInput may call for a reorg if the pipeline is stalled and the plasma DA manager
	// continued syncing origins detached from the pipeline origin.
	if errors.Is(err, plasma.ErrReorgRequired) {
		// challenge for a new previously derived commitment expired.
		return nil, NewResetError(err)
	} else if errors.Is(err, plasma.ErrExpiredChallenge) {
		// this commitment was challenged and the challenge expired.
		s.log.Warn("challenge expired, skipping batch", "comm", s.comm)
		s.comm = nil
		// skip the input
		return s.Next(ctx)
	} else if errors.Is(err, plasma.ErrMissingPastWindow) {
		return nil, NewCriticalError(fmt.Errorf("data for comm %x not available: %w", s.comm, err))
	} else if errors.Is(err, plasma.ErrPendingChallenge) {
		// continue stepping without slowing down.
		return nil, NotEnoughData
	} else if err != nil {
		// return temporary error so we can keep retrying.
		return nil, NewTemporaryError(fmt.Errorf("failed to fetch input data with comm %x from da service: %w", s.comm, err))
	}
	// inputs are limited to a max size to ensure they can be challenged in the DA contract.
	if s.comm.CommitmentType() == plasma.Keccak256CommitmentType && len(data) > plasma.MaxInputSize {
		s.log.Warn("input data exceeds max size", "size", len(data), "max", plasma.MaxInputSize)
		s.comm = nil
		return s.Next(ctx)
	}
	// reset the commitment so we can fetch the next one from the source at the next iteration.
	s.comm = nil
	return data, nil
}
