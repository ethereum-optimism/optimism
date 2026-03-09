package model

import (
	"encoding/json"
	"time"
)

// EventType identifies what kind of event occurred.
type EventType string

const (
	// Planning events.
	EventPlanCreated     EventType = "plan.created"
	EventTargetsComputed EventType = "targets.computed"

	// Execution events.
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventTestPassed   EventType = "test.passed"
	EventTestFailed   EventType = "test.failed"
	EventTestRetried  EventType = "test.retried"

	// Classification events.
	EventFlakeDetected EventType = "flake.detected"
	EventRealFailure   EventType = "failure.real"
	EventInfraFailure  EventType = "failure.infrastructure"

	// Comparison events.
	EventComparisonComplete EventType = "comparison.complete"
	EventFalseNegative      EventType = "false_negative.detected"
	EventMatchConfirmed     EventType = "match.confirmed"

	// Graph maintenance events.
	EventGraphGap          EventType = "graph.gap_detected"
	EventGraphUpdated      EventType = "graph.updated"
	EventConfidenceChanged EventType = "confidence.changed"

	// Decision events.
	EventPipelineDecision EventType = "pipeline.decision"

	// Report events.
	EventWeeklyReport EventType = "report.weekly"

	// Flake lifecycle events.
	EventFlakeStateChanged EventType = "flake.state_changed"
	EventFlakeQuarantined  EventType = "flake.quarantined"
	EventFlakeRestored     EventType = "flake.restored"

	// Placement events.
	EventPlacementChanged EventType = "placement.changed"

	// Correlation events.
	EventCorrelationFound   EventType = "correlation.found"
	EventCorrelationDecayed EventType = "correlation.decayed"

	// Auto-revert events.
	EventAutoRevertTriggered EventType = "auto_revert.triggered"
	EventAutoRevertSkipped   EventType = "auto_revert.skipped"

	// Cache events.
	EventCacheHit          EventType = "cache.hit"
	EventCacheMiss         EventType = "cache.miss"
	EventCacheVerifyPassed EventType = "cache.verify_passed"
	EventCacheVerifyFailed EventType = "cache.verify_failed"
)

// Event is the universal telemetry unit.
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	// Context (present on every event — allows joining across a pipeline run).
	PipelineID string `json:"pipeline_id"`
	PR         int    `json:"pr,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Branch     string `json:"branch,omitempty"`

	// Payload (type-specific data).
	Payload json.RawMessage `json:"payload"`
}

// TargetsComputedPayload is the payload for EventTargetsComputed.
type TargetsComputedPayload struct {
	Language  string  `json:"language"`
	Selected  int     `json:"selected"`
	Total     int     `json:"total"`
	SkipRate  float64 `json:"skip_rate"`
	AlwaysRun int     `json:"always_run"`
}

// FlakePayload is the payload for EventFlakeDetected.
type FlakePayload struct {
	Result      TestResult `json:"result"`
	Fingerprint string     `json:"fingerprint"`
}

// RetryPayload is the payload for EventTestRetried.
type RetryPayload struct {
	Original TestResult `json:"original"`
	Retry    TestResult `json:"retry"`
}

// CachePayload is the payload for cache events.
type CachePayload struct {
	Category string `json:"category"`
	CacheKey string `json:"cache_key"`
	Reason   string `json:"reason,omitempty"`
}

// FalseNegativeDetail describes a test that shadow CI missed.
type FalseNegativeDetail struct {
	Test                TestIdentifier `json:"test"`
	Language            string         `json:"language"`
	FailedInMainCI      bool           `json:"failed_in_main_ci"`
	InShadowAffectedSet bool           `json:"in_shadow_affected_set"`
	MissedBecause       string         `json:"missed_because"`
}
