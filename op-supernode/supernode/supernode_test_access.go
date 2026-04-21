package supernode

// This file collects Supernode methods that expose test-only access to the
// interop activity. They must not be called by production code paths. Keeping
// them in one file makes the test-only surface easy to audit alongside
// interop/interop_test_access.go.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

var errNoInteropActivity = errors.New("supernode: no interop activity")

// InteropActivity returns the single interop activity registered with the
// supernode, or nil if interop is not configured or has not started yet.
// Callers must not cache the returned pointer across RestartInteropActivity;
// that path swaps the underlying activity for a fresh instance.
func (s *Supernode) InteropActivity() *interop.Interop {
	s.activitiesMu.RLock()
	defer s.activitiesMu.RUnlock()
	for _, a := range s.activities {
		if ia, ok := a.(*interop.Interop); ok {
			return ia
		}
	}
	return nil
}

// verifierReplacer is the subset of simpleChainContainer we depend on in
// RestartInteropActivity to swap a verifier registration without touching
// the public ChainContainer interface.
type verifierReplacer interface {
	ReplaceVerifier(old, new activity.VerificationActivity) bool
}

// RestartInteropActivity stops the running interop activity (if any),
// optionally wipes its on-disk logs DB files, constructs a fresh instance
// from the originally-configured parameters, re-registers it with each chain
// container as a verifier, and starts it under the supernode's existing
// lifecycle context. The HTTP server, chain containers, and all other
// activities keep running. This is the core primitive for tests that want
// to exercise log backfill against a running, ready cluster without the
// cost and flakiness of restarting the entire supernode.
//
// Any test-only mutations on the old activity are discarded when it is
// Stopped.
func (s *Supernode) RestartInteropActivity(wipeLogsDBs bool) error {
	if s.lifecycleCtx == nil {
		return fmt.Errorf("supernode: RestartInteropActivity called before Start")
	}
	if s.interopActivationTs == nil {
		return fmt.Errorf("supernode: RestartInteropActivity called but interop was never configured")
	}
	// Validate the DataDir precondition up front so a misconfigured call fails
	// before we tear down the old activity and lose its in-memory state.
	if wipeLogsDBs && (s.cfg == nil || s.cfg.DataDir == "") {
		return fmt.Errorf("supernode: cannot wipe logs DBs without a configured DataDir")
	}

	old := s.InteropActivity()
	if old == nil {
		return errNoInteropActivity
	}

	// Stop the old activity: cancels its ctx, waits its loop to exit on its
	// own, then closes verifiedDB and all logs DBs. Safe to ignore errors as
	// Stop only surfaces close errors and we're about to wipe/reopen.
	_ = old.Stop(context.Background())

	if wipeLogsDBs {
		for chainID := range s.chains {
			chainDir := filepath.Join(s.cfg.DataDir, fmt.Sprintf("chain-%s", chainID))
			if err := os.RemoveAll(chainDir); err != nil {
				return fmt.Errorf("supernode: wipe chain dir %s: %w", chainDir, err)
			}
			s.log.Info("wiped interop chain data dir", "chain", chainID, "path", chainDir)
		}
	}

	newIA := interop.New(
		s.log.New("activity", "interop"),
		*s.interopActivationTs,
		s.interopMsgExpiryWindow,
		s.chains,
		s.cfg.DataDir,
		s.l1Client,
		s.cfg.InteropLogBackfillDepth,
	)
	if newIA == nil {
		return fmt.Errorf("supernode: failed to construct replacement interop activity")
	}

	// Replace in s.activities so Reset-callback fan-out and test-only accessors
	// find the new instance. Locked because onChainReset and InteropActivity
	// iterate this slice from concurrent goroutines (chain containers are still
	// running across the restart).
	replaced := false
	s.activitiesMu.Lock()
	for i, a := range s.activities {
		if a == old {
			s.activities[i] = newIA
			replaced = true
			break
		}
	}
	s.activitiesMu.Unlock()
	if !replaced {
		return fmt.Errorf("supernode: old interop activity not found in activities slice")
	}

	// Swap verifier registration on every chain container.
	for chainID, chain := range s.chains {
		r, ok := chain.(verifierReplacer)
		if !ok {
			return fmt.Errorf("supernode: chain container for %s does not support ReplaceVerifier", chainID)
		}
		if !r.ReplaceVerifier(old, newIA) {
			return fmt.Errorf("supernode: old interop activity not registered as verifier on chain %s", chainID)
		}
	}

	// Launch the replacement activity on the existing lifecycle context so
	// it shares the supernode's shutdown path. Wait-group participation mirrors
	// how activities are launched in Start().
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := newIA.Start(s.lifecycleCtx)
		switch err {
		case nil:
			s.log.Error("activity quit unexpectedly", "name", newIA.Name())
		case context.Canceled:
			s.log.Info("activity closing due to cancelled context", "name", newIA.Name())
		case context.DeadlineExceeded:
			s.log.Warn("activity quit due to deadline exceeded", "name", newIA.Name())
		default:
			s.log.Error("error running restarted interop activity", "name", newIA.Name(), "error", err)
		}
	}()

	s.log.Info("interop activity restarted", "wipedLogsDBs", wipeLogsDBs)
	return nil
}
