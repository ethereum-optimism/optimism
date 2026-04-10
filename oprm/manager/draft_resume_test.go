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

func TestPlanRunResumesExistingDraftRC(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	_, err := store.Save(run)
	require.NoError(t, err)

	gh := &fakeGHProvider{
		releasesByRepo: map[string][]ghcli.Release{
			"ethereum-optimism/optimism": {
				{TagName: "op-node/v1.16.11"},
				{TagName: "op-node/v1.16.12-rc.1", Draft: true, PreRelease: true},
				{TagName: "op-batcher/v1.16.6"},
				{TagName: "kona-node/v1.2.14"},
				{TagName: "op-reth/v1.11.5"},
			},
			"ethereum-optimism/op-geth": {
				{TagName: "v1.101702.0"},
			},
		},
		compareByKey: map[string]*ghcli.CompareResult{
			"ethereum-optimism/optimism:op-node/v1.16.11:op-node/v1.16.12-rc.1": {
				Files: []ghcli.ChangedFile{{Filename: "op-node/flags/flags.go"}},
			},
			"ethereum-optimism/optimism:op-batcher/v1.16.6:develop": {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/optimism:kona-node/v1.2.14:develop":  {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/optimism:op-reth/v1.11.5:develop":    {Files: []ghcli.ChangedFile{}},
			"ethereum-optimism/op-geth:v1.101702.0:optimism":        {Files: []ghcli.ChangedFile{}},
		},
	}
	app := NewWithProviders(DefaultConfig(), io.Discard, io.Discard, store, &fakeGitProvider{}, gh, func() time.Time { return now })

	updatedRun, _, _, err := app.PlanRun(context.Background(), run.RunID, nil, PlanOptions{})
	require.NoError(t, err)
	proposal := updatedRun.Versions["op-node"]
	require.True(t, proposal.ResumeDraft)
	require.Equal(t, "v1.16.12", proposal.TargetRelease)
	require.Equal(t, "v1.16.12-rc.1", proposal.Proposed)
	require.Contains(t, updatedRun.Components, "op-node")
}
