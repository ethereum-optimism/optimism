package superchain

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type BlockSafetyLabel int

const (
	BlockUnsafe BlockSafetyLabel = iota - 1
	BlockCrossUnsafe
	BlockSafe
	BlockFinalized
)

type blockDependency struct {
	chainId     *big.Int
	blockNumber uint64
}

type blockDependent struct {
	chainId  *big.Int
	blockRef eth.L2BlockRef
}

type BlockDependencies struct {
	log log.Logger

	chains map[string]*sources.L2Client

	// We only track the heads and not any parent blocks previously added. The l2
	// client source implements caching, simplfying memory management here.
	heads map[string]common.Hash

	// block -> unverified messages
	unverifiedExecutingMessages map[common.Hash][]Message

	// chain -> block -> dependencies
	// (1) Link between blocks with executing message to the blocks that should contain the initiating message
	// (2) The parent block is by default a dependency for a derived block
	dependencies map[string]map[common.Hash][]blockDependency

	// chain -> block number -> dependents
	// (1) Link between blocks with an initiating message that's been executed.
	// (2) Any derived block is by default a dependent for the parent.
	//
	// The block number is used here since the block containing the initiating
	// message may not have yet been observed when processing the executing message.
	dependents map[string]map[uint64][]blockDependent

	// Start with a global lock on the graph and avoid the optimization if it's not contentious
	mu sync.Mutex
}

func (deps *BlockDependencies) BlockSafety(chainId *big.Int, blockRef eth.L2BlockRef) (BlockSafetyLabel, error) {
	deps.mu.Lock()
	defer deps.mu.Unlock()

	if len(deps.unverifiedExecutingMessages[blockRef.Hash]) > 0 {
		return BlockUnsafe, nil
	}

	for _, blockDependency := range deps.dependencies[chainId.String()][blockRef.Hash] {
		// Since there are no unverified messages in this block, we can safely fetch the
		// right block using the block number as the initiating message in the remote
		// block was validated.
		//
		// We also know the remote block specified by number hasn't been reorg'd, otherwise
		// the invalidation would have cascaded to this block on a reset.
		chain := deps.chains[blockDependency.chainId.String()]
		block, err := chain.L2BlockRefByNumber(context.TODO(), blockDependency.blockNumber)
		if err != nil {
			return BlockUnsafe, err
		}

		dependencyBlockSafety, err := deps.BlockSafety(blockDependency.chainId, block)
		if err != nil {
			return BlockUnsafe, err
		}

		if dependencyBlockSafety == BlockUnsafe {
			return BlockUnsafe, nil
		}
	}

	return BlockCrossUnsafe, nil
}

func (deps *BlockDependencies) AddBlock(chainId *big.Int, blockRef eth.L2BlockRef) error {
	deps.mu.Lock()
	defer deps.mu.Unlock()

	deps.log.Debug("adding block", "chain_id", chainId, "hash", blockRef.Hash)

	chainIdStr := chainId.String()
	chain, ok := deps.chains[chainIdStr]
	if !ok {
		return fmt.Errorf("chain %d not present in configuration", chainId)
	}

	head := deps.heads[chainIdStr]
	if blockRef.ParentHash != head {
		return fmt.Errorf("block %s does not build on head %s", blockRef.Hash, head)
	}

	_, txs, err := chain.InfoAndTxsByHash(context.TODO(), blockRef.Hash)
	if err != nil {
		return fmt.Errorf("unable to query txs: %w", err)
	}

	// default edge with the parent block
	deps.dependents[chainIdStr][blockRef.Number-1] = append(deps.dependents[chainIdStr][blockRef.Number-1], blockDependent{chainId, blockRef})
	deps.dependencies[chainIdStr][blockRef.Hash] = append(deps.dependencies[chainIdStr][blockRef.Hash], blockDependency{chainId, blockRef.Number - 1})

	// add edges for present executing messages
	deps.heads[chainIdStr] = blockRef.Hash
	for _, tx := range txs {
		if IsInboxExecutingMessageTx(tx) {
			_, id, payload, err := ParseInboxExecuteMessageTxData(tx.Data())
			if err != nil {
				log.Warn("skipping inbox tx with bad tx data", "err", err)
				continue
			}

			// todo: de-dup edges
			deps.unverifiedExecutingMessages[blockRef.Hash] = append(deps.unverifiedExecutingMessages[blockRef.Hash], Message{id, payload})
			deps.dependents[id.ChainId.String()][id.BlockNumber.Uint64()] = append(deps.dependents[id.ChainId.String()][id.BlockNumber.Uint64()], blockDependent{id.ChainId, blockRef})
			deps.dependencies[chainIdStr][blockRef.Hash] = append(deps.dependencies[chainIdStr][blockRef.Hash], blockDependency{id.ChainId, id.BlockNumber.Uint64()})
		}
	}

	// attempt resolution for this block & any set dependents
	deps.resolveUnverifiedExecutingMessages(chainId, blockRef)
	for _, dependentBlock := range deps.dependents[chainIdStr][blockRef.Number] {
		deps.resolveUnverifiedExecutingMessages(dependentBlock.chainId, dependentBlock.blockRef)
	}

	return nil
}

func (deps *BlockDependencies) resolveUnverifiedExecutingMessages(chainId *big.Int, blockRef eth.L2BlockRef) {
	deps.log.Debug("resolving unverified messages", "chain_id", chainId, "hash", blockRef.Hash)

	unverifiedMessages := deps.unverifiedExecutingMessages[blockRef.Hash]
	remainingUnverifiedMessages := make([]Message, 0, len(unverifiedMessages))
	for _, _ = range unverifiedMessages {
	}

	deps.unverifiedExecutingMessages[blockRef.Hash] = remainingUnverifiedMessages
}

func (deps *BlockDependencies) handleInvalidation(chainId *big.Int, blockRef eth.L2BlockRef) {
	deps.log.Debug("block invalidation", "chain_id", chainId, "hash", blockRef.Hash)

	// new head is the parent
	chainIdStr := chainId.String()
	deps.heads[chainIdStr] = blockRef.ParentHash

	// first invalidate dependents (includes derived blocks)
	for _, dependentBlock := range deps.dependents[chainIdStr][blockRef.Number] {
		deps.handleInvalidation(dependentBlock.chainId, dependentBlock.blockRef)
	}

	// remove all edges from this block
	delete(deps.dependents[chainIdStr], blockRef.Number)
	delete(deps.dependencies[chainIdStr], blockRef.Hash)
}
