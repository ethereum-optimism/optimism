package interop

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-program/client/boot"
	"github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/cross"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var ErrInvalidBlockReplacement = errors.New("invalid block replacement error")

// ReceiptsToExecutingMessages returns the executing messages in the receipts indexed by their position in the log.
func ReceiptsToExecutingMessages(depset depset.ChainIndexFromID, receipts ethtypes.Receipts) (map[uint32]*supervisortypes.ExecutingMessage, uint32, error) {
	execMsgs := make(map[uint32]*supervisortypes.ExecutingMessage)
	var curr uint32
	for _, rcpt := range receipts {
		for _, l := range rcpt.Logs {
			execMsg, err := processors.DecodeExecutingMessageLog(l, depset)
			if err != nil {
				return nil, 0, err
			}
			if execMsg != nil {
				execMsgs[curr] = execMsg
			}
			curr++
		}
	}
	return execMsgs, curr, nil
}

type consolidateState struct {
	*types.TransitionState
	replacedChains map[eth.ChainID]bool
}

func (s *consolidateState) isReplaced(chainID eth.ChainID) bool {
	return s.replacedChains[chainID]
}

func (s *consolidateState) setReplaced(transitionStateIndex int, chainID eth.ChainID, outputRoot eth.Bytes32, replacementBlockHash common.Hash) {
	s.PendingProgress[transitionStateIndex].OutputRoot = outputRoot
	s.PendingProgress[transitionStateIndex].BlockHash = replacementBlockHash
	s.replacedChains[chainID] = true
}

func RunConsolidation(
	logger log.Logger,
	bootInfo *boot.BootInfoInterop,
	l1PreimageOracle l1.Oracle,
	l2PreimageOracle l2.Oracle,
	transitionState *types.TransitionState,
	superRoot *eth.SuperV1,
	tasks taskExecutor,
) (eth.Bytes32, error) {
	consolidateState := consolidateState{
		TransitionState: &types.TransitionState{
			PendingProgress: make([]types.OptimisticBlock, len(transitionState.PendingProgress)),
			SuperRoot:       transitionState.SuperRoot,
			Step:            transitionState.Step,
		},
		replacedChains: make(map[eth.ChainID]bool),
	}
	// We will be updating the transition state as blocks are replaced, so make a copy
	copy(consolidateState.PendingProgress, transitionState.PendingProgress)
	// Use a reference to the transition state so the consolidate oracle has a recent view.
	// The TransitionStateByRoot method isn't expected to be used during consolidation,
	// but we pass the state for safety in case this changes in the future.
	consolidateOracle := NewConsolidateOracle(l2PreimageOracle, consolidateState.TransitionState)

	// Keep consolidating until there are no more invalid blocks to replace
loop:
	for {
		err := singleRoundConsolidation(logger, bootInfo, l1PreimageOracle, consolidateOracle, &consolidateState, superRoot, tasks)
		switch {
		case err == nil:
			break loop
		case errors.Is(err, ErrInvalidBlockReplacement):
			continue
		default:
			return eth.Bytes32{}, err
		}
	}

	var consolidatedChains []eth.ChainIDAndOutput
	for i, chain := range superRoot.Chains {
		consolidatedChains = append(consolidatedChains, eth.ChainIDAndOutput{
			ChainID: chain.ChainID,
			Output:  consolidateState.PendingProgress[i].OutputRoot,
		})
	}
	consolidatedSuper := &eth.SuperV1{
		Timestamp: superRoot.Timestamp + 1,
		Chains:    consolidatedChains,
	}
	return eth.SuperRoot(consolidatedSuper), nil
}

