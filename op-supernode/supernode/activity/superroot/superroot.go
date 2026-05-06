package superroot

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/internal/syncstatus"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Superroot satisfies the RPC Activity interface
// it provides the superroot at a given timestamp for all chains
// along with the current L1s and the verified and optimistic L1:L2 pairs
type Superroot struct {
	log              gethlog.Logger
	chains           map[eth.ChainID]cc.ChainContainer
	verifiedProvider VerifiedSuperRootProvider
}

// VerifiedSuperRootProvider supplies durable verifier-owned superroot data.
type VerifiedSuperRootProvider interface {
	StoredSuperRootAtTimestamp(ts uint64) (data *eth.SuperRootResponseData, found bool, err error)
}

func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer, provider VerifiedSuperRootProvider) *Superroot {
	return &Superroot{
		log:              log,
		chains:           chains,
		verifiedProvider: provider,
	}
}

func (s *Superroot) Name() string { return "superroot" }

// Reset is a no-op for superroot: it does not maintain chain-specific cached state.
func (s *Superroot) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
}

func (s *Superroot) RPCNamespace() string    { return "superroot" }
func (s *Superroot) RPCService() interface{} { return &superrootAPI{s: s} }

type superrootAPI struct{ s *Superroot }

// AtTimestamp computes the super-root at the given timestamp, plus additional information about the current L1s, verified L2s, and optimistic L2s
func (api *superrootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	return api.s.atTimestamp(ctx, uint64(timestamp))
}

func (s *Superroot) atTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	aggregate, err := syncstatus.Aggregate(ctx, s.log, s.chains)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}

	var (
		optimistic   = make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
		verifiedData *eth.SuperRootResponseData
	)

	if s.verifiedProvider != nil {
		data, found, err := s.verifiedProvider.StoredSuperRootAtTimestamp(timestamp)
		if err != nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get stored verified superroot: %w", err)
		}
		if found {
			verifiedData = data
		}
	}

	// Collect optimistic data for all chains regardless of whether verified data is available.
	for chainID, chain := range s.chains {
		optimisticOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			// If optimistic data is also absent, the chain is simply excluded from OptimisticAtTimestamp.
			continue
		} else if err != nil {
			s.log.Warn("failed to get optimistic block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get optimistic block at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		// Also include the source L1 for context
		_, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			continue
		} else if err != nil {
			s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get optimistic source L1 at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		optimistic[chainID] = eth.OutputWithRequiredL1{
			Output:     optimisticOut,
			OutputRoot: eth.OutputRoot(optimisticOut),
			RequiredL1: optimisticL1,
		}
	}

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 aggregate.CurrentL1,
		CurrentSafeTimestamp:      aggregate.SafeTimestamp,
		CurrentLocalSafeTimestamp: aggregate.LocalSafeTimestamp,
		CurrentFinalizedTimestamp: aggregate.FinalizedTimestamp,
		OptimisticAtTimestamp:     optimistic,
		ChainIDs:                  aggregate.ChainIDs,
		Data:                      verifiedData,
	}
	return response, nil
}
