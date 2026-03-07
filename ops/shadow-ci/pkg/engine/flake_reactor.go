package engine

import (
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// FlakeReactor watches for flake events and manages the flake lifecycle.
type FlakeReactor struct {
	db      *model.FlakeDB
	store   events.Store
	emitter *events.Emitter
	config  FlakeReactorConfig
}

// FlakeReactorConfig controls lifecycle thresholds.
type FlakeReactorConfig struct {
	SuspectThreshold    int           // flakes in SuspectWindow to become suspected (default: 2)
	SuspectWindow       time.Duration // window for suspect threshold (default: 24h)
	QuarantineThreshold int           // flakes in QuarantineWindow to become quarantined (default: 3)
	QuarantineWindow    time.Duration // window for quarantine threshold (default: 7 days)
	FixedAfterClean     time.Duration // clean period to auto-recover (default: 14 days)
}

// DefaultFlakeReactorConfig returns sensible defaults.
func DefaultFlakeReactorConfig() FlakeReactorConfig {
	return FlakeReactorConfig{
		SuspectThreshold:    2,
		SuspectWindow:       24 * time.Hour,
		QuarantineThreshold: 3,
		QuarantineWindow:    7 * 24 * time.Hour,
		FixedAfterClean:     14 * 24 * time.Hour,
	}
}

// NewFlakeReactor creates a new FlakeReactor.
func NewFlakeReactor(db *model.FlakeDB, store events.Store, emitter *events.Emitter, config FlakeReactorConfig) *FlakeReactor {
	return &FlakeReactor{
		db:      db,
		store:   store,
		emitter: emitter,
		config:  config,
	}
}

// ProcessFlake handles a detected flake for a given test key.
func (fr *FlakeReactor) ProcessFlake(testKey, language, fingerprint string, now time.Time) {
	record, ok := fr.db.Records[testKey]
	if !ok {
		record = &model.FlakeRecord{
			TestKey:        testKey,
			Language:       language,
			Fingerprint:    fingerprint,
			State:          model.FlakeHealthy,
			Severity:       model.SeverityLow,
			FirstSeen:      now,
			StateChangedAt: now,
		}
		fr.db.Records[testKey] = record
	}

	record.FlakeCount++
	record.LastSeen = now
	record.Fingerprint = fingerprint

	switch record.State {
	case model.FlakeHealthy:
		if fr.countRecentFlakes(record, fr.config.SuspectWindow, now) >= fr.config.SuspectThreshold {
			fr.transition(record, model.FlakeSuspected, now)
		}
	case model.FlakeSuspected:
		if fr.countRecentFlakes(record, fr.config.QuarantineWindow, now) >= fr.config.QuarantineThreshold {
			fr.transition(record, model.FlakeQuarantined, now)
			fr.emitQuarantined(record)
		}
	case model.FlakeQuarantined, model.FlakeShaking, model.FlakeDiagnosed:
		// Already quarantined or beyond — update severity.
		fr.updateSeverity(record)
	case model.FlakeFixed, model.FlakeAccepted:
		// Recurrence — move back to suspected.
		fr.transition(record, model.FlakeSuspected, now)
	}
}

// CheckAutoRecovery looks for quarantined tests that have been clean and recovers them.
func (fr *FlakeReactor) CheckAutoRecovery(now time.Time) {
	for _, record := range fr.db.Records {
		if record.State == model.FlakeQuarantined || record.State == model.FlakeSuspected {
			if now.Sub(record.LastSeen) >= fr.config.FixedAfterClean {
				fr.transition(record, model.FlakeHealthy, now)
				if fr.emitter != nil {
					fr.emitter.Emit(model.EventFlakeRestored, map[string]string{
						"test_key": record.TestKey,
					})
				}
			}
		}
	}
}

// GetQuarantinedTests returns test keys currently quarantined.
func (fr *FlakeReactor) GetQuarantinedTests() []string {
	return fr.db.QuarantinedKeys()
}

func (fr *FlakeReactor) transition(record *model.FlakeRecord, newState model.FlakeState, now time.Time) {
	oldState := record.State
	record.State = newState
	record.StateChangedAt = now

	if fr.emitter != nil {
		fr.emitter.Emit(model.EventFlakeStateChanged, map[string]interface{}{
			"test_key":  record.TestKey,
			"old_state": string(oldState),
			"new_state": string(newState),
		})
	}
}

func (fr *FlakeReactor) emitQuarantined(record *model.FlakeRecord) {
	if fr.emitter != nil {
		fr.emitter.Emit(model.EventFlakeQuarantined, map[string]string{
			"test_key":    record.TestKey,
			"fingerprint": record.Fingerprint,
			"severity":    string(record.Severity),
		})
	}
}

func (fr *FlakeReactor) updateSeverity(record *model.FlakeRecord) {
	switch {
	case record.FlakeCount >= 20:
		record.Severity = model.SeverityCritical
	case record.FlakeCount >= 10:
		record.Severity = model.SeverityHigh
	case record.FlakeCount >= 5:
		record.Severity = model.SeverityMedium
	default:
		record.Severity = model.SeverityLow
	}
}

// countRecentFlakes approximates recent flake count using FlakeCount and time window.
// A more precise implementation would query the event store, but this heuristic
// works well for lifecycle transitions.
func (fr *FlakeReactor) countRecentFlakes(record *model.FlakeRecord, window time.Duration, now time.Time) int {
	if record.FirstSeen.After(now.Add(-window)) {
		// All flakes are within the window.
		return record.FlakeCount
	}
	// Estimate: assume roughly uniform distribution of flakes over lifetime.
	lifetime := now.Sub(record.FirstSeen)
	if lifetime == 0 {
		return record.FlakeCount
	}
	rate := float64(record.FlakeCount) / float64(lifetime)
	estimated := int(rate * float64(window))
	if estimated < 1 && record.FlakeCount > 0 {
		return 1
	}
	return estimated
}