func singleRoundConsolidation(
	logger log.Logger,
	bootInfo *boot.BootInfoInterop,
	l1PreimageOracle l1.Oracle,
	l2PreimageOracle *ConsolidateOracle,
	consolidateState *consolidateState,
	superRoot *eth.SuperV1,
	tasks taskExecutor,
) error {
	// The depset is the same for all chains. So it suffices to use any chain ID
	depset, err := bootInfo.Configs.DependencySet(superRoot.Chains[0].ChainID)
	if err != nil {
		return fmt.Errorf("failed to get dependency set: %w", err)
	}
	deps, err := newConsolidateCheckDeps(depset, bootInfo, consolidateState.TransitionState, superRoot.Chains, l2PreimageOracle)
	if err != nil {
		return fmt.Errorf("failed to create consolidate check deps: %w", err)
	}
	invalidChains := make(map[eth.ChainID]*ethtypes.Block)

	for i, chain := range superRoot.Chains {
		// Do not check chains that have been replaced with a deposits-only block.
		// They are already cross-safe because deposits-only blocks cannot contain executing messages.
		if consolidateState.isReplaced(chain.ChainID) {
			continue
		}

		agreedOutput := l2PreimageOracle.OutputByRoot(common.Hash(chain.Output), chain.ChainID)
		agreedOutputV0, ok := agreedOutput.(*eth.OutputV0)
		if !ok {
			return fmt.Errorf("unsupported L2 output version: %d", agreedOutput.Version())
		}
		agreedBlockHash := common.Hash(agreedOutputV0.BlockHash)

		progress := consolidateState.PendingProgress[i]
		// It's possible that the optimistic block is not canonical.
		// So we use the blockDataByHash hint to trigger a block rebuild to ensure that the block data, including receipts, are available.
		_ = l2PreimageOracle.BlockDataByHash(agreedBlockHash, progress.BlockHash, chain.ChainID)

		optimisticBlock, _ := l2PreimageOracle.ReceiptsByBlockHash(progress.BlockHash, chain.ChainID)

		candidate := supervisortypes.BlockSeal{
			Hash:      progress.BlockHash,
			Number:    optimisticBlock.NumberU64(),
			Timestamp: optimisticBlock.Time(),
		}
		if err := checkHazards(logger, deps, candidate, chain.ChainID); err != nil {
			if !isInvalidMessageError(err) {
				return err
			}
			invalidChains[chain.ChainID] = optimisticBlock
		}
	}

	if len(invalidChains) == 0 {
		return nil
	}

	for i, chain := range superRoot.Chains {
		if optimisticBlock, ok := invalidChains[chain.ChainID]; ok {
			chainAgreedPrestate := superRoot.Chains[i]
			replacementBlockHash, outputRoot, err := buildDepositOnlyBlock(
				logger,
				bootInfo,
				l1PreimageOracle,
				l2PreimageOracle,
				chainAgreedPrestate,
				tasks,
				optimisticBlock,
				// Update the preimage oracle database with the replaced block data
				l2PreimageOracle.KeyValueStore(),
			)
			if err != nil {
				return err
			}
			logger.Info(
				"Replaced block",
				"chain", chain.ChainID,
				"replacedBlock", eth.ToBlockID(optimisticBlock),
				"replacementBlockHash", replacementBlockHash,
				"outputRoot", outputRoot,
				"replacedOutputRoot", superRoot.Chains[i].Output,
			)
			superRoot.Chains[i].Output = outputRoot
			consolidateState.setReplaced(i, chain.ChainID, outputRoot, replacementBlockHash)
		}
	}
	return ErrInvalidBlockReplacement
}

func isInvalidMessageError(err error) bool {
	// TODO(#14011): Create an error category for InvalidExecutingMessage errors in the cross package for easier maintenance.
	return errors.Is(err, supervisortypes.ErrConflict) ||
		errors.Is(err, cross.ErrExecMsgHasInvalidIndex) ||
		errors.Is(err, cross.ErrExecMsgUnknownChain) ||
		errors.Is(err, cross.ErrCycle) || errors.Is(err, supervisortypes.ErrUnknownChain)
}

type ConsolidateCheckDeps interface {
	cross.UnsafeFrontierCheckDeps
	cross.CycleCheckDeps
	cross.UnsafeStartDeps
}

