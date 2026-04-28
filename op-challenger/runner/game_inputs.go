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
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// Simulate a dispute game created roughly 16 days ago on a proposal that was ~7 days old
// at creation time (just above the anchor state), then played out over the maximum ~16-day
// duration with clock extensions. The L1 head in the game inputs therefore sits ~16 days
// behind the current chain head, and the disputed L2 block is derived from an L1 block
// another ~7 days earlier than that. Conversions assume a 12-second L1 block time.
const (
	gameL1HeadAgeBlocks   = uint64(16 * 24 * 60 * 60 / 12) // 115200
	disputeL1OffsetBlocks = uint64(7 * 24 * 60 * 60 / 12)  // 50400
)

func createGameInputs(ctx context.Context, log log.Logger, rollupClient *sources.RollupClient, superNodeClient *sources.SuperNodeClient, l1Client *ethclient.Client, typeName string, gameType gameTypes.GameType, ageGameInputs bool) (utils.LocalGameInputs, error) {
	switch gameType {
	case gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType, gameTypes.SuperCannonKonaGameType:
		if superNodeClient == nil {
			return utils.LocalGameInputs{}, fmt.Errorf("game type %s requires supernode rpc to be set", gameType)
		}
		return createGameInputsInterop(ctx, log, superNodeClient, typeName)
	default:
		if rollupClient == nil {
			return utils.LocalGameInputs{}, fmt.Errorf("game type %s requires rollup rpc to be set", gameType)
		}
		return createGameInputsSingle(ctx, log, rollupClient, l1Client, typeName, ageGameInputs)
	}
}

func createGameInputsSingle(ctx context.Context, log log.Logger, client *sources.RollupClient, l1Client *ethclient.Client, typeName string, ageGameInputs bool) (utils.LocalGameInputs, error) {
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get rollup sync status: %w", err)
	}
	log.Info("Got sync status", "status", status, "type", typeName)

	refHead := status.FinalizedL1
	if status.FinalizedL1.Number > status.CurrentL1.Number {
		// Restrict the reference head to a block that has actually been processed by op-node.
		// This only matters if op-node is behind and hasn't processed all finalized L1 blocks yet.
		refHead = status.CurrentL1
		log.Info("Node has not completed syncing finalized L1 block, using CurrentL1 instead", "type", typeName)
	} else if status.FinalizedL1.Number == 0 {
		// The node is resetting its pipeline and has set FinalizedL1 to 0, use the current L1 instead as it is the best
		// hope of getting a non-zero L1 block
		refHead = status.CurrentL1
		log.Warn("Node has zero finalized L1 block, using CurrentL1 instead", "type", typeName)
	}
	if refHead.Number == 0 {
		return utils.LocalGameInputs{}, errors.New("l1 head is 0")
	}

	gameL1Num, disputeL1Num, err := selectGameAndDisputeL1Blocks(ctx, log, client, refHead.Number, typeName, ageGameInputs)
	if err != nil {
		return utils.LocalGameInputs{}, err
	}

	l1HeadHash := refHead.Hash
	if gameL1Num != refHead.Number {
		l1HeadHeader, err := l1Client.HeaderByNumber(ctx, new(big.Int).SetUint64(gameL1Num))
		if err != nil {
			return utils.LocalGameInputs{}, fmt.Errorf("failed to fetch l1 head at block %v: %w", gameL1Num, err)
		}
		l1HeadHash = l1HeadHeader.Hash()
	}
	log.Info("Using L1 head", "num", gameL1Num, "hash", l1HeadHash, "disputeL1", disputeL1Num, "type", typeName)

	blockNumber, err := findL2BlockNumberToDispute(ctx, log, client, disputeL1Num)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to find l2 block number to dispute: %w", err)
	}
	if blockNumber == 0 {
		// L2 genesis can't be disputed (no parent block to use as the agreed prestate).
		return utils.LocalGameInputs{}, errors.New("dispute l2 block is at or below genesis")
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
		L1Head:           l1HeadHash,
		L2Head:           parentOutput.BlockRef.Hash,
		L2OutputRoot:     common.Hash(parentOutput.OutputRoot),
		L2Claim:          common.Hash(claimOutput.OutputRoot),
		L2SequenceNumber: new(big.Int).SetUint64(blockNumber),
	}
	return localInputs, nil
}

