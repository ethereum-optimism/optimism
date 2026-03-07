package events

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStore_EmitAndQuery(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	emitter := NewEmitter(store, "pipeline-1", 42, "abc123", "feat/test")

	err := emitter.Emit(model.EventFlakeDetected, model.FlakePayload{
		Result:      model.TestResult{Test: model.TestIdentifier{Name: "TestFlaky", Package: "pkg/x"}},
		Fingerprint: "go:pkg/x:abc",
	})
	require.NoError(t, err)

	err = emitter.Emit(model.EventTestPassed, map[string]any{"test": "TestGood"})
	require.NoError(t, err)

	// Query all events.
	events, err := store.Query(EventFilter{})
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Query by type.
	flakes, err := store.Query(EventFilter{Types: []model.EventType{model.EventFlakeDetected}})
	require.NoError(t, err)
	assert.Len(t, flakes, 1)
	assert.Equal(t, model.EventFlakeDetected, flakes[0].Type)
	assert.Equal(t, "pipeline-1", flakes[0].PipelineID)
	assert.Equal(t, 42, flakes[0].PR)

	// Query by PR.
	pr := 42
	prEvents, err := store.Query(EventFilter{PR: &pr})
	require.NoError(t, err)
	assert.Len(t, prEvents, 2)
}

func TestLocalStore_TimeFilter(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)
	emitter := NewEmitter(store, "p1", 0, "", "")

	emitter.Emit(model.EventTestPassed, nil)

	future := time.Now().Add(1 * time.Hour)
	events, err := store.Query(EventFilter{After: future})
	require.NoError(t, err)
	assert.Empty(t, events)

	past := time.Now().Add(-1 * time.Hour)
	events, err = store.Query(EventFilter{After: past})
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestLocalStore_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	events, err := store.Query(EventFilter{})
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestLocalStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Write with one store instance.
	store1 := NewLocalStore(dir)
	emitter := NewEmitter(store1, "p1", 0, "", "")
	emitter.Emit(model.EventPlanCreated, map[string]string{"id": "plan-1"})

	// Read with another store instance.
	store2 := NewLocalStore(dir)
	events, err := store2.Query(EventFilter{})
	require.NoError(t, err)
	assert.Len(t, events, 1)

	// Verify the NDJSON file exists.
	entries, _ := os.ReadDir(dir + "/events")
	assert.NotEmpty(t, entries)
}
