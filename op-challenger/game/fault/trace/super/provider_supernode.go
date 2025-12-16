package super

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/superroot"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type SuperNodeRootProvider interface {
	// TODO: If AtTimestampResponse is being reused it should be put in op-service
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (superroot.AtTimestampResponse, error)
}

type SuperNodeTraceProvider struct {
	PreimagePrestateProvider
	logger             log.Logger
	rollupCfgs         *RollupConfigs
	rootProvider       SuperNodeRootProvider
	prestateTimestamp  uint64
	poststateTimestamp uint64
	l1Head             eth.BlockID
	gameDepth          types.Depth
}

func NewSuperNodeTraceProvider(logger log.Logger, rollupCfgs *RollupConfigs, prestateProvider PreimagePrestateProvider, rootProvider SuperNodeRootProvider, l1Head eth.BlockID, gameDepth types.Depth, prestateTimestamp, poststateTimestamp uint64) *SuperNodeTraceProvider {
	return &SuperNodeTraceProvider{
		logger:                   logger,
		rollupCfgs:               rollupCfgs,
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
	// TODO: Ideally we could check here if the node is in sync enough using root.CurrentL1Verified
	// but we would need to get that value even if the response is not found
	// TODO: Also make sure the client when written actually returns ethereum.NotFound not just the string "not found"
	if errors.Is(err, ethereum.NotFound) {
		// No block at this timestamp so it must be invalid
		return InvalidTransition, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve super root at timestamp %v: %w", timestamp, err)
	}
	if root.MaxVerifiedRequiredL1.Number > s.l1Head.Number {
		return InvalidTransition, nil
	}
	return root.Super.Marshal(), nil
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
	// TODO: Ideally we could check here if the node is in sync enough using root.CurrentL1Verified (as above)
	if errors.Is(err, ethereum.NotFound) {
		// No block at this timestamp so it must be invalid
		return InvalidTransition, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve previous super root at timestamp %v: %w", timestamp, err)
	}
	if prevRoot.MaxVerifiedRequiredL1.Number > s.l1Head.Number {
		// The previous root was not safe at the game L1 head so we must have already transitioned to the invalid hash
		// prior to this step and it then repeats forever.
		return InvalidTransition, nil
	}
	nextTimestamp := timestamp + 1
	nextRoot, err := s.rootProvider.SuperRootAtTimestamp(ctx, nextTimestamp)
	// TODO: Ideally we could check here if the node is in sync enough using root.CurrentL1Verified (as above)
	// Note that if we do the in sync check on every call we could safely be load balanced and would just error if
	// we get load balanced to a node that is not in sync enough. Unclear how useful that is if we keep hitting an unhealthy node though.
	// May need to have a SuperRootAtTimestamps (plural) method to get prev and next in one call.
	if errors.Is(err, ethereum.NotFound) {
		// No block at this timestamp so it must be invalid
		return InvalidTransition, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve next super root at timestamp %v: %w", nextTimestamp, err)
	}

	prevSuper := prevRoot.Super
	expectedState := interopTypes.TransitionState{
		SuperRoot:       prevSuper.Marshal(),
		PendingProgress: make([]interopTypes.OptimisticBlock, 0, step),
		Step:            step,
	}
	nextSuperV1, ok := nextRoot.Super.(*eth.SuperV1)
	if !ok {
		return nil, fmt.Errorf("unsupported super root type %T", nextRoot.Super)
	}
	for i := uint64(0); i < min(step, uint64(len(nextSuperV1.Chains))); i++ {
		chainInfo := nextSuperV1.Chains[i]
		// Check if the chain's optimistic root was safe at the game's L1 head
		if verified, ok := nextRoot.VerifiedAtTimestamp[chainInfo.ChainID]; !ok {
			return nil, fmt.Errorf("no safe head known for chain %v at %v: %w", chainInfo.ChainID, nextTimestamp, err)
		} else if verified.MinRequiredL1.Number > s.l1Head.Number {
			return InvalidTransition, nil
		}

		rawOutput := nextRoot.OptimisticAtTimestamp[chainInfo.ChainID].Output
		expectedState.PendingProgress = append(expectedState.PendingProgress, interopTypes.OptimisticBlock{
			BlockHash:  rawOutput.BlockRef.Hash,
			OutputRoot: rawOutput.OutputRoot,
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
