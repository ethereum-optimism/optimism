package engine

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestFlakeReactor_HealthyToSuspected(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	now := time.Now()
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now)
	assert.Equal(t, model.FlakeHealthy, db.Records["pkg/foo/TestFlaky"].State)

	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(1*time.Hour))
	assert.Equal(t, model.FlakeSuspected, db.Records["pkg/foo/TestFlaky"].State)
}

func TestFlakeReactor_SuspectedToQuarantined(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	now := time.Now()
	// Get to suspected.
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now)
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(1*time.Hour))
	assert.Equal(t, model.FlakeSuspected, db.Records["pkg/foo/TestFlaky"].State)

	// More flakes within quarantine window.
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(2*time.Hour))
	assert.Equal(t, model.FlakeQuarantined, db.Records["pkg/foo/TestFlaky"].State)
}

func TestFlakeReactor_AutoRecovery(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	now := time.Now()
	// Get to quarantined.
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now)
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(1*time.Hour))
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(2*time.Hour))
	assert.Equal(t, model.FlakeQuarantined, db.Records["pkg/foo/TestFlaky"].State)

	// 14 days of clean — auto-recover.
	reactor.CheckAutoRecovery(now.Add(15 * 24 * time.Hour))
	assert.Equal(t, model.FlakeHealthy, db.Records["pkg/foo/TestFlaky"].State)
}

func TestFlakeReactor_BelowThreshold(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", time.Now())
	assert.Equal(t, model.FlakeHealthy, db.Records["pkg/foo/TestFlaky"].State)
	assert.Equal(t, 1, db.Records["pkg/foo/TestFlaky"].FlakeCount)
}

func TestFlakeReactor_GetQuarantinedTests(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	now := time.Now()
	// Quarantine one test.
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now)
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(1*time.Hour))
	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(2*time.Hour))

	// Leave another as suspected.
	reactor.ProcessFlake("pkg/bar/TestAlsoFlaky", "go", "fp2", now)
	reactor.ProcessFlake("pkg/bar/TestAlsoFlaky", "go", "fp2", now.Add(1*time.Hour))

	quarantined := reactor.GetQuarantinedTests()
	assert.Len(t, quarantined, 1)
	assert.Contains(t, quarantined, "pkg/foo/TestFlaky")
}

func TestFlakeReactor_SeverityEscalation(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	now := time.Now()
	for i := 0; i < 5; i++ {
		reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(time.Duration(i)*time.Hour))
	}
	assert.Equal(t, model.SeverityMedium, db.Records["pkg/foo/TestFlaky"].Severity)

	for i := 5; i < 12; i++ {
		reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", now.Add(time.Duration(i)*time.Hour))
	}
	assert.Equal(t, model.SeverityHigh, db.Records["pkg/foo/TestFlaky"].Severity)
}

func TestFlakeReactor_RecurrenceAfterFixed(t *testing.T) {
	db := model.NewFlakeDB()
	db.Records["pkg/foo/TestFlaky"] = &model.FlakeRecord{
		TestKey:        "pkg/foo/TestFlaky",
		Language:       "go",
		State:          model.FlakeFixed,
		FlakeCount:     5,
		FirstSeen:      time.Now().Add(-30 * 24 * time.Hour),
		LastSeen:       time.Now().Add(-15 * 24 * time.Hour),
		StateChangedAt: time.Now().Add(-15 * 24 * time.Hour),
	}

	config := DefaultFlakeReactorConfig()
	reactor := NewFlakeReactor(db, nil, nil, config)

	reactor.ProcessFlake("pkg/foo/TestFlaky", "go", "fp1", time.Now())
	assert.Equal(t, model.FlakeSuspected, db.Records["pkg/foo/TestFlaky"].State)
}