func checkHazards(logger log.Logger, deps ConsolidateCheckDeps, candidate supervisortypes.BlockSeal, chainID eth.ChainID) error {
	hazards, err := cross.CrossUnsafeHazards(deps, logger, chainID, candidate)
	if err != nil {
		return err
	}
	if err := cross.HazardUnsafeFrontierChecks(deps, hazards); err != nil {
		return err
	}
	if err := cross.HazardCycleChecks(deps.DependencySet(), deps, candidate.Timestamp, hazards); err != nil {
		return err
	}
	return nil
}

type consolidateCheckDeps struct {
	oracle      l2.Oracle
	depset      depset.DependencySet
	canonBlocks map[eth.ChainID]*l2.FastCanonicalBlockHeaderOracle
}

func newConsolidateCheckDeps(
	depset depset.DependencySet,
	bootInfo *boot.BootInfoInterop,
	transitionState *types.TransitionState,
	chains []eth.ChainIDAndOutput,
	oracle l2.Oracle,
) (*consolidateCheckDeps, error) {
	// TODO(#14415): handle case where dep set changes in a given timestamp
	canonBlocks := make(map[eth.ChainID]*l2.FastCanonicalBlockHeaderOracle)
	for i, chain := range chains {
		progress := transitionState.PendingProgress[i]
		// This is the optimistic head. It's OK if it's replaced by a deposits-only block.
		// Because by then the replacement block won't be used for hazard checks.
		head := oracle.BlockByHash(progress.BlockHash, chain.ChainID)
		blockByHash := func(hash common.Hash) *ethtypes.Block {
			return oracle.BlockByHash(hash, chain.ChainID)
		}
		l2ChainConfig, err := bootInfo.Configs.ChainConfig(chain.ChainID)
		if err != nil {
			return nil, fmt.Errorf("no chain config available for chain ID %v: %w", chain.ChainID, err)
		}
		fallback := l2.NewCanonicalBlockHeaderOracle(head.Header(), blockByHash)
		canonBlocks[chain.ChainID] = l2.NewFastCanonicalBlockHeaderOracle(head.Header(), blockByHash, l2ChainConfig, oracle, rawdb.NewMemoryDatabase(), fallback)
	}

	return &consolidateCheckDeps{
		oracle:      oracle,
		depset:      depset,
		canonBlocks: canonBlocks,
	}, nil
}

func (d *consolidateCheckDeps) Contains(chain eth.ChainID, query supervisortypes.ContainsQuery) (includedIn supervisortypes.BlockSeal, err error) {
	// We can assume the oracle has the block the executing message is in
	block, err := d.CanonBlockByNumber(d.oracle, query.BlockNum, chain)
	if err != nil {
		return supervisortypes.BlockSeal{}, err
	}
	_, receipts := d.oracle.ReceiptsByBlockHash(block.Hash(), chain)
	var current uint32
	for _, receipt := range receipts {
		for i, log := range receipt.Logs {
			if current+uint32(i) == query.LogIdx {
				checksum := supervisortypes.ChecksumArgs{
					BlockNumber: query.BlockNum,
					LogIndex:    query.LogIdx,
					Timestamp:   query.Timestamp,
					ChainID:     chain,
					LogHash:     logToLogHash(log),
				}.Checksum()
				if checksum != query.Checksum {
					return supervisortypes.BlockSeal{}, fmt.Errorf("checksum mismatch: %s != %s: %w", checksum, query.Checksum, supervisortypes.ErrConflict)
				} else if block.Time() != query.Timestamp {
					return supervisortypes.BlockSeal{}, fmt.Errorf("block timestamp mismatch: %d != %d: %w", block.Time(), query.Timestamp, supervisortypes.ErrConflict)
				} else {
					return supervisortypes.BlockSeal{
						Hash:      block.Hash(),
						Number:    block.NumberU64(),
						Timestamp: block.Time(),
					}, nil
				}
			}
		}
		current += uint32(len(receipt.Logs))
	}
	return supervisortypes.BlockSeal{}, fmt.Errorf("log not found")
}

