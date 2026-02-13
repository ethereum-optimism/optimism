package source

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var ErrNoSuperRootData = errors.New("supernode response has no super root data")

// SuperNodeClient is the interface for interacting with an op-supernode.
type SuperNodeClient interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
	Close()
}

// SuperNodeProposalSource fetches super root proposals from op-supernode instances.
// It supports multiple clients for fault tolerance, querying them in parallel for sync status
// and falling back to subsequent clients for proposals if earlier ones fail.
type SuperNodeProposalSource struct {
	log     log.Logger
	clients []SuperNodeClient
}

func NewSuperNodeProposalSource(logger log.Logger, clients ...SuperNodeClient) *SuperNodeProposalSource {
	if len(clients) == 0 {
		panic("no supernode clients provided")
	}
	return &SuperNodeProposalSource{
		log:     logger,
		clients: clients,
	}
}

type supernodeStatusResult struct {
	idx  int
	resp eth.SuperRootAtTimestampResponse
	err  error
}

// SyncStatus returns the current L1 view and the safe/finalized L2 timestamps as reported by the supernode.
// The timestamps are precomputed by the supernode as the conservative minimum across the dependency set.
func (s *SuperNodeProposalSource) SyncStatus(ctx context.Context) (SyncStatus, error) {
	now := uint64(time.Now().Unix())

	var wg sync.WaitGroup
	results := make(chan supernodeStatusResult, len(s.clients))
	wg.Add(len(s.clients))
	for i, client := range s.clients {
		go func() {
			defer wg.Done()
			resp, err := client.SuperRootAtTimestamp(ctx, now)
			results <- supernodeStatusResult{
				idx:  i,
				resp: resp,
				err:  err,
			}
		}()
	}
	wg.Wait()
	close(results)

	var errs []error
	var earliestResponse eth.SuperRootAtTimestampResponse
	var hasValidResponse bool
	for result := range results {
		if result.err != nil {
			s.log.Warn("Failed to retrieve sync status from supernode", "idx", result.idx, "err", result.err)
			errs = append(errs, result.err)
			continue
		}
		// Use the response with the lowest CurrentL1 block number (most conservative)
		if !hasValidResponse || result.resp.CurrentL1.Number < earliestResponse.CurrentL1.Number {
			earliestResponse = result.resp
			hasValidResponse = true
		}
	}

	if !hasValidResponse {
		return SyncStatus{}, fmt.Errorf("no available sync status sources: %w", errors.Join(errs...))
	}

	return SyncStatus{
		CurrentL1:   earliestResponse.CurrentL1,
		SafeL2:      earliestResponse.CurrentSafeTimestamp,
		FinalizedL2: earliestResponse.CurrentFinalizedTimestamp,
	}, nil
}

// ProposalAtSequenceNum fetches the super-root proposal at the given timestamp.
// It fails over across supernode clients until it finds a response with valid super-root data.
func (s *SuperNodeProposalSource) ProposalAtSequenceNum(ctx context.Context, timestamp uint64) (Proposal, error) {
	var errs []error
	for i, client := range s.clients {
		resp, err := client.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			errs = append(errs, err)
			s.log.Warn("Failed to retrieve proposal from supernode", "idx", i, "err", err)
			continue
		}

		if resp.Data == nil {
			errs = append(errs, fmt.Errorf("%w: timestamp %d from supernode idx %d", ErrNoSuperRootData, timestamp, i))
			s.log.Warn("Supernode response has no super root data", "idx", i, "timestamp", timestamp)
			continue
		}

		super := resp.Data.Super
		if super == nil {
			errs = append(errs, fmt.Errorf("super root data missing super value from supernode idx %d", i))
			s.log.Warn("Super root data missing super value from supernode", "idx", i, "timestamp", timestamp)
			continue
		}
		if ver := super.Version(); ver != eth.SuperRootVersionV1 {
			errs = append(errs, fmt.Errorf("unsupported super root version %d from supernode idx %d", ver, i))
			s.log.Warn("Unsupported super root version from supernode", "idx", i, "version", ver)
			continue
		}

		return Proposal{
			Root:        common.Hash(resp.Data.SuperRoot),
			SequenceNum: timestamp,
			Super:       super,
			CurrentL1: eth.BlockID{
				Hash:   resp.CurrentL1.Hash,
				Number: resp.CurrentL1.Number,
			},
			// Unsupported by super root proposals
			Legacy: LegacyProposalData{},
		}, nil
	}
	return Proposal{}, fmt.Errorf("no available proposal sources: %w", errors.Join(errs...))
}

func (s *SuperNodeProposalSource) Close() {
	for _, client := range s.clients {
		client.Close()
	}
}
