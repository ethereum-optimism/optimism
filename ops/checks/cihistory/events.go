// Package cihistory closes the calibration loop for the checks engine.
//
// Given a history of PRs with their file diffs and which checks passed
// or failed, it writes two artifacts:
//
//  1. Observed-correlation edges (source file → check) carrying the
//     historical precision of that check given a change to that file.
//     Phase 1 (Resolve) consumes these edges like it consumes coverage
//     edges, so signal from "file X historically co-occurs with check
//     Y failing" flows into the selector with no code changes beyond
//     Phase 1.
//
//  2. policy/learned.yaml with per-kind failure-rate priors derived
//     from base rates over the window. Layered on top of the embedded
//     baseline via the policy loader.
//
// Ingestion is explicit: an operator (or a scheduled job) runs
// `checks ingest ci-history` with a source of events. For portability
// and determinism, the events source is a pluggable Fetcher; a default
// FileFetcher reads events from a JSON file so the whole pipeline can
// be driven without network access. A CircleCI/GitHub-backed Fetcher
// is a natural follow-up.
package cihistory

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Event is one merged PR's CI outcome.
type Event struct {
	PR        int        `json:"pr"`
	MergedAt  time.Time  `json:"merged_at"`
	Files     []string   `json:"files"`  // repo-relative paths changed in this PR
	Checks    []CheckRun `json:"checks"` // check runs observed on this PR
}

// CheckRun is one check's outcome on a specific PR.
type CheckRun struct {
	ID     string `json:"id"`     // must match a catalog CheckType.ID
	Failed bool   `json:"failed"`
}

// LoadEvents parses a JSON file containing []Event. Used by FileFetcher
// and by tests.
func LoadEvents(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading events: %w", err)
	}
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parsing events: %w", err)
	}
	return events, nil
}
