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
	require.Len(t, tasks, 8)
	require.Equal(t, "op-node.review-diff", tasks[0].ID)
	require.Equal(t, workflow.StatusNeedsConfirmation, tasks[0].Status)
	require.Equal(t, "op-node.prepare-release-notes", tasks[1].ID)
	require.Equal(t, workflow.StatusPending, tasks[1].Status)
	require.Equal(t, "op-node.create-tag", tasks[2].ID)
	require.Equal(t, workflow.StatusPending, tasks[2].Status)
	require.Equal(t, "op-node.push-tag", tasks[3].ID)
	require.Equal(t, workflow.StatusPending, tasks[3].Status)
	require.Equal(t, "op-node.github-draft-release", tasks[4].ID)
	require.Equal(t, workflow.StatusPending, tasks[4].Status)
	require.Equal(t, "op-node.docker-build", tasks[5].ID)
	require.Equal(t, workflow.StatusPending, tasks[5].Status)
	require.Equal(t, "op-node.rollout", tasks[6].ID)
	require.Equal(t, workflow.StatusPending, tasks[6].Status)
	require.Equal(t, "op-node.finalize-release", tasks[7].ID)
	require.Equal(t, workflow.StatusPending, tasks[7].Status)
}

func TestExecuteReviewTaskUnblocksPrepareReleaseNotesTask(t *testing.T) {
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
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.prepare-release-notes").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.create-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestPrepareReleaseNotesWritesArtifact(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)

	app := NewWithProviders(DefaultConfig(), nil, nil, store, &fakeGitProvider{}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.prepare-release-notes")
	require.Equal(t, workflow.StatusCompleted, task.Status)
	require.Contains(t, task.Reason, "release notes written to")
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.create-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
	require.FileExists(t, filepath.Join(store.RunsDir(), run.RunID, "release-notes", "op-node-v1.2.4-rc.1.md"))
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusExternallySatisfied, updatedRun.FindTask("op-node.create-tag").Status)
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.create-tag")
	require.Equal(t, workflow.StatusFailed, task.Status)
	require.Contains(t, task.Reason, "points to")
}

func TestCreateTagUnblocksPushTagTask(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{}
	git := &fakeGitProvider{localTags: map[string]bool{}, headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.create-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.push-tag").Status)
}

func TestCreateTagTreatsExistingLocalTagAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
	}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.create-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "local tag already exists and matches HEAD")
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.push-tag").Status)
}

func TestCreateTagTreatsExistingRemoteTagAsExternallySatisfied(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": testCommitSHA},
	}
	git := &fakeGitProvider{localTags: map[string]bool{}, headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.create-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "remote tag already exists and matches HEAD")
	require.Equal(t, workflow.StatusExternallySatisfied, updatedRun.FindTask("op-node.push-tag").Status)
}

