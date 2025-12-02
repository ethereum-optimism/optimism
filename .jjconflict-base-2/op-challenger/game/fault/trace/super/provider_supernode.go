package super

import (
	"context"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	types2 "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type SuperNodeRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
}

type SuperNodeTraceProvider struct {
	PreimagePrestateProvider
	logger             log.Logger
	rootProvider       SuperNodeRootProvider
	prestateTimestamp  uint64
	poststateTimestamp uint64
	l1Head             eth.BlockID
	gameDepth          types.Depth
}

func NewSuperNodeTraceProvider(logger log.Logger, prestateProvider PreimagePrestateProvider, rootProvider SuperNodeRootProvider, l1Head eth.BlockID, gameDepth types.Depth, prestateTimestamp, poststateTimestamp uint64) *SuperNodeTraceProvider {
	return &SuperNodeTraceProvider{
		logger:                   logger,
		PreimagePrestateProvider: prestateProvider,
		rootProvider:             rootProvider,
		prestateTimestamp:        prestateTimestamp,
		poststateTimestamp:       poststateTimestamp,
		l1Head:                   l1Head,
		gameDepth:                gameDepth,
	}
}

func (s *SuperNodeTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	preimage, err := s.GetPreimageBytes(ctx, pos)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(preimage), nil
}

func (s *SuperNodeTraceProvider) getPreimageBytesAtTimestampBoundary(ctx context.Context, timestamp uint64) ([]byte, error) {
	root, err := s.rootProvider.SuperRootAtTimestamp(ctx, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve super root at timestamp %v: %w", timestamp, err)
	}
	if root.CurrentL1.Number < s.l1Head.Number {
		// Node has not processed the game's L1 head so it is not safe to play until it syncs further.
		return nil, types2.ErrNotInSync
	}
	if root.Data == nil {
		// No block at this timestamp so it must be invalid
		return InvalidTransition, nil
	}
	if root.Data.VerifiedRequiredL1.Number > s.l1Head.Number {
		return InvalidTransition, nil
	}
	return root.Data.Super.Marshal(), nil
}

func (s *SuperNodeTraceProvider) GetPreimageBytes(ctx context.Context, pos types.Position) ([]byte, error) {
	// Find the timestamp and step at position
	timestamp, step, err := s.ComputeStep(pos)
	if err != nil {
		return nil, err
	}
	s.logger.Trace("Getting claim", "pos", pos.ToGIndex(), "timestamp", timestamp, "step", step)
	if step == 0 {
		return s.getPreimageBytesAtTimestampBoundary(ctx, timestamp)
	}
	// Fetch the super root at the next timestamp since we are part way through the transition to it
	prevRoot, err := s.rootProvider.SuperRootAtTimestamp(ctx, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve previous super root at timestamp %v: %w", timestamp, err)
	}
	if prevRoot.CurrentL1.Number < s.l1Head.Number {
		return nil, types2.ErrNotInSync
	}
	if prevRoot.Data == nil {
		// No block at this timestamp so it must be invalid
		return InvalidTransition, nil
	}
	if prevRoot.Data.VerifiedRequiredL1.Number > s.l1Head.Number {
		// The previous root was not safe at the game L1 head so we must have already transitioned to the invalid hash
		// prior to this step and it then repeats forever.
		return InvalidTransition, nil
	}
	nextTimestamp := timestamp + 1
	nextRoot, err := s.rootProvider.SuperRootAtTimestamp(ctx, nextTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve next super root at timestamp %v: %w", nextTimestamp, err)
	}
	if nextRoot.CurrentL1.Number < s.l1Head.Number {
		return nil, types2.ErrNotInSync
	}

	prevSuper := prevRoot.Data.Super
	expectedState := interopTypes.TransitionState{
		SuperRoot:       prevSuper.Marshal(),
		PendingProgress: make([]interopTypes.OptimisticBlock, 0, step),
		Step:            step,
	}

	// Should already be sorted but be defensive and sort it ourselves
	slices.SortFunc(nextRoot.ChainIDs, func(a, b eth.ChainID) int {
		return a.Cmp(b)
	})
	for i := uint64(0); i < min(step, uint64(len(nextRoot.ChainIDs))); i++ {
		chainID := nextRoot.ChainIDs[i]
		// Check if the chain's optimistic root was safe at the game's L1 head
		optimistic, ok := nextRoot.OptimisticAtTimestamp[chainID]
		if !ok {
			// No block at this timestamp for a chain that needs to be processed at this step, so return invalid
			return InvalidTransition, nil
		}
		if optimistic.RequiredL1.Number > s.l1Head.Number {
			// Not enough data on L1 to derive the optimistic block, move to invalid transition.
			return InvalidTransition, nil
		}

		expectedState.PendingProgress = append(expectedState.PendingProgress, interopTypes.OptimisticBlock{
			BlockHash:  optimistic.Output.BlockRef.Hash,
			OutputRoot: optimistic.Output.OutputRoot,
		})
	}
	return expectedState.Marshal(), nil
}

func (s *SuperNodeTraceProvider) ComputeStep(pos types.Position) (timestamp uint64, step uint64, err error) {
	bigIdx := pos.TraceIndex(s.gameDepth)
	if !bigIdx.IsUint64() {
		err = fmt.Errorf("%w: %v", ErrIndexTooBig, bigIdx)
		return
	}

	traceIdx := bigIdx.Uint64() + 1
	timestampIncrements := traceIdx / StepsPerTimestamp
	timestamp = s.prestateTimestamp + timestampIncrements
	if timestamp >= s.poststateTimestamp { // Apply trace extension once the claimed timestamp is reached
		timestamp = s.poststateTimestamp
		step = 0
	} else {
		step = traceIdx % StepsPerTimestamp
	}
	return
}

func (s *SuperNodeTraceProvider) GetStepData(_ context.Context, _ types.Position) (prestate []byte, proofData []byte, preimageData *types.PreimageOracleData, err error) {
	return nil, nil, nil, ErrGetStepData
}

func (s *SuperNodeTraceProvider) GetL2BlockNumberChallenge(_ context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	// Never need to challenge L2 block number for super root games.
	return nil, types.ErrL2BlockNumberValid
}

var _ types.TraceProvider = (*SuperNodeTraceProvider)(nil)
