package engine

import (
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestAutoRevert_RealFailure_Reverts(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultAutoRevertConfig()
	config.DryRun = false
	ar := NewAutoReverter(nil, db, nil, config)

	decision := ar.Evaluate([]string{"pkg/foo/TestBroken"}, "abc123", 42)
	assert.True(t, decision.ShouldRevert)
	assert.Equal(t, "abc123", decision.CulpritCommit)
	assert.Equal(t, 42, decision.CulpritPR)
	assert.Contains(t, decision.FailedTests, "pkg/foo/TestBroken")
}

func TestAutoRevert_KnownFlake_Skips(t *testing.T) {
	db := model.NewFlakeDB()
	db.Records["pkg/foo/TestFlaky"] = &model.FlakeRecord{
		TestKey: "pkg/foo/TestFlaky",
		State:   model.FlakeQuarantined,
	}

	config := DefaultAutoRevertConfig()
	ar := NewAutoReverter(nil, db, nil, config)

	decision := ar.Evaluate([]string{"pkg/foo/TestFlaky"}, "abc123", 42)
	assert.False(t, decision.ShouldRevert)
	assert.Contains(t, decision.Reason, "known flakes")
}

func TestAutoRevert_NoFailures_Skips(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultAutoRevertConfig()
	ar := NewAutoReverter(nil, db, nil, config)

	decision := ar.Evaluate(nil, "abc123", 42)
	assert.False(t, decision.ShouldRevert)
}

func TestAutoRevert_MixedFlakesAndReal(t *testing.T) {
	db := model.NewFlakeDB()
	db.Records["pkg/foo/TestFlaky"] = &model.FlakeRecord{
		TestKey: "pkg/foo/TestFlaky",
		State:   model.FlakeQuarantined,
	}

	config := DefaultAutoRevertConfig()
	ar := NewAutoReverter(nil, db, nil, config)

	decision := ar.Evaluate([]string{"pkg/foo/TestFlaky", "pkg/bar/TestReal"}, "abc123", 42)
	assert.True(t, decision.ShouldRevert)
	assert.Equal(t, []string{"pkg/bar/TestReal"}, decision.FailedTests)
}

func TestAutoRevert_DryRun(t *testing.T) {
	db := model.NewFlakeDB()
	config := DefaultAutoRevertConfig()
	config.DryRun = true
	ar := NewAutoReverter(nil, db, nil, config)

	decision := ar.Evaluate([]string{"pkg/foo/TestBroken"}, "abc123", 42)
	assert.True(t, decision.ShouldRevert)
	// Dry run still returns should_revert=true but won't actually create PR.
}
