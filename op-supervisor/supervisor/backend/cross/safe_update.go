package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type CrossSafeDeps interface {
	reads.Acquirer

	CrossSafe(chainID eth.ChainID) (pair types.DerivedBlockSealPair, err error)

	SafeFrontierCheckDeps
	SafeStartDeps

	CandidateCrossSafe(chain eth.ChainID) (candidate types.DerivedBlockRefPair, err error)
	NextSource(chain eth.ChainID, source eth.BlockID) (after eth.BlockRef, err error)
	PreviousCrossDerived(chain eth.ChainID, derived eth.BlockID) (prevDerived types.BlockSeal, err error)

	OpenBlock(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)

	UpdateCrossSafe(chain eth.ChainID, l1View eth.BlockRef, lastCrossDerived eth.BlockRef) error

	// InvalidateLocalSafe is called when a local block cannot be upgraded to cross-safe, and has to be dropped.
	// This is called relative to what was determined based on the l1Scope.
	// It is called with the candidate, the block that will be invalidated.
	// The replacement of this candidate will effectively be "derived from"
	// the scope that the candidate block was invalidated at.
	InvalidateLocalSafe(chainID eth.ChainID, candidate types.DerivedBlockRefPair) error
}

func CrossSafeUpdate(logger log.Logger, chainID eth.ChainID, d CrossSafeDeps, linker depset.LinkChecker) error {
	h := d.AcquireHandle()
	defer h.Release()
	logger.Debug("Cross-safe update call")
	candidate, err := scopedCrossSafeUpdate(h, logger, chainID, d, linker)
	if err == nil {
		// if we made progress, and no errors, then there is no need to bump the L1 scope yet.
		return h.Err() // make sure the read-consistency is still translated into an error
	}
	if errors.Is(err, types.ErrAwaitReplacementBlock) {
		logger.Info("Awaiting replacement block", "err", err)
		return err
	}
	if errors.Is(err, types.ErrConflict) {
		logger.Warn("Found a conflicting local-safe block that cannot be promoted to cross-safe",
			"scope", candidate.Source, "invalidated", candidate, "err", err)
		return d.InvalidateLocalSafe(chainID, candidate)
	}
	if !errors.Is(err, types.ErrOutOfScope) {
		return fmt.Errorf("failed to determine cross-safe update scope of chain %s: %w", chainID, err)
	}
	// candidate scope is expected to be set if ErrOutOfScope is returned.
	if candidate.Source == (eth.BlockRef{}) {
		return fmt.Errorf("expected L1 scope to be defined with ErrOutOfScope: %w", err)
	}
	logger.Debug("Cross-safe updating ran out of L1 scope", "scope", candidate.Source, "err", err)
	h.DependOnSourceBlock(candidate.Source.Number + 1)
	// bump the L1 scope up, and repeat the prev L2 block, not the candidate
	newScope, err := d.NextSource(chainID, candidate.Source.ID())
	if err != nil {
		return fmt.Errorf("failed to identify new L1 scope to expand to after %s: %w", candidate.Source, err)
	}
	currentCrossSafe, err := d.CrossSafe(chainID)
	if err != nil {
		// TODO: if genesis isn't cross-safe by default, then we can't register something as cross-safe here
		return fmt.Errorf("failed to identify cross-safe scope to repeat: %w", err)
	}
	parent, err := d.PreviousCrossDerived(chainID, currentCrossSafe.Derived.ID())
	if err != nil {
		return fmt.Errorf("cannot find parent-block of cross-safe: %w", err)
	}

	var crossSafeRef eth.BlockRef
	// During non-genesis Interop activation, the parent may be zero, which is ok.
	if parent.ID() == (eth.BlockID{}) {
		crossSafeRef = currentCrossSafe.Derived.WithZeroParent()
	} else {
		crossSafeRef = currentCrossSafe.Derived.MustWithParent(parent.ID())
	}

	// If any of the reads were invalidated due to reorg,
	// don't attempt to proceed with an update, as the reasoning for the update may be wrong.
	if !h.IsValid() {
		logger.Warn("Cross-safe reads were inconsistent, aborting scope-bump", "aborted", newScope)
		return types.ErrInvalidatedRead
	}
	logger.Debug("Bumping cross-safe scope", "scope", newScope, "crossSafe", crossSafeRef)
	if err := d.UpdateCrossSafe(chainID, newScope, crossSafeRef); err != nil {
		return fmt.Errorf("failed to update cross-safe head with L1 scope increment to %s and repeat of L2 block %s: %w", candidate.Source, crossSafeRef, err)
	}
	return nil
}

