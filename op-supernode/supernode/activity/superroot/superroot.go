package superroot

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/internal/syncstatus"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Superroot composes the super-root at a given timestamp across the
// configured dep set, returning aggregated sync status and the per-chain
// optimistic outputs alongside.
type Superroot struct {
	log      gethlog.Logger
	chains   map[eth.ChainID]cc.ChainContainer
	verified interop.VerifiedResultReader
}

func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer, verified interop.VerifiedResultReader) *Superroot {
	return &Superroot{
		log:      log,
		chains:   chains,
		verified: verified,
	}
}

func (s *Superroot) Name() string { return "superroot" }

// Reset is a no-op for superroot - it always queries chain containers directly
// and doesn't maintain any chain-specific cached state.
func (s *Superroot) Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
	// No-op: superroot queries chain containers directly
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

	result, verifierL1, vrErr := s.verified.VerifiedResultAtTimestamp(timestamp)

	// Any chain-level error building the optimistic branch must fail the
	// call: op-challenger reads OptimisticAtTimestamp at step>0 and a silent
	// partial map would produce permanent InvalidTransition commitments on
	// chain. The pre-interop regime also relies on this map for Data.
	optimisticBranch, err := s.buildOptimisticBranch(ctx, timestamp)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("build optimistic branch at %d: %w", timestamp, err)
	}

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 aggregate.CurrentL1,
		CurrentSafeTimestamp:      aggregate.SafeTimestamp,
		CurrentLocalSafeTimestamp: aggregate.LocalSafeTimestamp,
		CurrentFinalizedTimestamp: aggregate.FinalizedTimestamp,
		OptimisticAtTimestamp:     optimisticBranch,
		ChainIDs:                  aggregate.ChainIDs,
	}

	// When interop is engaged (verified entry or active-but-not-yet-verified
	// regimes), use the lower of aggregate.CurrentL1 and the verifier's
	// CurrentL1 observed atomically with the verifiedDB read. Reporting the
	// snapshot L1 closes the race where aggregate's read happens to straddle
	// a commit (entry-now-present but currentL1 not yet observed) or a
	// rewind (currentL1 already dropped but aggregate captured a stale high
	// value).
	switch {
	case vrErr == nil:
		if verifierL1.Number < response.CurrentL1.Number {
			response.CurrentL1 = verifierL1
		}
		data, derr := s.composeVerifiedData(ctx, timestamp, result)
		if derr != nil {
			return eth.SuperRootAtTimestampResponse{}, derr
		}
		response.Data = data
		return response, nil
	case errors.Is(vrErr, ethereum.NotFound):
		// Interop active at T but no VerifiedResult yet. Leave Data nil;
		// the verifiedDB write gate guarantees CurrentL1 has not reached
		// VerifiedRequiredL1(T).
		if verifierL1.Number < response.CurrentL1.Number {
			response.CurrentL1 = verifierL1
		}
		return response, nil
	case errors.Is(vrErr, interop.ErrNotActive), errors.Is(vrErr, interop.ErrBeforeVerifiedDB), errors.Is(vrErr, interop.ErrNotStarted):
		// Optimistic outputs are canonical: pre-interop consensus,
		// safe-head handoff, or pre-Start retry path.
		response.Data = composeHandoffDataFromOptimistic(timestamp, s.chains, optimisticBranch)
		return response, nil
	default:
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("read verifiedDB at %d: %w", timestamp, vrErr)
	}
}

// buildOptimisticBranch gathers per-chain optimistic outputs. Chains that
// return NotFound on OptimisticOutputAtTimestamp or OptimisticAt are omitted
// (chain hasn't derived T yet); any other error fails the call.
func (s *Superroot) buildOptimisticBranch(ctx context.Context, timestamp uint64) (map[eth.ChainID]eth.OutputWithRequiredL1, error) {
	out := make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
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
		out[chainID] = eth.OutputWithRequiredL1{
			Output:     optimisticOut,
			OutputRoot: eth.OutputRoot(optimisticOut),
			RequiredL1: optimisticL1,
		}
	}
	return out, nil
}

// composeVerifiedData composes the post-interop Data from a VerifiedResult.
// The verifiedDB entry pins per-chain canonical block hashes; per-chain output
// roots come from a by-hash read against the L2 EL.
func (s *Superroot) composeVerifiedData(ctx context.Context, timestamp uint64, result interop.VerifiedResult) (*eth.SuperRootResponseData, error) {
	// Reject a dep-set mismatch: either side would yield a super root that
	// disagrees with peers running the full dep set.
	if len(result.L2Heads) != len(s.chains) {
		return nil, fmt.Errorf("dep-set size mismatch at %d: verifiedDB=%d chains, boot view=%d chains", timestamp, len(result.L2Heads), len(s.chains))
	}

	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(s.chains))
	for chainID, chain := range s.chains {
		head, ok := result.L2Heads[chainID]
		if !ok {
			return nil, fmt.Errorf("verifiedDB entry at %d missing chain %s — dep-set mismatch", timestamp, chainID)
		}
		outRoot, err := chain.OutputRootAtL2BlockHash(ctx, head.Hash)
		if err != nil {
			return nil, fmt.Errorf("output root for chain %s at block %s: %w", chainID, head.Hash, err)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
	}
	super := eth.NewSuperV1(timestamp, chainOutputs...)
	return &eth.SuperRootResponseData{
		VerifiedRequiredL1: result.L1Inclusion,
		Super:              super,
		SuperRoot:          eth.SuperRoot(super),
	}, nil
}

// composeHandoffDataFromOptimistic builds Data from the optimistic map. Used
// for pre-activation and below-firstVerifiable regimes, where the safe-head
// handoff guarantees the optimistic outputs are canonical. Returns nil if
// any chain is missing from the map (chain hasn't derived T).
func composeHandoffDataFromOptimistic(timestamp uint64, chains map[eth.ChainID]cc.ChainContainer, optimisticBranch map[eth.ChainID]eth.OutputWithRequiredL1) *eth.SuperRootResponseData {
	if len(optimisticBranch) != len(chains) {
		return nil
	}
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(chains))
	var maxL1 eth.BlockID
	for chainID := range chains {
		entry, ok := optimisticBranch[chainID]
		if !ok {
			return nil
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: entry.OutputRoot})
		if entry.RequiredL1.Number > maxL1.Number {
			maxL1 = entry.RequiredL1
		}
	}
	super := eth.NewSuperV1(timestamp, chainOutputs...)
	return &eth.SuperRootResponseData{
		VerifiedRequiredL1: maxL1,
		Super:              super,
		SuperRoot:          eth.SuperRoot(super),
	}
}
