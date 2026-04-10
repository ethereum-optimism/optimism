package journal

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{
		GHLogin:  "alice",
		GitName:  "Alice Example",
		GitEmail: "alice@example.com",
	}, ".oprm/releases")
	run.Tasks = append(run.Tasks, workflow.TaskState{
		ID:        "doctor.gh-cli",
		Title:     "gh installed and authenticated",
		Status:    workflow.StatusCompleted,
		UpdatedAt: now,
		Reason:    "ok",
	})
	run.AddTimeline(now, "initialized release run", "release manager: alice")

	data, err := Marshal(run)
	require.NoError(t, err)
	loaded, err := Unmarshal(data)
	require.NoError(t, err)
	require.Equal(t, run.RunID, loaded.RunID)
	require.Equal(t, run.BaseBranch, loaded.BaseBranch)
	require.Equal(t, run.ReleaseManager, loaded.ReleaseManager)
	require.Len(t, loaded.Tasks, 1)
	require.Equal(t, workflow.StatusCompleted, loaded.Tasks[0].Status)
	require.Len(t, loaded.Timeline, 1)
}

func TestStoreSaveAndLoad(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, store.RunsDir())

	path, err := store.Save(run)
	require.NoError(t, err)
	require.FileExists(t, path)

	loaded, loadedPath, err := store.Load(run.RunID)
	require.NoError(t, err)
	require.Equal(t, path, loadedPath)
	require.Equal(t, run.RunID, loaded.RunID)
}
