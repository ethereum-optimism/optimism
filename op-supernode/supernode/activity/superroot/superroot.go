package superroot

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
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

func (s *Superroot) RPCNamespace() string    { return "superroot" }
func (s *Superroot) RPCService() interface{} { return &superrootAPI{s: s} }

type superrootAPI struct{ s *Superroot }

// OutputWithSource is the full Output and its source L1 block
type OutputWithRequiredL1 struct {
	Output     *eth.OutputResponse `json:"output"`
	RequiredL1 eth.BlockID         `json:"required_l1"`
}

// L2WithRequiredL1 is a verified L2 block and the minimum L1 block at which the verification is possible
type L2WithRequiredL1 struct {
	L2         eth.BlockID `json:"l2"`
	RequiredL1 eth.BlockID `json:"required_l1"`
}

type SuperRootResponseData struct {
	// UnverifiedAtTimestamp is the L2 block that would be applied if verification were assumed to be successful,
	// and the minimum L1 block required to derive them.
	UnverifiedAtTimestamp map[eth.ChainID]OutputWithRequiredL1 `json:"unverified_at_timestamp"`

	// VerifiedRequiredL1 is the minimum L1 block including the required data to fully verify all blocks at this timestamp
	VerifiedRequiredL1 eth.BlockID `json:"verified_required_l1"`

	// Super is the unhashed data for the superroot at the given timestamp after all verification is applied.
	Super eth.Super `json:"super"`

	// SuperRoot is the superroot at the given timestamp after all verification is applied.
	SuperRoot eth.Bytes32 `json:"super_root"`
}

// AtTimestampResponse is the response superroot_atTimestamp
type AtTimestampResponse struct {
	// CurrentL1Derived is a map from chain ID to the highest L1 block that has been fully derived for that chain. It may not have been fully validated.
	CurrentL1Derived map[eth.ChainID]eth.BlockID `json:"current_l1_derived"`

	// CurrentL1 is the highest L1 block that has been fully derived and verified by all chains.
	CurrentL1 eth.BlockID `json:"current_l1"`

	// Data provides information about the super root at the requested timestamp if present. If block data at the
	// requested timestamp is not present, the data will be nil.
	Data []*SuperRootResponseData
}

// AtTimestamp computes the super-root at the given timestamp, plus additional information about the current L1s, verified L2s, and optimistic L2s
func (api *superrootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (AtTimestampResponse, error) {
	return api.s.atTimestamps(ctx, timestamp)
}

// AtTimestamp computes the super-root at the given timestamp, plus additional information about the current L1s, verified L2s, and optimistic L2s
func (api *superrootAPI) AtTimestamps(ctx context.Context, timestamps []hexutil.Uint64) (AtTimestampResponse, error) {
	return api.s.atTimestamps(ctx, timestamps...)
}

func (s *Superroot) atTimestamps(ctx context.Context, timestamps ...hexutil.Uint64) (AtTimestampResponse, error) {
	currentL1Derived := map[eth.ChainID]eth.BlockID{}
	minCurrentL1 := eth.BlockID{}

	// get current l1s
	// this informs callers that the chains local views have considered at least up to this L1 block
	// but does not guarantee verifiers have processed this L1 block yet. This field is likely unhelpful, but I await feedback to confirm
	for chainID, chain := range s.chains {
		currentL1, err := chain.CurrentL1(ctx)
		if err != nil {
			s.log.Warn("failed to get current L1", "chain_id", chainID.String(), "err", err)
			return AtTimestampResponse{}, err
		}
		currentL1Derived[chainID] = currentL1.ID()
		if currentL1.ID().Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1.ID()
		}
	}

	data := make([]*SuperRootResponseData, len(timestamps))
	for i, timestamp := range timestamps {
		superRootData, err := s.dataAtTimestamp(ctx, uint64(timestamp))
		if errors.Is(err, engine_controller.ErrNotFound) {
			// Leave this entry in data as nil as no super root is available
			continue
		} else if err != nil {
			return AtTimestampResponse{}, fmt.Errorf("failed to compute superroot at timestamp %v: %w", timestamp, err)
		}
		data[i] = superRootData
	}
	return AtTimestampResponse{
		CurrentL1Derived: currentL1Derived,
		CurrentL1:        minCurrentL1,
		Data:             data,
	}, nil
}

func (s *Superroot) dataAtTimestamp(ctx context.Context, timestamp uint64) (*SuperRootResponseData, error) {
	verified := map[eth.ChainID]L2WithRequiredL1{}
	optimistic := map[eth.ChainID]OutputWithRequiredL1{}
	maxVerifiedRequiredL1 := eth.BlockID{}
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(s.chains))

	// collect verified and optimistic L2 and L1 blocks at the given timestamp
	for chainID, chain := range s.chains {
		// verifiedAt returns the L2 block which is fully verified at the given timestamp, and the minimum L1 block at which verification is possible
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		if errors.Is(err, engine_controller.ErrNotFound) {
			// We don't have a fully verified block at the specified timestamp so no super root can be produced.
			// Return only the current derived L1 block info.
			return nil, engine_controller.ErrNotFound
		} else if err != nil {
			s.log.Warn("failed to get verified L1", "chain_id", chainID.String(), "err", err)
			return nil, fmt.Errorf("failed to get verified L1 for chain ID %v: %w", chainID, err)
		}
		verified[chainID] = L2WithRequiredL1{
			L2:         verifiedL2,
			RequiredL1: verifiedL1,
		}
		if verifiedL1.Number > maxVerifiedRequiredL1.Number || maxVerifiedRequiredL1 == (eth.BlockID{}) {
			maxVerifiedRequiredL1 = verifiedL1
		}
		// Compute output root at or before timestamp using the verified L2 block number
		outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
		if err != nil {
			s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
			return nil, fmt.Errorf("failed to compute output root at L2 block %v for chain ID %v: %w", verifiedL2.Number, chainID, err)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
		// Optimistic output is the full output at the optimistic L2 block for the timestamp
		optimisticOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic L1", "chain_id", chainID.String(), "err", err)
			return nil, fmt.Errorf("failed to get optimistic L1 for chain ID %v: %w", chainID, err)
		}
		// Also include the source L1 for context
		_, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
			return nil, fmt.Errorf("failed to get optimistic L1 for chain ID %v: %w", chainID, err)
		}
		optimistic[chainID] = OutputWithRequiredL1{
			Output:     optimisticOut,
			RequiredL1: optimisticL1,
		}
	}

	// Build super root from collected outputs
	superV1 := eth.NewSuperV1(timestamp, chainOutputs...)
	superRoot := eth.SuperRoot(superV1)

	return &SuperRootResponseData{
		UnverifiedAtTimestamp: optimistic,
		VerifiedRequiredL1:    maxVerifiedRequiredL1,
		Super:                 superV1,
		SuperRoot:             superRoot,
	}, nil
}
