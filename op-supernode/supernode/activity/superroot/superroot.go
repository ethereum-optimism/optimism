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
//
// StoredOptimisticAtTimestamp returns the per-chain optimistic snapshot
// captured at the moment of verification. Serving from this durable snapshot
// avoids races against chain rewinds and pipeline resets between the verifier
// observing the chains and the RPC response being constructed.
//
// VerifierCurrentL1 returns the verifier's frontier L1, used by callers like
// op-challenger to decide whether all L1 data up to the game's L1 head has
// been processed. Returns ok=false before the verifier has populated it.
type VerifiedSuperRootProvider interface {
	StoredSuperRootAtTimestamp(ts uint64) (data *eth.SuperRootResponseData, found bool, err error)
	StoredOptimisticAtTimestamp(ts uint64) (optimistic map[eth.ChainID]eth.OutputWithRequiredL1, found bool, err error)
	VerifierCurrentL1() (eth.BlockID, bool)
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
		optimistic   map[eth.ChainID]eth.OutputWithRequiredL1
		verifiedData *eth.SuperRootResponseData
	)

	// Default CurrentL1 to the live aggregate; overridden below with the
	// verifier's durable frontier when available.
	currentL1 := aggregate.CurrentL1

	if s.verifiedProvider != nil {
		data, found, err := s.verifiedProvider.StoredSuperRootAtTimestamp(timestamp)
		if err != nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get stored verified superroot: %w", err)
		}
		if found {
			verifiedData = data
		}

		stored, found, err := s.verifiedProvider.StoredOptimisticAtTimestamp(timestamp)
		if err != nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get stored optimistic snapshot: %w", err)
		}
		if found {
			optimistic = stored
		}

		// Prefer the verifier's frontier L1 over the live aggregate. The frontier
		// is what op-challenger needs to decide whether the game's L1 head has
		// been fully processed; it advances every Wait round so it tracks live
		// L1 progress within a few blocks even between L2 batches.
		if l1, ok := s.verifiedProvider.VerifierCurrentL1(); ok {
			currentL1 = l1
		}
	}

	// Fall back to live state for timestamps the verifier has not yet committed
	// (e.g. the in-flight pre-verification window). Once the verifier records
	// a timestamp, the durable snapshot above is the source of truth.
	if optimistic == nil {
		optimistic, err = s.collectLiveOptimistic(ctx, timestamp)
		if err != nil {
			return eth.SuperRootAtTimestampResponse{}, err
		}
	}

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 currentL1,
		CurrentSafeTimestamp:      aggregate.SafeTimestamp,
		CurrentLocalSafeTimestamp: aggregate.LocalSafeTimestamp,
		CurrentFinalizedTimestamp: aggregate.FinalizedTimestamp,
		OptimisticAtTimestamp:     optimistic,
		ChainIDs:                  aggregate.ChainIDs,
		Data:                      verifiedData,
	}
	return response, nil
}

// collectLiveOptimistic queries each chain for its optimistic head at the
// timestamp. Used only when the verifier has no durable snapshot yet (the
// pre-verification window). For verified timestamps the durable snapshot is
// preferred — see StoredOptimisticAtTimestamp.
func (s *Superroot) collectLiveOptimistic(ctx context.Context, timestamp uint64) (map[eth.ChainID]eth.OutputWithRequiredL1, error) {
	optimistic := make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
	for chainID, chain := range s.chains {
		optimisticOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			continue
		} else if err != nil {
			s.log.Warn("failed to get optimistic block", "chain_id", chainID.String(), "err", err)
			return nil, fmt.Errorf("failed to get optimistic block at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		_, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			continue
		} else if err != nil {
			s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
			return nil, fmt.Errorf("failed to get optimistic source L1 at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		optimistic[chainID] = eth.OutputWithRequiredL1{
			Output:     optimisticOut,
			OutputRoot: eth.OutputRoot(optimisticOut),
			RequiredL1: optimisticL1,
		}
	}
	return optimistic, nil
}
