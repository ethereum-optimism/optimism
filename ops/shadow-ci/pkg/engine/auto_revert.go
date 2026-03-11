package engine

import (
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Notifier sends alerts about revert decisions.
type Notifier interface {
	NotifyRevert(decision *RevertDecision) error
}

// AutoReverter monitors post-merge test failures and creates revert decisions.
type AutoReverter struct {
	store    events.Store
	flakeDB  *model.FlakeDB
	emitter  *events.Emitter
	config   AutoRevertConfig
	notifier Notifier // nil = no notifications
}

// AutoRevertConfig controls auto-revert behavior.
type AutoRevertConfig struct {
	Org             string
	Repo            string
	MinFailures     int  // minimum real failures to trigger revert (default: 1)
	SkipKnownFlakes bool // don't revert for known flakes (default: true)
	DryRun          bool
}

// DefaultAutoRevertConfig returns sensible defaults.
func DefaultAutoRevertConfig() AutoRevertConfig {
	return AutoRevertConfig{
		MinFailures:     1,
		SkipKnownFlakes: true,
		DryRun:          true,
	}
}

// RevertDecision describes whether to revert and why.
type RevertDecision struct {
	ShouldRevert  bool     `json:"should_revert"`
	Reason        string   `json:"reason"`
	CulpritCommit string   `json:"culprit_commit,omitempty"`
	CulpritPR     int      `json:"culprit_pr,omitempty"`
	FailedTests   []string `json:"failed_tests"`
	RevertPRURL   string   `json:"revert_pr_url,omitempty"`
}

// NewAutoReverter creates a new AutoReverter.
func NewAutoReverter(store events.Store, flakeDB *model.FlakeDB, emitter *events.Emitter, config AutoRevertConfig, notifier Notifier) *AutoReverter {
	return &AutoReverter{
		store:    store,
		flakeDB:  flakeDB,
		emitter:  emitter,
		config:   config,
		notifier: notifier,
	}
}

// Evaluate determines whether a set of test failures should trigger a revert.
func (ar *AutoReverter) Evaluate(failedTests []string, commit string, pr int) *RevertDecision {
	decision := &RevertDecision{
		CulpritCommit: commit,
		CulpritPR:     pr,
	}

	// Filter out known flakes.
	var realFailures []string
	for _, testKey := range failedTests {
		if ar.config.SkipKnownFlakes && ar.flakeDB != nil && ar.flakeDB.IsQuarantined(testKey) {
			continue
		}
		realFailures = append(realFailures, testKey)
	}

	decision.FailedTests = realFailures

	if len(realFailures) == 0 {
		decision.Reason = "all failures are known flakes"
		return decision
	}

	if len(realFailures) < ar.config.MinFailures {
		decision.Reason = "below minimum failure threshold"
		return decision
	}

	decision.ShouldRevert = true
	decision.Reason = "real test failures detected on develop"

	if ar.emitter != nil {
		if ar.config.DryRun {
			ar.emitter.Emit(model.EventAutoRevertSkipped, decision)
		} else {
			ar.emitter.Emit(model.EventAutoRevertTriggered, decision)
		}
	}

	// Send notification if configured.
	if ar.notifier != nil {
		if err := ar.notifier.NotifyRevert(decision); err != nil {
			if ar.emitter != nil {
				ar.emitter.Emit(model.EventAutoRevertSkipped, map[string]any{
					"reason": "notification failed",
					"error":  err.Error(),
				})
			}
		}
	}

	return decision
}
