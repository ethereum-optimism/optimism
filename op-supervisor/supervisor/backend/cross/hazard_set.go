package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	errHazardSetNilDeps = errors.New("hazard dependencies cannot be nil")
)

type HazardDeps interface {
	Contains(chain eth.ChainID, query types.ContainsQuery) (types.BlockSeal, error)
	IsCrossValidBlock(chainID eth.ChainID, block eth.BlockID) error
	OpenBlock(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)
}

// HazardSet tracks blocks that must be checked before a candidate can be promoted
type HazardSet struct {
	entries map[eth.ChainID]types.BlockSeal
}

// NewHazardSet creates a new HazardSet with the given dependencies and initial block
func NewHazardSet(deps HazardDeps, linker depset.LinkChecker, logger log.Logger, chainID eth.ChainID, block types.BlockSeal) (*HazardSet, error) {
	if deps == nil {
		return nil, errHazardSetNilDeps
	}
	h := &HazardSet{
		entries: make(map[eth.ChainID]types.BlockSeal),
	}
	logger.Debug("Building new HazardSet", "chainID", chainID, "block", block)
	if err := h.build(deps, linker, logger, chainID, block); err != nil {
		return nil, fmt.Errorf("failed to build hazard set: %w", err)
	}
	logger.Debug("Successfully built HazardSet", "chainID", chainID, "block", block)
	return h, nil
}

func NewHazardSetFromEntries(entries map[eth.ChainID]types.BlockSeal) *HazardSet {
	return &HazardSet{entries: entries}
}

// potentialHazard represents a block that needs to be processed for hazards
type potentialHazard struct {
	chainID eth.ChainID
	block   types.BlockSeal
}

// checkChainCanExecute verifies that a chain can execute messages at a given timestamp.
// If there are any executing messages, then the chain must be able to execute at the timestamp.
func (h *HazardSet) checkChainCanExecute(linker depset.LinkChecker, chainID eth.ChainID, block types.BlockSeal, execMsgs map[uint32]*types.ExecutingMessage) error {
	for i, msg := range execMsgs {
		if !linker.CanExecute(chainID, block.Timestamp, msg.ChainID, msg.Timestamp) {
			return fmt.Errorf("executing message %d in block %s (chain %s) may not execute %s: %w", i, block, chainID, msg, types.ErrConflict)
		}
	}
	return nil
}

// checkMessageWithOlderTimestamp handles messages from past blocks.
// It ensures non-cyclic ordering relative to other messages.
func (h *HazardSet) checkMessageWithOlderTimestamp(deps HazardDeps, initChainID eth.ChainID, includedIn types.BlockSeal, candidateTimestamp uint64) error {
	if err := deps.IsCrossValidBlock(initChainID, includedIn.ID()); err != nil {
		return fmt.Errorf("included in non-cross valid block %s: %w", includedIn, err)
	}
	// Time expiry was already checked before; the DB has to support the time range.
	return nil
}

// checkMessageWithCurrentTimestamp handles messages from the same time as the candidate block.
// We have to inspect ordering of individual log events to ensure non-cyclic cross-chain message ordering.
// And since we may have back-and-forth messaging, we cannot wait till the initiating side is cross-safe.
// Thus check that it was included in a local-safe block, and then proceed with transitive block checks,
// to ensure the local block we depend on is becoming cross-safe also.
// Also returns a boolean indicating if the message already exists in the hazard set.
func (h *HazardSet) checkMessageWithCurrentTimestamp(initChainID eth.ChainID, includedIn types.BlockSeal) (bool, error) {
	existing, ok := h.entries[initChainID]
	if ok {
		if existing.ID() != includedIn.ID() {
			return true, fmt.Errorf("found dependency on %s (chain %s), but already depend on %s: %w", includedIn, initChainID, existing, types.ErrConflict)
		}
	}
	return ok, nil
}

// build adds a block to the hazard set and recursively adds any blocks that it depends on.
// Warning for future: If we have sub-second distinct blocks (different block number),
// we need to increase precision on the above timestamp invariant.
// Otherwise a local block can depend on a future local block of the same chain,
// simply by pulling in a block of another chain,
// which then depends on a block of the original chain,
// all with the same timestamp, without message cycles.
func (h *HazardSet) build(deps HazardDeps, linker depset.LinkChecker, logger log.Logger, chainID eth.ChainID, block types.BlockSeal) error {
	stack := []potentialHazard{{chainID: chainID, block: block}}

	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		candidate := next.block
		destChainID := next.chainID
		logger.Debug("Processing block for hazards", "chainID", destChainID, "block", candidate)

		// Get the block and ensure it's allowed to execute messages.
		opened, _, execMsgs, err := deps.OpenBlock(destChainID, candidate.Number)
		if err != nil {
			return fmt.Errorf("failed to open block: %w", err)
		}
		if opened.ID() != candidate.ID() {
			return fmt.Errorf("unsafe L2 DB has %s, but candidate cross-safe was %s: %w", opened, candidate, types.ErrConflict)
		}
		// Performance & safety: check all executing messages can exist (chain ID linking, timestamp invariants) first.
		if err := h.checkChainCanExecute(linker, destChainID, candidate, execMsgs); err != nil {
			return err
		}
		// Now that we have established chains and timestamps are valid things to link, build the hazard set.
		for _, msg := range execMsgs {
			logger.Debug("Processing message", "chainID", destChainID, "block", candidate, "msg", msg)
			q := types.ContainsQuery{
				Timestamp: msg.Timestamp,
				BlockNum:  msg.BlockNum,
				LogIdx:    msg.LogIdx,
				Checksum:  msg.Checksum,
			}
			includedIn, err := deps.Contains(msg.ChainID, q)
			if err != nil {
				return fmt.Errorf("executing msg %s failed inclusion check: %w", msg, err)
			}

			if msg.Timestamp < candidate.Timestamp {
				if err := h.checkMessageWithOlderTimestamp(deps, msg.ChainID, includedIn, candidate.Timestamp); err != nil {
					return fmt.Errorf("executing msg %s failed old-timestamp check: %w", msg, err)
				}
			} else if msg.Timestamp == candidate.Timestamp {
				exists, err := h.checkMessageWithCurrentTimestamp(msg.ChainID, includedIn)
				if err != nil {
					return fmt.Errorf("executing msg %s failed same-timestamp check: %w", msg, err)
				}

				if !exists {
					logger.Debug("Adding block to the hazard set", "chainID", msg.ChainID, "block", includedIn)
					h.entries[msg.ChainID] = includedIn
					stack = append(stack, potentialHazard{
						chainID: msg.ChainID,
						block:   includedIn,
					})
				}
			} else {
				return fmt.Errorf("executing message %s in %s breaks timestamp invariant: %w", msg, candidate, types.ErrConflict)
			}
		}
	}
	return nil
}

func (h *HazardSet) Entries() map[eth.ChainID]types.BlockSeal {
	if h == nil {
		return nil
	}
	return h.entries
}
