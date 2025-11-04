package superroot

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
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

// API Response Shape not final
type derivedPair struct {
	L2 eth.BlockID
	L1 eth.BlockID
}
type atTimestampResponse struct {
	CurrentL1s map[eth.ChainID]eth.BlockID
	Verified   map[eth.ChainID]derivedPair
	Optimistic map[eth.ChainID]derivedPair
	SuperRoot  eth.Bytes32
}

// AtL1 computes the super-root at the given L1 block number.
func (api *superrootAPI) AtTimestamp(ctx context.Context, timestamp uint64) (atTimestampResponse, error) {
	return api.s.atTimestamp(ctx, timestamp)
}

func (s *Superroot) atTimestamp(ctx context.Context, timestamp uint64) (atTimestampResponse, error) {
	currentL1s := map[eth.ChainID]eth.BlockID{}
	verified := map[eth.ChainID]derivedPair{}
	optimistic := map[eth.ChainID]derivedPair{}
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
		currentL1s[chainID] = currentL1.ID()
	}

	for chainID, chain := range s.chains {
		// verifiedAt returns the L2 block which is fully verified at the given timestamp, and the minimum L1 block at which verification is possible
		verifiedL2, verifiedL1, err := chain.VerifiedAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get verified L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, err
		}
		verified[chainID] = derivedPair{
			L2: verifiedL2,
			L1: verifiedL1,
		}
		// Compute output root at or before timestamp using the verified L2 block number
		outRoot, err := chain.OutputRootAtL2BlockNumber(ctx, verifiedL2.Number)
		if err != nil {
			s.log.Warn("failed to compute output root at L2 block", "chain_id", chainID.String(), "l2_number", verifiedL2.Number, "err", err)
			return atTimestampResponse{}, err
		}
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{ChainID: chainID, Output: outRoot})
		// optimisticAt returns the L2 block which would be applied if verification were assumed to be successful
		optimisticL2, optimisticL1, err := chain.OptimisticAt(ctx, timestamp)
		if err != nil {
			s.log.Warn("failed to get optimistic L1", "chain_id", chainID.String(), "err", err)
			return atTimestampResponse{}, err
		}
		optimistic[chainID] = derivedPair{
			L2: optimisticL2,
			L1: optimisticL1,
		}
	}

	// Build super root from collected outputs
	superV1 := eth.NewSuperV1(timestamp, chainOutputs...)
	superRoot := eth.SuperRoot(superV1)

	return atTimestampResponse{
		CurrentL1s: currentL1s,
		Verified:   verified,
		Optimistic: optimistic,
		SuperRoot:  superRoot,
	}, nil
}
