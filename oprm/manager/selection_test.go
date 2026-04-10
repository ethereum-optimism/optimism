package manager

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/stretchr/testify/require"
)

func TestPlanRunWithoutExplicitSelectionLeavesSelectionUnconfirmed(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	gh := &fakeGHProvider{
		user: &ghcli.User{Login: "alice"},
		releasesByRepo: map[string][]ghcli.Release{
			"ethereum-optimism/optimism": {
				{TagName: "op-node/v1.2.3"},
				{TagName: "op-batcher/v1.0.0"},
				{TagName: "kona-node/v1.2.3"},
				{TagName: "op-reth/v1.1.0"},
			},
			"ethereum-optimism/op-geth": {{TagName: "v1.101605.0"}},
		},
		compareByKey: map[string]*ghcli.CompareResult{
			"ethereum-optimism/optimism:op-node/v1.2.3:develop":    {Files: []ghcli.ChangedFile{{Filename: "op-node/node.go"}}},
			"ethereum-optimism/optimism:op-batcher/v1.0.0:develop": {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/optimism:kona-node/v1.2.3:develop":  {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/optimism:op-reth/v1.1.0:develop":    {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/op-geth:v1.101605.0:optimism":       {Files: []ghcli.ChangedFile{}},
		},
	}
	app := NewWithProviders(DefaultConfig(), io.Discard, io.Discard, store, &fakeGitProvider{config: map[string]string{"user.name": "Alice Example", "user.email": "alice@example.com"}}, gh, func() time.Time { return now })
	run, _, _, err := app.CreateRun(context.Background())
	require.NoError(t, err)

	updatedRun, _, _, err := app.PlanRun(context.Background(), run.RunID, nil, PlanOptions{})
	require.NoError(t, err)
	require.False(t, updatedRun.SelectionConfirmed)
	require.NotEmpty(t, updatedRun.Candidates)
	require.Equal(t, []string{"op-node"}, updatedRun.Components)
	require.Len(t, updatedRun.Tasks, 8) // doctor only
}

func TestUpdateSelectionConfirmsSelectionAndCreatesTasks(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	run.Candidates = []string{"op-batcher", "op-node"}
	run.Components = []string{"op-node"}
	_, err := store.Save(run)
	require.NoError(t, err)

	app := NewWithProviders(DefaultConfig(), io.Discard, io.Discard, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	updatedRun, _, err := app.UpdateSelection(run.RunID, []string{"op-batcher", "op-node"}, true)
	require.NoError(t, err)
	require.True(t, updatedRun.SelectionConfirmed)
	require.Equal(t, []string{"op-batcher", "op-node"}, updatedRun.Components)
	require.NotNil(t, updatedRun.FindTask("op-node.review-diff"))
	require.NotNil(t, updatedRun.FindTask("op-batcher.review-diff"))
}
