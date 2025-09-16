package runner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func createGameInputs(ctx context.Context, log log.Logger, rollupClient *sources.RollupClient, supervisorClient *sources.SupervisorClient, typeName string, traceType types.TraceType) (utils.LocalGameInputs, error) {
	switch traceType {
	case types.TraceTypeSuperCannon, types.TraceTypeSuperPermissioned, types.TraceTypeSuperAsteriscKona:
		if supervisorClient == nil {
			return utils.LocalGameInputs{}, fmt.Errorf("trace type %s requires supervisor rpc to be set", traceType)
		}
		return createGameInputsInterop(ctx, log, supervisorClient, typeName)
	default:
		if rollupClient == nil {
			return utils.LocalGameInputs{}, fmt.Errorf("trace type %s requires rollup rpc to be set", traceType)
		}
		return createGameInputsSingle(ctx, log, rollupClient, typeName)
	}
}

func createGameInputsSingle(ctx context.Context, log log.Logger, client *sources.RollupClient, typeName string) (utils.LocalGameInputs, error) {
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get rollup sync status: %w", err)
	}
	log.Info("Got sync status", "status", status, "type", typeName)

	l1Head := status.FinalizedL1
	if status.FinalizedL1.Number > status.CurrentL1.Number {
		// Restrict the L1 head to a block that has actually been processed by op-node.
		// This only matters if op-node is behind and hasn't processed all finalized L1 blocks yet.
		l1Head = status.CurrentL1
		log.Info("Node has not completed syncing finalized L1 block, using CurrentL1 instead", "type", typeName)
	} else if status.FinalizedL1.Number == 0 {
		// The node is resetting its pipeline and has set FinalizedL1 to 0, use the current L1 instead as it is the best
		// hope of getting a non-zero L1 block
		l1Head = status.CurrentL1
		log.Warn("Node has zero finalized L1 block, using CurrentL1 instead", "type", typeName)
	}
	log.Info("Using L1 head", "head", l1Head, "type", typeName)
	if l1Head.Number == 0 {
		return utils.LocalGameInputs{}, errors.New("l1 head is 0")
	}
	blockNumber, err := findL2BlockNumberToDispute(ctx, log, client, l1Head.Number)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to find l2 block number to dispute: %w", err)
	}
	claimOutput, err := client.OutputAtBlock(ctx, blockNumber)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim output: %w", err)
	}
	parentOutput, err := client.OutputAtBlock(ctx, blockNumber-1)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim output: %w", err)
	}
	localInputs := utils.LocalGameInputs{
		L1Head:           l1Head.Hash,
		L2Head:           parentOutput.BlockRef.Hash,
		L2OutputRoot:     common.Hash(parentOutput.OutputRoot),
		L2Claim:          common.Hash(claimOutput.OutputRoot),
		L2SequenceNumber: new(big.Int).SetUint64(blockNumber),
	}
	return localInputs, nil
}