// selectGameAndDisputeL1Blocks returns the L1 head used for the game and the L1 block whose
// safe head bounds the disputed L2 block. A candidate is only accepted if the L2 safe head at
// that block is non-zero, so we don't pick L1 blocks that predate the L2 chain's first batch.
func selectGameAndDisputeL1Blocks(ctx context.Context, log log.Logger, client *sources.RollupClient, refHeadNum uint64, typeName string, ageGameInputs bool) (uint64, uint64, error) {
	if !ageGameInputs {
		return refHeadNum, refHeadNum, nil
	}

	gameL1Num := refHeadNum
	if refHeadNum >= gameL1HeadAgeBlocks {
		candidate := refHeadNum - gameL1HeadAgeBlocks
		ok, err := hasNonZeroSafeHead(ctx, client, candidate)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to check safe head at game L1 block %v: %w", candidate, err)
		}
		if ok {
			gameL1Num = candidate
		} else {
			log.Info("Game L1 block has no L2 safe head, falling back to reference head", "gameL1", candidate, "refHead", refHeadNum, "type", typeName)
		}
	} else {
		log.Info("Reference head younger than game L1 age, falling back to reference head", "refHead", refHeadNum, "ageBlocks", gameL1HeadAgeBlocks, "type", typeName)
	}

	disputeL1Num := gameL1Num
	if gameL1Num >= disputeL1OffsetBlocks {
		candidate := gameL1Num - disputeL1OffsetBlocks
		ok, err := hasNonZeroSafeHead(ctx, client, candidate)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to check safe head at dispute L1 block %v: %w", candidate, err)
		}
		if ok {
			disputeL1Num = candidate
		} else {
			log.Info("Dispute L1 block has no L2 safe head, falling back to game L1 head", "disputeL1", candidate, "gameL1", gameL1Num, "type", typeName)
		}
	} else {
		log.Info("Game L1 block younger than dispute offset, falling back to game L1 head", "gameL1", gameL1Num, "offsetBlocks", disputeL1OffsetBlocks, "type", typeName)
	}

	return gameL1Num, disputeL1Num, nil
}

// hasNonZeroSafeHead reports whether op-node has a non-genesis L2 safe head at the given L1
// block. Returns false if the safedb has no entry for the block, or if the recorded safe head
// is L2 block 0 (the L1 block predates the L2 chain's first batch).
func hasNonZeroSafeHead(ctx context.Context, client *sources.RollupClient, l1Num uint64) (bool, error) {
	safeHead, err := client.SafeHeadAtL1Block(ctx, l1Num)
	if errors.Is(err, safedb.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return safeHead.SafeHead.Number > 0, nil
}

func createGameInputsInterop(ctx context.Context, log log.Logger, client *sources.SuperNodeClient, typeName string) (utils.LocalGameInputs, error) {
	status, err := client.SyncStatus(ctx)
	if err != nil {
		return utils.LocalGameInputs{}, fmt.Errorf("failed to get supernode sync status: %w", err)
	}
	log.Info("Got sync status", "status", status, "type", typeName)

	claimTimestamp := status.FinalizedTimestamp
	agreedTimestamp := claimTimestamp - 1
	if claimTimestamp == 0 {
		return utils.LocalGameInputs{}, errors.New("finalized timestamp is 0")
	}
	l1Head := status.CurrentL1
	log.Info("Using L1 head", "head", l1Head, "type", typeName)
	if l1Head.Number == 0 {
		return utils.LocalGameInputs{}, errors.New("l1 head is 0")
	}

	prestateProvider := super.NewSuperNodePrestateProvider(client, agreedTimestamp)
	gameDepth := types.Depth(30)
	provider := super.NewSuperNodeTraceProvider(log, prestateProvider, client, l1Head, gameDepth, agreedTimestamp, claimTimestamp+10)
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

// findL2BlockNumberToDispute finds a safe l2 block number at different positions in a span batch.
// disputeL1Num is the L1 block whose safe head is the upper bound on the L2 block we pick; the
// function then walks further back in L1 to find a span batch boundary so that the selection
// can land at random positions within (or either edge of) a span batch.
func findL2BlockNumberToDispute(ctx context.Context, log log.Logger, client *sources.RollupClient, disputeL1Num uint64) (uint64, error) {
	safeHead, err := client.SafeHeadAtL1Block(ctx, disputeL1Num)
	if err != nil {
		return 0, fmt.Errorf("failed to find safe head from l1 block %v: %w", disputeL1Num, err)
	}
	maxL2BlockNum := safeHead.SafeHead.Number

	// Find a prior span batch boundary
	// Limits how far back we search to 10 * 32 blocks
	const skipSize = uint64(32)
	l1Num := disputeL1Num
	for i := 0; i < 10; i++ {
		if l1Num < skipSize {
			// Too close to genesis, give up and just use the original block
			log.Info("Failed to find prior batch.")
			return maxL2BlockNum, nil
		}
		l1Num -= skipSize
		prevSafeHead, err := client.SafeHeadAtL1Block(ctx, l1Num)
		if errors.Is(err, safedb.ErrNotFound) {
			// Walked back past the earliest known safe head (e.g. L2 genesis). Give up
			// looking for an earlier boundary and just use what we have.
			log.Info("Walked past earliest known safe head, using current max L2 block", "l2BlockNum", maxL2BlockNum)
			return maxL2BlockNum, nil
		}
		if err != nil {
			return 0, fmt.Errorf("failed to get prior safe head at L1 block %v: %w", l1Num, err)
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
	log.Warn("Failed to find prior batch", "l2BlockNum", maxL2BlockNum, "earliestCheckL1Block", l1Num)
	return maxL2BlockNum, nil
}