// scopedCrossSafeUpdate runs through the cross-safe update checks.
// If no L2 cross-safe progress can be made without additional L1 input data,
// then a types.ErrOutOfScope error is returned,
// with the current scope that will need to be expanded for further progress.
func scopedCrossSafeUpdate(h reads.Handle, logger log.Logger, chainID eth.ChainID, d CrossSafeDeps, linker depset.LinkChecker) (update types.DerivedBlockRefPair, err error) {
	candidate, err := d.CandidateCrossSafe(chainID)
	if err != nil {
		return candidate, fmt.Errorf("failed to determine candidate block for cross-safe: %w", err)
	}
	h.DependOnSourceBlock(candidate.Source.Number)
	h.DependOnDerivedTime(candidate.Derived.Time)
	logger.Debug("Candidate cross-safe", "scope", candidate.Source, "candidate", candidate.Derived)

	hazards, err := CrossSafeHazards(d, linker, logger, chainID, candidate.Source.ID(), types.BlockSealFromRef(candidate.Derived))
	if err != nil {
		return candidate, fmt.Errorf("failed to determine dependencies of cross-safe candidate %s: %w", candidate.Derived, err)
	}
	if err := HazardSafeFrontierChecks(d, candidate.Source.ID(), hazards); err != nil {
		return candidate, fmt.Errorf("failed to verify block %s in cross-safe frontier: %w", candidate.Derived, err)
	}
	if err := HazardCycleChecks(d, candidate.Derived.Time, hazards); err != nil {
		return candidate, fmt.Errorf("failed to verify block %s in cross-safe check for cycle hazards: %w", candidate, err)
	}
	// If any of the reads were inconsistent, don't continue with updating.
	if !h.IsValid() {
		logger.Warn("Cross-safe updating reads were inconsistent, aborting update", "aborted", candidate)
		return
	}
	// promote the candidate block to cross-safe
	if err := d.UpdateCrossSafe(chainID, candidate.Source, candidate.Derived); err != nil {
		return candidate, fmt.Errorf("failed to update cross-safe head to %s, derived from scope %s: %w", candidate.Derived, candidate.Source, err)
	}
	return candidate, nil
}

type CrossSafeWorker struct {
	logger  log.Logger
	chainID eth.ChainID
	d       CrossSafeDeps
	linker  depset.LinkChecker
}

func (c *CrossSafeWorker) OnEvent(ev event.Event) bool {
	switch ev.(type) {
	case superevents.UpdateCrossSafeRequestEvent:
		if err := CrossSafeUpdate(c.logger, c.chainID, c.d, c.linker); err != nil {
			if errors.Is(err, types.ErrFuture) {
				c.logger.Debug("Worker awaits additional blocks", "err", err)
			} else {
				c.logger.Warn("Failed to process work", "err", err)
			}
		}
	default:
		return false
	}
	return true
}

var _ event.Deriver = (*CrossUnsafeWorker)(nil)

func NewCrossSafeWorker(logger log.Logger, chainID eth.ChainID, d CrossSafeDeps, linker depset.LinkChecker) *CrossSafeWorker {
	logger = logger.New("chain", chainID, "worker", "cross-safe")
	return &CrossSafeWorker{
		logger:  logger,
		chainID: chainID,
		d:       d,
		linker:  linker,
	}
}