func TestCreateTagFailsWhenExistingRemoteTagPointsElsewhere(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{
		tagExistsByKey: map[string]bool{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": true},
		tagCommitByKey: map[string]string{"ethereum-optimism/optimism:op-node/v1.2.4-rc.1": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	git := &fakeGitProvider{headSHA: testCommitSHA}
	app := NewWithProviders(DefaultConfig(), nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	err := app.ExecuteTask(run.RunID, "op-node.create-tag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "points to")
}

func TestPushTagUnblocksDraftReleaseTask(t *testing.T) {
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.push-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.github-draft-release").Status)
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	task := updatedRun.FindTask("op-node.push-tag")
	require.Equal(t, workflow.StatusExternallySatisfied, task.Status)
	require.Contains(t, task.Reason, "remote tag already exists and matches HEAD")
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.github-draft-release").Status)
}

func TestPushTagFailsWhenExistingRemoteTagPointsElsewhere(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	run.FindTask("op-node.prepare-release-notes").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.create-tag").Status = workflow.StatusExternallySatisfied
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

func TestPushTagUsesRemoteMatchingConfiguredTarget(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	gh := &fakeGHProvider{tagExistsByKey: map[string]bool{}, tagCommitByKey: map[string]string{}}
	cfg := DefaultConfig()
	remoteUsed := ""
	git := &fakeGitProvider{
		localTags:     map[string]bool{"op-node/v1.2.4-rc.1": true},
		resolveCommit: map[string]string{"op-node/v1.2.4-rc.1": testCommitSHA},
		headSHA:       testCommitSHA,
		remotesByPath: map[string]map[string]string{
			cfg.MonorepoPath: {
				"origin":   "git@github.com:nonsense/optimism.git",
				"upstream": "git@github.com:ethereum-optimism/optimism.git",
			},
		},
		onPushTag: func(remote string, tag string) {
			remoteUsed = remote
			gh.tagExistsByKey["ethereum-optimism/optimism:"+tag] = true
			gh.tagCommitByKey["ethereum-optimism/optimism:"+tag] = testCommitSHA
		},
	}
	app := NewWithProviders(cfg, nil, nil, store, git, gh, func() time.Time { return now.Add(time.Minute) })

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	require.Equal(t, "upstream", remoteUsed)
}

func TestCreateOrUpdateDraftReleaseUnblocksBuildConfirmation(t *testing.T) {
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.github-draft-release"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.github-draft-release").Status)
	require.Equal(t, workflow.StatusNeedsConfirmation, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.rollout").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestManualConfirmBuildsReadyUnblocksRolloutTask(t *testing.T) {
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.github-draft-release"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.docker-build"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.docker-build").Status)
	require.Equal(t, workflow.StatusNeedsConfirmation, updatedRun.FindTask("op-node.rollout").Status)
	require.Equal(t, workflow.StatusPending, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestRolloutUnblocksFinalizeReleaseTask(t *testing.T) {
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

	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.prepare-release-notes"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.create-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.push-tag"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.github-draft-release"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.docker-build"))
	require.NoError(t, app.ExecuteTask(run.RunID, "op-node.rollout"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-node.rollout").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-node.finalize-release").Status)
}

func TestFinalizeReleaseCreatesTargetTagAndPublishesRelease(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	run := seededTaskRun(t, store, now)
	run.FindTask("op-node.prepare-release-notes").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.create-tag").Status = workflow.StatusExternallySatisfied
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.push-tag").Status = workflow.StatusExternallySatisfied
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.github-draft-release").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.docker-build").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.rollout").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

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

	notesPath := filepath.Join(store.RunsDir(), run.RunID, "release-notes", "op-node-v1.2.4-rc.1.md")
	require.NoError(t, writeReleaseNotesFixture(notesPath))
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
	run := seededTaskRun(t, store, now)
	run.FindTask("op-node.prepare-release-notes").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.create-tag").Status = workflow.StatusExternallySatisfied
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.push-tag").Status = workflow.StatusExternallySatisfied
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.github-draft-release").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.docker-build").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.FindTask("op-node.rollout").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

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

func TestCreateTagSupportsLocalOpGethCheckout(t *testing.T) {
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
	run.FindTask("op-geth.prepare-release-notes").Status = workflow.StatusCompleted
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err := store.Save(run)
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.MonorepoPath = t.TempDir()
	cfg.OpGeth.CheckoutPath = "../op-geth"
	app := NewWithProviders(cfg, nil, nil, store, &fakeGitProvider{localTags: map[string]bool{}, headSHA: testCommitSHA}, &fakeGHProvider{}, func() time.Time { return now.Add(time.Minute) })
	require.NoError(t, app.ExecuteTask(run.RunID, "op-geth.create-tag"))
	updatedRun, _, err := app.LoadRun(run.RunID)
	require.NoError(t, err)
	require.Equal(t, workflow.StatusCompleted, updatedRun.FindTask("op-geth.create-tag").Status)
	require.Equal(t, workflow.StatusReady, updatedRun.FindTask("op-geth.push-tag").Status)
}

func writeReleaseNotesFixture(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("# op-node v1.2.4-rc.1\n\nnotes\n"), 0o644)
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