func logToLogHash(l *ethtypes.Log) common.Hash {
	payloadHash := crypto.Keccak256Hash(supervisortypes.LogToMessagePayload(l))
	return supervisortypes.PayloadHashToLogHash(payloadHash, l.Address)
}

func (d *consolidateCheckDeps) IsCrossUnsafe(chainID eth.ChainID, block eth.BlockID) error {
	// Assumed to be cross-unsafe. And hazard checks will catch any future blocks prior to calling this
	return nil
}

func (d *consolidateCheckDeps) IsLocalUnsafe(chainID eth.ChainID, block eth.BlockID) error {
	// Always assumed to be local-unsafe
	return nil
}

func (d *consolidateCheckDeps) FindBlockID(chainID eth.ChainID, num uint64) (blockID eth.BlockID, err error) {
	block, err := d.CanonBlockByNumber(d.oracle, num, chainID)
	if err != nil {
		return eth.BlockID{}, err
	}
	return eth.BlockID{
		Hash:   block.Hash(),
		Number: block.NumberU64(),
	}, nil
}

func (d *consolidateCheckDeps) OpenBlock(
	chainID eth.ChainID,
	blockNum uint64,
) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*supervisortypes.ExecutingMessage, err error) {
	block, err := d.CanonBlockByNumber(d.oracle, blockNum, chainID)
	if err != nil {
		return eth.BlockRef{}, 0, nil, err
	}
	ref = eth.BlockRef{
		Hash:   block.Hash(),
		Number: block.NumberU64(),
	}
	_, receipts := d.oracle.ReceiptsByBlockHash(block.Hash(), chainID)
	execMsgs, logCount, err = ReceiptsToExecutingMessages(d.depset, receipts)
	if err != nil {
		return eth.BlockRef{}, 0, nil, err
	}
	return ref, logCount, execMsgs, nil
}

func (d *consolidateCheckDeps) DependencySet() depset.DependencySet {
	return d.depset
}

func (d *consolidateCheckDeps) CanonBlockByNumber(oracle l2.Oracle, blockNum uint64, chainID eth.ChainID) (*ethtypes.Block, error) {
	head := d.canonBlocks[chainID].GetHeaderByNumber(blockNum)
	if head == nil {
		return nil, fmt.Errorf("head not found for chain %v", chainID)
	}
	return d.oracle.BlockByHash(head.Hash(), chainID), nil
}

var _ ConsolidateCheckDeps = (*consolidateCheckDeps)(nil)

func buildDepositOnlyBlock(
	logger log.Logger,
	bootInfo *boot.BootInfoInterop,
	l1PreimageOracle l1.Oracle,
	l2PreimageOracle l2.Oracle,
	chainAgreedPrestate eth.ChainIDAndOutput,
	tasks taskExecutor,
	optimisticBlock *ethtypes.Block,
	db l2.KeyValueStore,
) (common.Hash, eth.Bytes32, error) {
	rollupCfg, err := bootInfo.Configs.RollupConfig(chainAgreedPrestate.ChainID)
	if err != nil {
		return common.Hash{}, eth.Bytes32{}, fmt.Errorf("no rollup config available for chain ID %v: %w", chainAgreedPrestate.ChainID, err)
	}
	l2ChainConfig, err := bootInfo.Configs.ChainConfig(chainAgreedPrestate.ChainID)
	if err != nil {
		return common.Hash{}, eth.Bytes32{}, fmt.Errorf("no chain config available for chain ID %v: %w", chainAgreedPrestate.ChainID, err)
	}
	blockHash, outputRoot, err := tasks.BuildDepositOnlyBlock(
		logger,
		rollupCfg,
		l2ChainConfig,
		bootInfo.L1Head,
		chainAgreedPrestate.Output,
		l1PreimageOracle,
		l2PreimageOracle,
		optimisticBlock,
		db,
	)
	if err != nil {
		return common.Hash{}, eth.Bytes32{}, err
	}
	return blockHash, outputRoot, nil
}