func createGameInputsInterop(ctx context.Context, log log.Logger, client *sources.SupervisorClient, typeName string) (utils.LocalGameInputs, error) {
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get supervisor sync status: %w", err)
	}
	log.Info("Got sync status", "status", status, "type", typeName)

	claimTimestamp := status.FinalizedTimestamp
	agreedTimestamp := claimTimestamp - 1
	if claimTimestamp == 0 {
		return utils.LocalGameInputs{}, errors.New("finalized timestamp is 0")
	}
	l1Head := status.MinSyncedL1
	log.Info("Using L1 head", "head", l1Head, "type", typeName)
	if l1Head.Number == 0 {
		return utils.LocalGameInputs{}, errors.New("l1 head is 0")
	}

	prestateProvider := super.NewSuperRootPrestateProvider(client, agreedTimestamp)
	gameDepth := types.Depth(30)
	provider := super.NewSuperTraceProvider(log, nil, prestateProvider, client, l1Head.ID(), gameDepth, agreedTimestamp, claimTimestamp+10)
	var agreedPrestate []byte
	var claim common.Hash
	switch rand.IntN(3) {
	case 0: // Derive block on first chain
		log.Info("Running first chain")
		prestate, err := prestateProvider.AbsolutePreState(ctx)
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get pre-state commitment: %w", err)
		}
		agreedPrestate = prestate.Marshal()
		claim, err = provider.Get(ctx, types.NewPosition(gameDepth, big.NewInt(0)))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim: %w", err)
		}
	case 1: // Derive block on second chain
		log.Info("Deriving second chain")
		agreedPrestate, err = provider.GetPreimageBytes(ctx, types.NewPosition(gameDepth, big.NewInt(0)))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get agreed prestate at position 0: %w", err)
		}
		claim, err = provider.Get(ctx, types.NewPosition(gameDepth, big.NewInt(1)))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim: %w", err)
		}
	case 2: // Consolidate
		log.Info("Running consolidate step")
		step := int64(super.StepsPerTimestamp - 1)
		agreedPrestate, err = provider.GetPreimageBytes(ctx, types.NewPosition(gameDepth, big.NewInt(step-1)))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get agreed prestate at position 0: %w", err)
		}
		claim, err = provider.Get(ctx, types.NewPosition(gameDepth, big.NewInt(step)))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to get claim: %w", err)
		}
	}
	localInputs := utils.LocalGameInputs{
		L1Head:           l1Head.Hash,
		AgreedPreState:   agreedPrestate,
		L2Claim:          claim,
		L2SequenceNumber: new(big.Int).SetUint64(claimTimestamp + 10), // Anything beyond the claim
	}
	return localInputs, nil
}

// findL2BlockNumberToDispute finds a safe l2 block number at different positions in a span batch
func findL2BlockNumberToDispute(ctx context.Context, log log.Logger, client *sources.RollupClient, l1HeadNum uint64) (uint64, error) {
	safeHead, err := client.SafeHeadAtL1Block(ctx, l1HeadNum)
	if err != nil {
		return 0, fmt.Errorf("failed to find safe head from l1 head %v: %w", l1HeadNum, err)
	}
	maxL2BlockNum := safeHead.SafeHead.Number

	// Find a prior span batch boundary
	// Limits how far back we search to 10 * 32 blocks
	const skipSize = uint64(32)
	for i := 0; i < 10; i++ {
		if l1HeadNum < skipSize {
			// Too close to genesis, give up and just use the original block
			log.Info("Failed to find prior batch.")
			return maxL2BlockNum, nil
		}
		l1HeadNum -= skipSize
		prevSafeHead, err := client.SafeHeadAtL1Block(ctx, l1HeadNum)
		if err != nil {
			return 0, fmt.Errorf("failed to get prior safe head at L1 block %v: %w", l1HeadNum, err)
		}
		if prevSafeHead.SafeHead.Number < maxL2BlockNum {
			switch rand.IntN(3) {
			case 0: // First block of span batch after prevSafeHead
				return prevSafeHead.SafeHead.Number + 1, nil
			case 1: // Last block of span batch ending at prevSafeHead
				return prevSafeHead.SafeHead.Number, nil
			case 2: // Random block, probably but not guaranteed to be in the middle of a span batch
				firstBlockInSpanBatch := prevSafeHead.SafeHead.Number + 1
				if maxL2BlockNum <= firstBlockInSpanBatch {
					// There is only one block in the next batch so we just have to use it
					return maxL2BlockNum, nil
				}
				offset := rand.IntN(int(maxL2BlockNum - firstBlockInSpanBatch))
				return firstBlockInSpanBatch + uint64(offset), nil
			}
		}
	}
	log.Warn("Failed to find prior batch", "l2BlockNum", maxL2BlockNum, "earliestCheckL1Block", l1HeadNum)
	return maxL2BlockNum, nil
}
