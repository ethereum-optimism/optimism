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

// ErrInconsistentSnapshot is returned when a chain's Generation() changed
// during atTimestamp's per-chain reads. Callers should treat it as a
// transient retryable signal.
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
	// Capture each chain's Generation; re-checked at the end to discard
	// data gathered across a state-mutating event.
	startGens := make(map[eth.ChainID]uint64, len(s.chains))
	for chainID, chain := range s.chains {
		startGens[chainID] = chain.Generation()
	}

	aggregate, err := syncstatus.Aggregate(ctx, s.log, s.chains)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}

	var (
		optimistic         = make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
		verifiedRequiredL1 eth.BlockID
		chainOutputs       = make([]eth.ChainIDAndOutput, 0, len(s.chains))
	)

	notFound := false
	// Collect verified L2 and L1 blocks at the given timestamp
	for chainID, chain := range s.chains {
		// verifiedAt returns the L2 block which is fully verified at the given timestamp, and the minimum L1 block at which verification is possible
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			notFound = true
			continue
		} else if err != nil {
			s.log.Warn("failed to get verified block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get verified block: %w", err)
		}
		// Verified data is available: track the L1 block that includes the data
		// for every chain — i.e. the MAX of per-chain minimum-required L1s.
		if verifiedL1.Number > verifiedRequiredL1.Number {
			verifiedRequiredL1 = verifiedL1
		}
		// Compute output root at or before timestamp using the verified L2 block number
		outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
		if err != nil {
			s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to compute output root at L2 block %d for chain ID %v: %w", verifiedL2.Number, chainID, err)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
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

	for chainID, chain := range s.chains {
		if endGen := chain.Generation(); endGen != startGens[chainID] {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("chain %v gen %d → %d: %w", chainID, startGens[chainID], endGen, ErrInconsistentSnapshot)
		}
	}

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 aggregate.CurrentL1,
		CurrentSafeTimestamp:      aggregate.SafeTimestamp,
		CurrentLocalSafeTimestamp: aggregate.LocalSafeTimestamp,
		CurrentFinalizedTimestamp: aggregate.FinalizedTimestamp,
		OptimisticAtTimestamp:     optimistic,
		ChainIDs:                  aggregate.ChainIDs,
	}
	if !notFound {
		// Build super root from collected outputs
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
