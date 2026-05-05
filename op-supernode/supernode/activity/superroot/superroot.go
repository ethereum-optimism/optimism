package superroot

import (
	"context"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
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
	// Gather every per-chain field needed for the response in a single call per
	// chain. The handler is responsible only for cross-chain aggregation; per-chain
	// reads (sync status, verified, optimistic, output root, etc.) live behind
	// ChainContainer.GatherSuperRootData so consistency guarantees can be applied
	// at a single seam.
	perChain := make(map[eth.ChainID]cc.ChainSuperRootData, len(s.chains))
	chainIDs := make([]eth.ChainID, 0, len(s.chains))
	for chainID, chain := range s.chains {
		data, err := chain.GatherSuperRootData(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to gather super root data", "chain_id", chainID.String(), "err", err)
			return eth.SuperRootAtTimestampResponse{}, err
		}
		perChain[chainID] = data
		chainIDs = append(chainIDs, chainID)
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
		data := perChain[chainID]
		status := data.SyncStatus
		if status == nil {
			status = &eth.SyncStatus{}
		}

		// CurrentL1 is the minimum L1 block every derivation pipeline AND every
		// registered verifier has processed.
		currentL1 := status.CurrentL1.ID()
		if minCurrentL1 == (eth.BlockID{}) || currentL1.Number < minCurrentL1.Number {
			minCurrentL1 = currentL1
		}
		for _, verifierL1 := range data.VerifierCurrentL1s {
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

		if data.Verified == nil {
			notFound = true
		} else {
			// MAX across chains of the minimum-required L1 for verification.
			if data.Verified.L1.Number > verifiedRequiredL1.Number {
				verifiedRequiredL1 = data.Verified.L1
			}
			chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: data.Verified.Output})
		}

		if data.Optimistic != nil {
			optimistic[chainID] = eth.OutputWithRequiredL1{
				Output:     data.Optimistic.Output,
				OutputRoot: eth.OutputRoot(data.Optimistic.Output),
				RequiredL1: data.Optimistic.L1,
			}
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
