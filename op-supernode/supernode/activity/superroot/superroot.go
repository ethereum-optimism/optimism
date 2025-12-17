package superroot

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
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
type OutputWithSource struct {
	Output   *eth.OutputResponse
	SourceL1 eth.BlockID
}

// L2WithRequiredL1 is a verified L2 block and the minimum L1 block at which the verification is possible
type L2WithRequiredL1 struct {
	L2            eth.BlockID
	MinRequiredL1 eth.BlockID
}

// atTimestampResponse is the response superroot_atTimestamp
// it contains:
// - CurrentL1Derived: the current L1 block that each chain has derived up to (without any verification)
// - CurrentL1Verified: the current L1 block that each verifier has processed up to
// - VerifiedAtTimestamp: the L2 blocks which are fully verified at the given timestamp, and the minimum L1 block at which verification is possible
// - OptimisticAtTimestamp: the L2 blocks which would be applied if verification were assumed to be successful, and their L1 sources
// - SuperRoot: the superroot at the given timestamp using verified L2 blocks
type atTimestampResponse struct {
	CurrentL1Derived      map[eth.ChainID]eth.BlockID
	CurrentL1Verified     map[string]eth.BlockID
	VerifiedAtTimestamp   map[eth.ChainID]L2WithRequiredL1
	OptimisticAtTimestamp map[eth.ChainID]OutputWithSource
	MinCurrentL1          eth.BlockID
	MinVerifiedRequiredL1 eth.BlockID
	SuperRoot             eth.Bytes32
}

// AtTimestamp computes the super-root at the given timestamp, plus additional information about the current L1s, verified L2s, and optimistic L2s
func (api *superrootAPI) AtTimestamp(ctx context.Context, timestamp uint64) (atTimestampResponse, error) {
	return api.s.atTimestamp(ctx, timestamp)
}

func (s *Superroot) atTimestamp(ctx context.Context, timestamp uint64) (atTimestampResponse, error) {
	currentL1Derived := map[eth.ChainID]eth.BlockID{}
	// there are no Verification Activities yet, so there is no call to make to collect their CurrentL1
	// this will be replaced with a call to the Verification Activities when they are implemented
	currentL1Verified := map[string]eth.BlockID{}
	verified := map[eth.ChainID]L2WithRequiredL1{}
	optimistic := map[eth.ChainID]OutputWithSource{}
	minCurrentL1 := eth.BlockID{}
	minVerifiedRequiredL1 := eth.BlockID{}
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(s.chains))

	// get current l1s
	// this informs callers that the chains local views have considered at least up to this L1 block
	// but does not guarantee verifiers have processed this L1 block yet. This field is likely unhelpful, but I await feedback to confirm
	for chainID, chain := range s.chains {
		currentL1, err := chain.CurrentL1(ctx)
		if err != nil {
			s.log.Warn("failed to get current L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, err
		}
		currentL1Derived[chainID] = currentL1.ID()
		if currentL1.ID().Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1.ID()
		}
	}

	// collect verified and optimistic L2 and L1 blocks at the given timestamp
	for chainID, chain := range s.chains {
		// verifiedAt returns the L2 block which is fully verified at the given timestamp, and the minimum L1 block at which verification is possible
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get verified L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, fmt.Errorf("%w: %w", ethereum.NotFound, err)
		}
		verified[chainID] = L2WithRequiredL1{
			L2:            verifiedL2,
			MinRequiredL1: verifiedL1,
		}
		if verifiedL1.Number < minVerifiedRequiredL1.Number || minVerifiedRequiredL1 == (eth.BlockID{}) {
			minVerifiedRequiredL1 = verifiedL1
		}
		// Compute output root at or before timestamp using the verified L2 block number
		outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
		if err != nil {
			s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
			return atTimestampResponse{}, fmt.Errorf("%w: %w", ethereum.NotFound, err)
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
		// Optimistic output is the full output at the optimistic L2 block for the timestamp
		optimisticOut, err := chain.OptimisticOutputAtTimestamp(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, fmt.Errorf("%w: %w", ethereum.NotFound, err)
		}
		// Also include the source L1 for context
		_, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic source L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, fmt.Errorf("%w: %w", ethereum.NotFound, err)
		}
		optimistic[chainID] = OutputWithSource{
			Output:   optimisticOut,
			SourceL1: optimisticL1,
		}
	}

	// Build super root from collected outputs
	superV1 := eth.NewSuperV1(timestamp, chainOutputs...)
	superRoot := eth.SuperRoot(superV1)

	return atTimestampResponse{
		CurrentL1Derived:      currentL1Derived,
		CurrentL1Verified:     currentL1Verified,
		VerifiedAtTimestamp:   verified,
		OptimisticAtTimestamp: optimistic,
		MinCurrentL1:          minCurrentL1,
		MinVerifiedRequiredL1: minVerifiedRequiredL1,
		SuperRoot:             superRoot,
	}, nil
}
