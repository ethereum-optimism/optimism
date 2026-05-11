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

// Superroot satisfies the RPC Activity interface. It composes the super-root
// at a given timestamp across the configured dep set, returning aggregated
// sync status and the per-chain optimistic outputs alongside.
//
// Two regimes coexist in atTimestamp:
//   - Post-interop (T past activation and firstVerifiable): Data sourced
//     strictly from interop.VerifiedResultReader + EL-by-hash reads. Bound by
//     the cross-safe write gate; satisfies the invariant in §1/§2 of the
//     superroot_atTimestamp design docs.
//   - Pre-interop (T before activation, or interop not configured): Data
//     composed directly from the per-chain optimistic outputs. Pre-interop,
//     local-safe blocks cannot be invalidated, so the optimistic outputs ARE
//     the canonical outputs at T. Consistency matches the legacy two-call
//     client pattern.
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

	// Resolve regime first: a single in-memory check plus at most one bbolt
	// View. Does not call into any chain container.
	result, vrErr := s.verified.VerifiedResultAtTimestamp(timestamp)

	// Build the optimistic branch. Matches the existing semantics: chains
	// that hit NotFound on either OptimisticOutputAtTimestamp or OptimisticAt
	// are omitted; any other chain-level error fails the call. This is
	// load-bearing for the pre-interop regime (Data is built from this map)
	// and important for the post-interop regimes too — op-challenger reads
	// the OptimisticAtTimestamp map at step>0 and a silent partial map would
	// produce permanent InvalidTransition commitments on chain.
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

	switch {
	case vrErr == nil:
		data, derr := s.composeVerifiedData(ctx, timestamp, result, aggregate)
		if derr != nil {
			return eth.SuperRootAtTimestampResponse{}, derr
		}
		response.Data = data
		return response, nil
	case errors.Is(vrErr, ethereum.NotFound):
		// Interop is active at T but the verifier has not committed a
		// VerifiedResult yet. Data == nil — by construction of the
		// verifiedDB write gate plus the CurrentL1 cap, entry absence
		// implies CurrentL1 < VerifiedRequiredL1(T).
		return response, nil
	case errors.Is(vrErr, interop.ErrNotActive):
		// Pre-interop fallback. Reuse the optimistic map: pre-interop,
		// local-safe outputs cannot be invalidated, so the optimistic
		// outputs are canonical.
		response.Data = composePreInteropDataFromOptimistic(timestamp, s.chains, optimisticBranch)
		return response, nil
	default:
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("read verifiedDB at %d: %w", timestamp, vrErr)
	}
}

// buildOptimisticBranch iterates s.chains and gathers per-chain optimistic
// outputs. Chains whose OptimisticOutputAtTimestamp or OptimisticAt returned
// NotFound are silently omitted (legacy "chain hasn't derived T yet"). Any
// other error is returned immediately so the caller can surface it as a
// transport error to the RPC client.
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

// composeVerifiedData composes the strict (post-interop) Data from a
// VerifiedResult. The verifiedDB entry pins per-chain canonical block hashes;
// per-chain output roots come from a by-hash read against the L2 EL.
func (s *Superroot) composeVerifiedData(ctx context.Context, timestamp uint64, result interop.VerifiedResult, aggregate eth.SuperNodeSyncStatusResponse) (*eth.SuperRootResponseData, error) {
	// Dep-set sanity. Either direction is a configuration divergence:
	// - missing chain (s.chains has key not in L2Heads): partial answer.
	// - extra chain (L2Heads has key not in s.chains): peers with the full
	//   dep set would compute a different super root → §2 Agreement violation.
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
	data := &eth.SuperRootResponseData{
		VerifiedRequiredL1: result.L1Inclusion,
		Super:              super,
		SuperRoot:          eth.SuperRoot(super),
	}

	// Rewind-race coupling check. Verifier rewind drops i.currentL1 to zero
	// (interop.go:649) before deleting verifiedDB rows. If we read
	// verifiedDB before the rewind got to the delete step, the aggregate
	// CurrentL1 will reflect the post-currentL1=0 value. Catch that here.
	if aggregate.CurrentL1.Number < data.VerifiedRequiredL1.Number {
		return nil, fmt.Errorf("rewind in flight at %d: CurrentL1=%d < VerifiedRequiredL1=%d", timestamp, aggregate.CurrentL1.Number, data.VerifiedRequiredL1.Number)
	}
	return data, nil
}

// composePreInteropDataFromOptimistic builds Data by reusing the optimistic
// map. Returns nil if the optimistic map is short (any chain in s.chains is
// missing — legacy "chain hasn't derived" → Data == nil).
func composePreInteropDataFromOptimistic(timestamp uint64, chains map[eth.ChainID]cc.ChainContainer, optimisticBranch map[eth.ChainID]eth.OutputWithRequiredL1) *eth.SuperRootResponseData {
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
