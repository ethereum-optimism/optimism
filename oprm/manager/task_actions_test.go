package manager

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/stretchr/testify/require"
)

const testCommitSHA = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

func TestEnsureComponentTasks(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	tasks := rebuildTasks(nil, []string{"op-node"}, true, now)
	require.Len(t, tasks, 7)
	require.Equal(t, "op-node.review-diff", tasks[0].ID)
	require.Equal(t, workflow.StatusNeedsConfirmation, tasks[0].Status)
	require.Equal(t, "op-node.local-tag", tasks[1].ID)
	require.Equal(t, workflow.StatusPending, tasks[1].Status)
	require.Equal(t, "op-node.push-tag", tasks[2].ID)
	require.Equal(t, workflow.StatusPending, tasks[2].Status)
	require.Equal(t, "op-node.github-draft-release", tasks[3].ID)
	require.Equal(t, workflow.StatusPending, tasks[3].Status)
	require.Equal(t, "op-node.docker-build", tasks[4].ID)
	require.Equal(t, workflow.StatusPending, tasks[4].Status)
	require.Equal(t, rolloutTaskID, tasks[5].ID)
	require.Equal(t, workflow.StatusPending, tasks[5].Status)
	require.Equal(t, "op-node.finalize-release", tasks[6].ID)
	require.Equal(t, workflow.StatusPending, tasks[6].Status)
}

func TestExecuteReviewTaskUnblocksLocalTagTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	run.Candidates = []string{"op-node"}
	run.Components = []string{"op-node"}
	run.SelectionConfirmed = true
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.review-diff"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.review-diff").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.local-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask(rolloutTaskID).Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestLoadRunAutoReconcilesReadyTagTasks(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA},
	}
	git := &fakeGitProvider{headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusExternallySatisfied, updatedRun.FindTask("op-node.local-tag").Status)
	require.Equal(t, workflow.StatusExternallySatisfied, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.github-draft-release").Status)
}

func TestAutoReconcileMarksReadyTagTaskFailedOnMismatch(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	git := &fakeGitProvider{headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.local-tag")
	require.Equal(t, workflow.StatusFailed, task.Status)
	require.Contains(t, task.Reason, "points to")
}

func TestLocalTagCreatesTagAndUnblocksPushTagTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{headSHA: testCommitSHA}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.local-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.push-tag").Status)
}

func TestLocalTagTreatsExistingLocalTagAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.local-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "local tag already exists and matches HEAD")
}

func TestLocalTagTreatsExistingRemoteTagAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA},
	}
	git := &fakeGitProvider{headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.local-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "remote tag already exists and matches HEAD")
	require.Equal(t, workflow.StatusExternallySatisfied, updatedRun.FindTask("op-node.push-tag").Status)
}

func TestLocalTagFailsWhenExistingRemoteTagPointsElsewhere(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	git := &fakeGitProvider{headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	err := app.ExecuteTask(run.RunID, "op-node.local-tag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "points to")
}

func TestPushTagUnblocksDraftReleaseTaskAndMaterializesNotes(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{tagExistsByKey: map[string]bool{}, tagCommitByKey: map[string]string{}}
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
		onPushTag: func(_ string, tag string) {
			gh.tagExistsByKey["ethereum-optimism/optimism:"+tag] = true
			gh.tagCommitByKey["ethereum-optimism/optimism:"+tag] = testCommitSHA
		},
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.FileExists(t, filepath.Join(store.RunsDir(), run.RunID, "release-notes", "op-node-v1.2.4-rc.1.md"))
}

func TestPushTagTreatsExistingRemoteTagAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA},
	}
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.push-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "remote tag already exists and matches HEAD")
}

func TestPushTagFailsWhenExistingRemoteTagPointsElsewhere(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	run.FindTask("op-node.local-tag").Status = workflow.StatusExternallySatisfied
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	err = app.ExecuteTask(run.RunID, "op-node.push-tag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "points to")
}

func TestGitHubDraftReleaseCreatesDraftAndUnblocksDockerBuildTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{tagExistsByKey: map[string]bool{}, tagCommitByKey: map[string]string{}, releasesByRepo: map[string][]ghcli.Release{}}
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
		onPushTag: func(_ string, tag string) {
			gh.tagExistsByKey["ethereum-optimism/optimism:"+tag] = true
			gh.tagCommitByKey["ethereum-optimism/optimism:"+tag] = testCommitSHA
		},
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.local-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.github-draft-release"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.Equal(t, workflow.StatusNeedsConfirmation, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask(rolloutTaskID).Status)
}

func TestDockerBuildUnblocksRolloutTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := draftReadyRun(t, store, now)
	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.docker-build"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusNeedsConfirmation, updatedRun.FindTask(rolloutTaskID).Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestRolloutSingletonWaitsForAllDockerBuilds(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	tasks := rebuildTasks(nil, []string{"op-batcher", "op-node"}, true, now)
	find := func(id string) *workflow.TaskState {
		for i := range tasks {
			if tasks[i].ID == id {
				return &tasks[i]
			}
		}
		return nil
	}

	for _, id := range []string{"op-batcher.review-diff", "op-batcher.local-tag", "op-batcher.push-tag", "op-batcher.github-draft-release", "op-node.review-diff", "op-node.local-tag", "op-node.push-tag", "op-node.github-draft-release"} {
		find(id).Status = workflow.StatusCompleted
	}
	find("op-batcher.docker-build").Status = workflow.StatusCompleted
	tasks = rebuildTasks(tasks, []string{"op-batcher", "op-node"}, true, now.Add(time.Minute))
	require.Equal(t, workflow.StatusNeedsConfirmation, find("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, find(rolloutTaskID).Status)

	find("op-node.docker-build").Status = workflow.StatusCompleted
	tasks = rebuildTasks(tasks, []string{"op-batcher", "op-node"}, true, now.Add(2*time.Minute))
	require.Equal(t, workflow.StatusNeedsConfirmation, find(rolloutTaskID).Status)
}

func TestRolloutUnblocksFinalizeReleaseTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := draftReadyRun(t, store, now)
	run.FindTask("op-node.docker-build").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	require.NoError(t, app.ExecuteTask(run.RunID, rolloutTaskID))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask(rolloutTaskID).Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestFinalizeReleaseCreatesTargetTagAndPublishesRelease(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := finalizedReadyRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA},
		releasesByRepo: map[string][]ghcli.Release{
			"ethereum-optimism/optimism": {{ID: 1, TagName: "op-node/v1.2.4-rc.1", Name: "op-node/v1.2.4-rc.1", URL: "https://example/releases/op-node-v1.2.4-rc.1", Draft: true}},
		},
	}
	git := &fakeGitProvider{
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
		onPushTag: func(_ string, tag string) {
			gh.tagExistsByKey["ethereum-optimism/optimism:"+tag] = true
			gh.tagCommitByKey["ethereum-optimism/optimism:"+tag] = testCommitSHA
		},
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.finalize-release"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.finalize-release")
	require.Equal(t, workflow.StatusCompleted, task.Status)
	require.Contains(t, task.Reason, "published release")
	require.True(t, gh.tagExistsByKey["ethereum-optimism/optimism:op-node/v1.2.4"])
	require.Equal(t, testCommitSHA, gh.tagCommitByKey["ethereum-optimism/optimism:op-node/v1.2.4"])
	require.Equal(t, "op-node/v1.2.4", gh.releasesByRepo["ethereum-optimism/optimism"][0].TagName)
	require.False(t, gh.releasesByRepo["ethereum-optimism/optimism"][0].Draft)
}

func TestFinalizeReleaseTreatsPublishedTargetReleaseAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := finalizedReadyRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{
			"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true,
			"ethereum-optimism/optimism:op-node/v1.2.4":      true,
		},
		tagCommitByKey: map[string]string{
			"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA,
			"ethereum-optimism/optimism:op-node/v1.2.4":      testCommitSHA,
		},
		releasesByRepo: map[string][]ghcli.Release{
			"ethereum-optimism/optimism": {{ID: 1, TagName: "op-node/v1.2.4", Name: "op-node/v1.2.4", URL: "https://example/releases/op-node-v1.2.4", Draft: false}},
		},
	}
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4": true},
		resolveCommit: map[string]string{"op-node/v1.2.4": testCommitSHA},
		headSHA:       testCommitSHA,
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.finalize-release")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "release already published as op-node/v1.2.4")
}

func TestLocalTagSupportsLocalOpGethCheckout(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	run.Candidates = []string{"op-geth"}
	run.Components = []string{"op-geth"}
	run.SelectionConfirmed = true
	run.Versions["op-geth"] = release.VersionProposal{TargetRelease: "v1.2.4", Proposed: "v1.2.4-rc.1"}
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-geth.review-diff").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.MonorepoPath = t.TempDir()
	cfg.OpGeth.CheckoutPath = "../op-geth"
	app := NewWithProviders(cfg, nil, nil, store, &fakeGitProvider{headSHA: testCommitSHA}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	require.NoError(t, app.ExecuteTask(run.RunID, "op-geth.local-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-geth.local-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-geth.push-tag").Status)
}

func seededTaskRun(t *testing.T, store *journal.Store, now time.Time) *release.Run {
	t.Helper()
	run := release.NewRun("20260410T120000Z", now, "ethereum-optimism/optimism", "develop", release.ReleaseManager{GHLogin: "alice"}, store.RunsDir())
	run.Candidates = []string{"op-node"}
	run.Components = []string{"op-node"}
	run.SelectionConfirmed = true
	run.Versions["op-node"] = release.VersionProposal{
		LatestRelease:  "v1.2.3",
		TargetRelease:  "v1.2.4",
		Proposed:       "v1.2.4-rc.1",
		ChangeEvidence: []string{"op-node/node.go"},
		Review: release.ReviewInfo{
			FromRef:         "op-node/v1.2.3",
			ToRef:           "develop",
			CompareURL:      "https://example/compare",
			CommitSummaries: []string{"abcdef12 ship op-node rc"},
		},
	}
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.review-diff").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)
	return run
}

func draftReadyRun(t *testing.T, store *journal.Store, now time.Time) *release.Run {
	t.Helper()
	run := seededTaskRun(t, store, now)
	run.FindTask("op-node.local-tag").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.push-tag").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.github-draft-release").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)
	return run
}

func finalizedReadyRun(t *testing.T, store *journal.Store, now time.Time) *release.Run {
	t.Helper()
	run := draftReadyRun(t, store, now)
	run.FindTask("op-node.docker-build").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask(rolloutTaskID).Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)
	return run
}

func TestReleaseNotesBodyPublicAccessor(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now })
	body := app.ReleaseNotesBody(run, "op-node")
	require.Contains(t, body, "## What's Changed in op-node/v1.2.4")
	require.Contains(t, body, "**Full Changelog**: https://example/compare")
	require.Contains(t, body, "**🚢 Docker Image**: https://us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:v1.2.4")
}

func TestWriteReleaseNotesFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	require.NoError(t, os.WriteFile(path, []byte("ok"), 0o644))
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "ok", string(bytes))
}
