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

func (s *Superroot) ActivityName() string { return "superroot" }

// Reset is a no-op for superroot - it always queries chain containers directly
// and doesn't maintain any chain-specific cached state.
func (s *Superroot) Reset(chainID eth.ChainID, timestamp uint64) {
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
	var (
		optimistic            = make(map[eth.ChainID]eth.OutputWithRequiredL1, len(s.chains))
		minCurrentL1          eth.BlockID
		minVerifiedRequiredL1 eth.BlockID
		minSafeTimestamp      uint64
		minFinalizedTimestamp uint64
		safeInitialized       bool
		finalizedInitialized  bool
		chainOutputs          = make([]eth.ChainIDAndOutput, 0, len(s.chains))
	)
	// Get current l1s
	// this informs callers that the chains local views have considered at least up to this L1 block
	// TODO(#18651): Currently there are no verifiers to consider, but once there are, this needs to be updated to consider if
	// they have also processed the L1 data.
	for chainID, chain := range s.chains {
		status, err := chain.SyncStatus(ctx)
		if err != nil {
			s.log.Warn("failed to get sync status", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, err
		}
		if status == nil { // defensive
			status = &eth.SyncStatus{}
		}

		currentL1 := status.CurrentL1.ID()
		if currentL1.Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1
		}
		// Conservative aggregation across chains: take the minimum timestamps.
		// If any chain has a zero timestamp (not initialized), the aggregate is zero.
		if !safeInitialized {
			minSafeTimestamp = status.SafeL2.Time
			safeInitialized = true
		} else if minSafeTimestamp == 0 || status.SafeL2.Time == 0 {
			minSafeTimestamp = 0
		} else if status.SafeL2.Time < minSafeTimestamp {
			minSafeTimestamp = status.SafeL2.Time
		}

		if !finalizedInitialized {
			minFinalizedTimestamp = status.FinalizedL2.Time
			finalizedInitialized = true
		} else if minFinalizedTimestamp == 0 || status.FinalizedL2.Time == 0 {
			minFinalizedTimestamp = 0
		} else if status.FinalizedL2.Time < minFinalizedTimestamp {
			minFinalizedTimestamp = status.FinalizedL2.Time
		}
	}

	notFound := false
	chainIDs := make([]eth.ChainID, 0, len(s.chains))
	// collect verified and optimistic L2 and L1 blocks at the given timestamp
	for chainID, chain := range s.chains {
		chainIDs = append(chainIDs, chainID)
		// verifiedAt returns the L2 block which is fully verified at the given timestamp, and the minimum L1 block at which verification is possible
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		if errors.Is(err, ethereum.NotFound) {
			notFound = true
			continue // To allow other chains to populate unverified blocks
		} else if err != nil {
			s.log.Warn("failed to get verified block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get verified block: %w", err)
		}
		if verifiedL1.Number < minVerifiedRequiredL1.Number || minVerifiedRequiredL1 == (eth.BlockID{}) {
			minVerifiedRequiredL1 = verifiedL1
		}
		// Compute output root at or before timestamp using the verified L2 block number
		outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
		if err != nil {
			s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to compute output root at L2 block %d for chain ID %v: %w", verifiedL2.Number, chainID, err)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
		// Optimistic output is the full output at the optimistic L2 block for the timestamp
		optimisticOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic block", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get optimistic block at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		// Also include the source L1 for context
		_, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("failed to get optimistic source L1 at timestamp %v for chain ID %v: %w", timestamp, chainID, err)
		}
		optimistic[chainID] = eth.OutputWithRequiredL1{
			Output:     optimisticOut,
			RequiredL1: optimisticL1,
		}
	}

	slices.SortFunc(chainIDs, func(a, b eth.ChainID) int {
		return a.Cmp(b)
	})
	response := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 minCurrentL1,
		CurrentSafeTimestamp:      minSafeTimestamp,
		CurrentFinalizedTimestamp: minFinalizedTimestamp,
		OptimisticAtTimestamp:     optimistic,
		ChainIDs:                  chainIDs,
	}
	if !notFound {
		// Build super root from collected outputs
		superV1 := eth.NewSuperV1(timestamp, chainOutputs...)
		superRoot := eth.SuperRoot(superV1)
		response.Data = &eth.SuperRootResponseData{
			VerifiedRequiredL1: minVerifiedRequiredL1,
			Super:              superV1,
			SuperRoot:          superRoot,
		}
	}
	return response, nil
}
