package superroot

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// ErrInconsistentSnapshot is returned when atTimestamp detects that a chain's
// generation counter changed between the start and end of its work — meaning
// a virtual node restart, an engine rewind, or a derivation-pipeline reset
// happened during the gather. The data we read may straddle pre- and
// post-mutation state; callers should treat this as a transient retryable
// signal.
var ErrInconsistentSnapshot = errors.New("chain state changed during superroot gather")

// Superroot satisfies the RPC Activity interface
// it provides the superroot at a given timestamp for all chains
// along with the current L1s and the verified and optimistic L1:L2 pairs
type Superroot struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
}

func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Superroot {
	return &Superroot{
		log:    log,
		chains: chains,
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
	// Capture each chain's generation counter before any reads. ChainContainer
	// bumps Generation on every state-mutating event that could make data
	// gathered earlier inconsistent with state observed later (VN restart,
	// RewindEngine, inner-pipeline rollup.ResetEvent). After all reads we
	// re-read each counter; if any changed, we discard the response and let
	// the caller retry rather than return data that mixes pre- and
	// post-mutation state.
	chainIDs := make([]eth.ChainID, 0, len(s.chains))
	startGens := make(map[eth.ChainID]uint64, len(s.chains))
	for chainID, chain := range s.chains {
		chainIDs = append(chainIDs, chainID)
		startGens[chainID] = chain.Generation()
	}
	slices.SortFunc(chainIDs, func(a, b eth.ChainID) int { return a.Cmp(b) })

	var (
		minCurrentL1          eth.BlockID
		minSafeTimestamp      uint64
		minLocalSafeTimestamp uint64
		minFinalizedTimestamp uint64
		safeInitialized       bool
		localSafeInitialized  bool
		finalizedInitialized  bool

		optimistic         = make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
		verifiedRequiredL1 eth.BlockID
		chainOutputs       = make([]eth.ChainIDAndOutput, 0, len(s.chains))
		notFound           bool
	)

	for _, chainID := range chainIDs {
		chain := s.chains[chainID]

		status, err := chain.SyncStatus(ctx)
		if err != nil {
			s.log.Warn("failed to get sync status", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("sync status for chain %v: %w", chainID, err)
		}
		if status == nil {
			status = &eth.SyncStatus{}
		}

		// CurrentL1 is the minimum L1 block every derivation pipeline AND every
		// registered verifier has processed.
		currentL1 := status.CurrentL1.ID()
		if minCurrentL1 == (eth.BlockID{}) || currentL1.Number < minCurrentL1.Number {
			minCurrentL1 = currentL1
		}
		for _, verifierL1 := range chain.VerifierCurrentL1s() {
			if minCurrentL1 == (eth.BlockID{}) || verifierL1.Number < minCurrentL1.Number {
				minCurrentL1 = verifierL1
			}
		}

		// Conservative MIN across chains for the aggregate L2 timestamps.
		if !localSafeInitialized || status.LocalSafeL2.Time < minLocalSafeTimestamp {
			minLocalSafeTimestamp = status.LocalSafeL2.Time
			localSafeInitialized = true
		}
		if !safeInitialized || status.SafeL2.Time < minSafeTimestamp {
			minSafeTimestamp = status.SafeL2.Time
			safeInitialized = true
		}
		if !finalizedInitialized || status.FinalizedL2.Time < minFinalizedTimestamp {
			minFinalizedTimestamp = status.FinalizedL2.Time
			finalizedInitialized = true
		}

		// Verified path. NotFound is benign: the chain has no fully-verified
		// block at this timestamp, so the response carries no Data.Super.
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		switch {
		case errors.Is(err, ethereum.NotFound):
			notFound = true
		case err != nil:
			s.log.Warn("failed to get verified block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("verified at timestamp %d for chain %v: %w", timestamp, chainID, err)
		default:
			outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
			if err != nil {
				s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
				return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("output root at L2 block %d for chain %v: %w", verifiedL2.Number, chainID, err)
			}
			// MAX across chains of the minimum-required L1 for verification.
			if verifiedL1.Number > verifiedRequiredL1.Number {
				verifiedRequiredL1 = verifiedL1
			}
			chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
		}

		// Optimistic path. NotFound on either lookup just elides this chain
		// from OptimisticAtTimestamp.
		optOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		switch {
		case errors.Is(err, ethereum.NotFound):
			// no optimistic data for this chain
		case err != nil:
			s.log.Warn("failed to get optimistic block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("optimistic output at timestamp %d for chain %v: %w", timestamp, chainID, err)
		default:
			_, optL1, err := chain.OptimisticAt(ctx, timestamp)
			switch {
			case errors.Is(err, ethereum.NotFound):
				// source L1 unavailable — same treatment as a missing optimistic output
			case err != nil:
				s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
				return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("optimistic L1 at timestamp %d for chain %v: %w", timestamp, chainID, err)
			default:
				optimistic[chainID] = eth.OutputWithRequiredL1{
					Output:     optOut,
					OutputRoot: eth.OutputRoot(optOut),
					RequiredL1: optL1,
				}
			}
		}
	}

	// Final consistency check. If any chain's generation counter changed
	// during the reads above, the data may straddle a state-mutating event
	// (VN restart, engine rewind, pipeline reset) and we must not return it.
	for _, chainID := range chainIDs {
		if endGen := s.chains[chainID].Generation(); endGen != startGens[chainID] {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("chain %v gen %d → %d: %w", chainID, startGens[chainID], endGen, ErrInconsistentSnapshot)
		}
	}

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 minCurrentL1,
		CurrentSafeTimestamp:      minSafeTimestamp,
		CurrentLocalSafeTimestamp: minLocalSafeTimestamp,
		CurrentFinalizedTimestamp: minFinalizedTimestamp,
		OptimisticAtTimestamp:     optimistic,
		ChainIDs:                  chainIDs,
	}
	if !notFound {
		superV1 := eth.NewSuperV1(timestamp, chainOutputs...)
		superRoot := eth.SuperRoot(superV1)
		response.Data = &eth.SuperRootResponseData{
			VerifiedRequiredL1: verifiedRequiredL1,
			Super:              superV1,
			SuperRoot:          superRoot,
		}
	}
	return response, nil
}
