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
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/stretchr/testify/require"
)

func TestPlanRunPersistsSelectedComponentsAndProposals(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	_, err := store.Save(run)
	require.NoError(t, err)

	gh := &fakeGHProvider{
		releasesByRepo: map[string][]ghcli.Release{
			"ethereum-optimism/optimism": {
				{TagName: "op-node/v1.2.3"},
				{TagName: "op-node/v1.2.4-rc.2", Draft: true, PreRelease: true},
			},
			"ethereum-optimism/op-geth": {
				{TagName: "v1.101605.0"},
				{TagName: "v1.101605.1-rc.3", PreRelease: true},
			},
		},
		tagExistsByKey: map[string]bool{
			"ethereum-optimism/optimism:op-node/v1.3.0":      false,
			"ethereum-optimism/optimism:op-node/v1.3.0-rc.1": false,
			"ethereum-optimism/op-geth:v1.101605.1":          true,
			"ethereum-optimism/op-geth:v1.101605.1-rc.4":     false,
		},
		compareByKey: map[string]*ghcli.CompareResult{
			"ethereum-optimism/optimism:op-node/v1.2.3:op-node/v1.2.4-rc.2": {
				HTMLURL: "https://example/compare/op-node",
				Files:   []ghcli.ChangedFile{{Filename: "op-node/rollup/driver.go"}, {Filename: "docs/readme.md"}},
				Commits: []ghcli.CompareCommit{{SHA: "abcdef123456", Commit: ghcli.CompareCommitDetails{Message: "ship op-node rc"}}},
			},
			"ethereum-optimism/op-geth:v1.101605.0:optimism": {
				Files: []ghcli.ChangedFile{{Filename: "core/blockchain.go"}},
			},
		},
	}
	app := NewWithProviders(DefaultConfig(), io.Discard, io.Discard, store, &fakeGitProvider{}, gh, func() time.Time { return now })

	updatedRun, _, results, err := app.PlanRun(context.Background(), run.RunID, []string{"op-node", "op-geth"}, PlanOptions{
		ManualTargets: map[string]string{"op-node": "v1.3.0"},
		Bumps:         map[string]release.BumpKind{"op-geth": release.BumpPatch},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"op-geth", "op-node"}, updatedRun.Components)
	require.Len(t, results, 2)

	opGeth := updatedRun.Versions["op-geth"]
	require.True(t, opGeth.Changed)
	require.Equal(t, []string{"core/blockchain.go"}, opGeth.ChangeEvidence)
	require.Equal(t, "patch", opGeth.Bump)
	require.Equal(t, "v1.101605.1", opGeth.TargetRelease)
	require.Equal(t, "v1.101605.1-rc.4", opGeth.Proposed)
	require.Equal(t, "exists", opGeth.TargetTagRemoteState)
	require.Equal(t, "missing", opGeth.ProposedTagRemoteState)
	require.False(t, opGeth.ResumeDraft)

	opNode := updatedRun.Versions["op-node"]
	require.True(t, opNode.Changed)
	require.Equal(t, []string{"op-node/rollup/driver.go"}, opNode.ChangeEvidence)
	require.Equal(t, "manual", opNode.Bump)
	require.Equal(t, "v1.3.0", opNode.TargetRelease)
	require.Equal(t, "v1.3.0-rc.1", opNode.Proposed)
	require.Equal(t, "missing", opNode.TargetTagRemoteState)
	require.Equal(t, "missing", opNode.ProposedTagRemoteState)
	require.False(t, opNode.ResumeDraft)
	require.True(t, opNode.ManualOverride)
	require.Equal(t, "op-node/v1.2.3", opNode.Review.FromRef)
	require.Equal(t, "op-node/v1.2.4-rc.2", opNode.Review.ToRef)
	require.Equal(t, "draft-rc", opNode.Review.ToRefKind)
	require.Equal(t, "https://example/compare/op-node", opNode.Review.CompareURL)
	require.Equal(t, []string{"abcdef12 ship op-node rc"}, opNode.Review.CommitSummaries)
	require.Equal(t, 2, gh.compareCalls)
	require.Equal(t, workflow.StatusNeedsConfirmation, updatedRun.FindTask("op-geth.review-diff").Status)
	require.Nil(t, updatedRun.FindTask("op-geth.confirm-release-scope"))
}

func TestPlanRunBootstrapsFirstReleaseWhenComponentHasNoHistory(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "nonsense/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	_, err := store.Save(run)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.GitHub.Owner = "nonsense"
	cfg.GitHub.Repo = "optimism"

	gh := &fakeGHProvider{
		releasesByRepo: map[string][]ghcli.Release{
			"nonsense/optimism":         {},
			"ethereum-optimism/op-geth": {},
		},
	}
	app := NewWithProviders(cfg, io.Discard, io.Discard, store, &fakeGitProvider{}, gh, func() time.Time { return now })

	updatedRun, _, results, err := app.PlanRun(context.Background(), run.RunID, []string{"op-node"}, PlanOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)

	proposal := updatedRun.Versions["op-node"]
	require.True(t, proposal.Changed)
	require.Equal(t, []string{"no previous stable release found"}, proposal.ChangeEvidence)
	require.Equal(t, "manual", proposal.Bump)
	require.Equal(t, "v0.0.1", proposal.TargetRelease)
	require.Equal(t, "v0.0.1-rc.1", proposal.Proposed)
}
