package manager

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/stretchr/testify/require"
)

func TestPreviewTaskCommandsForPushTagUsesMatchingRemote(t *testing.T) {
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	git.remotesByPath[cfg.MonorepoPath] = map[string]string{
		"origin":   "git@github.com:nonsense/optimism.git",
		"upstream": "git@github.com:ethereum-optimism/optimism.git",
	}
	app := NewWithProviders(cfg, nil, nil, journal.NewStore(t.TempDir()), git, &fakeGHProvider{}, time.Now)
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, ".oprm/releases")
	run.Versions["op-node"] = release.VersionProposal{Proposed: "v1.2.4-rc.1"}

	commands, err := app.PreviewTaskCommands(run, "op-node.push-tag")
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Contains(t, commands[0], "push upstream op-node/v1.2.4-rc.1")
}

func TestPreviewTaskCommandsForRolloutSuggestsOpRollout(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	cfg := DefaultConfig()
	app := NewWithProviders(cfg, nil, nil, store, syncedFakeGit(cfg), &fakeGHProvider{}, time.Now)
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, store.RunsDir())
	run.Versions["op-node"] = release.VersionProposal{TargetRelease: "v1.2.4", Proposed: "v1.2.4-rc.1"}

	commands, err := app.PreviewTaskCommands(run, "stack.rollout")
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, "./op rollout", commands[0])
}

func TestPreviewTaskCommandsForFinalizeReleaseIncludesResolvedReleaseID(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	git.remotesByPath[cfg.MonorepoPath] = map[string]string{
		"origin":   "git@github.com:nonsense/optimism.git",
		"upstream": "git@github.com:ethereum-optimism/optimism.git",
	}
	gh := &fakeGHProvider{releasesByRepo: map[string][]ghcli.Release{
		"ethereum-optimism/optimism": {{ID: 42, TagName: "op-node/v1.2.4-rc.1", Draft: true}},
	}}
	app := NewWithProviders(cfg, nil, nil, store, git, gh, time.Now)
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, store.RunsDir())
	run.Versions["op-node"] = release.VersionProposal{TargetRelease: "v1.2.4", Proposed: "v1.2.4-rc.1"}

	commands, err := app.PreviewTaskCommands(run, "op-node.finalize-release")
	require.NoError(t, err)
	require.Len(t, commands, 3)
	require.Contains(t, commands[0], "tag -a op-node/v1.2.4 op-node/v1.2.4-rc.1")
	require.Contains(t, commands[1], "push upstream op-node/v1.2.4")
	require.Contains(t, commands[2], "repos/ethereum-optimism/optimism/releases/42")
	require.Contains(t, commands[2], "tag_name=op-node/v1.2.4")
	require.Contains(t, commands[2], "draft=false")
}

func TestPreviewTaskCommandsForDraftReleaseIncludesConfiguredRepo(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	cfg := DefaultConfig()
	app := NewWithProviders(cfg, nil, nil, store, syncedFakeGit(cfg), &fakeGHProvider{}, time.Now)
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, store.RunsDir())
	run.Versions["kona-node"] = release.VersionProposal{Proposed: "v0.0.1-rc.1"}

	commands, err := app.PreviewTaskCommands(run, "kona-node.github-draft-release")
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Contains(t, commands[0], "repos/ethereum-optimism/optimism/releases")
	require.Contains(t, commands[0], "prerelease=false")
}

func TestPreviewTaskCommandsForResumedDraftReleaseIncludesResolvedReleaseID(t *testing.T) {
	store := journal.NewStore(filepath.Join(t.TempDir(), ".oprm", "releases"))
	cfg := DefaultConfig()
	gh := &fakeGHProvider{releasesByRepo: map[string][]ghcli.Release{
		"ethereum-optimism/optimism": {{ID: 7, TagName: "kona-node/v0.0.1-rc.1", Draft: true}},
	}}
	app := NewWithProviders(cfg, nil, nil, store, syncedFakeGit(cfg), gh, time.Now)
	run := release.NewRun("run-1", time.Now(), "ethereum-optimism/optimism", "develop", release.ReleaseManager{}, store.RunsDir())
	run.Versions["kona-node"] = release.VersionProposal{Proposed: "v0.0.1-rc.1", ResumeDraft: true}

	commands, err := app.PreviewTaskCommands(run, "kona-node.github-draft-release")
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Contains(t, commands[0], "repos/ethereum-optimism/optimism/releases/7")
	require.Contains(t, commands[0], "draft=true")
}
