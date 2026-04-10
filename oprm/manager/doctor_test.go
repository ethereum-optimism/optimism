package manager

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/journal"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/providers/gitcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
	"github.com/stretchr/testify/require"
)

type fakeGitProvider struct {
	installedErr        error
	config              map[string]string
	configErr           error
	remotesByPath       map[string]map[string]string
	currentBranch       string
	currentBranchByPath map[string]string
	currentBranchErr    error
	headSHA             string
	headSHAErr          error
	resolveRef          map[string]string
	resolveRefErr       error
	resolveCommit       map[string]string
	resolveCommitErr    error
	ancestorByKey       map[string]bool
	ancestorErr         error
	workingTreeClean    bool
	workingTreeState    string
	workingTreeErr      error
	fetchTagsErr        error
	fetchedCalls        []string
	localTags           map[string]bool
	createTagErr        error
	pushTagErr          error
	onPushTag           func(remote string, tag string)
}

func (f *fakeGitProvider) Installed(context.Context) error { return f.installedErr }
func (f *fakeGitProvider) ConfigGet(_ context.Context, key string) (string, error) {
	if f.configErr != nil {
		return "", f.configErr
	}
	return f.config[key], nil
}
func (f *fakeGitProvider) Remotes(_ context.Context, repoPath string) (map[string]string, error) {
	if f.remotesByPath != nil {
		if remotes, ok := f.remotesByPath[repoPath]; ok {
			return remotes, nil
		}
	}
	if strings.Contains(repoPath, "op-geth") {
		return map[string]string{"origin": "git@github.com:ethereum-optimism/op-geth.git"}, nil
	}
	return map[string]string{"origin": "git@github.com:ethereum-optimism/optimism.git"}, nil
}
func (f *fakeGitProvider) CurrentBranch(_ context.Context, repoPath string) (string, error) {
	if f.currentBranchErr != nil {
		return "", f.currentBranchErr
	}
	if f.currentBranchByPath != nil {
		if branch, ok := f.currentBranchByPath[repoPath]; ok {
			return branch, nil
		}
	}
	if f.currentBranch == "" {
		if strings.Contains(repoPath, "op-geth") {
			return "optimism", nil
		}
		return "develop", nil
	}
	return f.currentBranch, nil
}
func (f *fakeGitProvider) HeadSHA(context.Context, string) (string, error) {
	if f.headSHAErr != nil {
		return "", f.headSHAErr
	}
	if f.headSHA == "" {
		return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", nil
	}
	return f.headSHA, nil
}
func (f *fakeGitProvider) ResolveRef(_ context.Context, _ string, ref string) (string, error) {
	if f.resolveRefErr != nil {
		return "", f.resolveRefErr
	}
	if f.resolveRef != nil {
		if value, ok := f.resolveRef[ref]; ok {
			return value, nil
		}
	}
	if f.headSHA != "" {
		return f.headSHA, nil
	}
	return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", nil
}
func (f *fakeGitProvider) ResolveCommit(_ context.Context, _ string, ref string) (string, error) {
	if f.resolveCommitErr != nil {
		return "", f.resolveCommitErr
	}
	if f.resolveCommit != nil {
		if value, ok := f.resolveCommit[ref]; ok {
			return value, nil
		}
	}
	if f.resolveRef != nil {
		if value, ok := f.resolveRef[ref]; ok {
			return value, nil
		}
	}
	if f.headSHA != "" {
		return f.headSHA, nil
	}
	return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", nil
}
func (f *fakeGitProvider) IsAncestor(_ context.Context, _ string, older string, newer string) (bool, error) {
	if f.ancestorErr != nil {
		return false, f.ancestorErr
	}
	if f.ancestorByKey != nil {
		return f.ancestorByKey[older+":"+newer], nil
	}
	return older == newer, nil
}
func (f *fakeGitProvider) WorkingTreeClean(context.Context, string) (bool, string, error) {
	if f.workingTreeErr != nil {
		return false, "", f.workingTreeErr
	}
	if f.workingTreeState == "" && !f.workingTreeClean {
		return true, "", nil
	}
	return f.workingTreeClean, f.workingTreeState, nil
}
func (f *fakeGitProvider) FetchTags(_ context.Context, repoPath string, remote string) error {
	if f.fetchTagsErr != nil {
		return f.fetchTagsErr
	}
	f.fetchedCalls = append(f.fetchedCalls, repoPath+":"+remote)
	return nil
}
func (f *fakeGitProvider) TagExists(_ context.Context, _ string, tag string) (bool, error) {
	if f.localTags == nil {
		return false, nil
	}
	return f.localTags[tag], nil
}
func (f *fakeGitProvider) CreateAnnotatedTag(_ context.Context, _ string, tag string, _ string) error {
	if f.createTagErr != nil {
		return f.createTagErr
	}
	if f.localTags == nil {
		f.localTags = make(map[string]bool)
	}
	if f.resolveCommit == nil {
		f.resolveCommit = make(map[string]string)
	}
	f.localTags[tag] = true
	f.resolveCommit[tag] = f.defaultCommitForRef("HEAD")
	return nil
}

func (f *fakeGitProvider) CreateAnnotatedTagAtRef(_ context.Context, _ string, tag string, ref string, _ string) error {
	if f.createTagErr != nil {
		return f.createTagErr
	}
	if f.localTags == nil {
		f.localTags = make(map[string]bool)
	}
	if f.resolveCommit == nil {
		f.resolveCommit = make(map[string]string)
	}
	f.localTags[tag] = true
	f.resolveCommit[tag] = f.defaultCommitForRef(ref)
	return nil
}
func (f *fakeGitProvider) PushTag(_ context.Context, _ string, remote string, tag string) error {
	if f.pushTagErr != nil {
		return f.pushTagErr
	}
	if f.localTags == nil {
		f.localTags = make(map[string]bool)
	}
	f.localTags[tag] = true
	if f.onPushTag != nil {
		f.onPushTag(remote, tag)
	}
	return nil
}

func (f *fakeGitProvider) defaultCommitForRef(ref string) string {
	if f.resolveCommit != nil {
		if value, ok := f.resolveCommit[ref]; ok {
			return value
		}
	}
	if f.resolveRef != nil {
		if value, ok := f.resolveRef[ref]; ok {
			return value
		}
	}
	if f.headSHA != "" {
		return f.headSHA
	}
	return "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
}

type fakeGHProvider struct {
	installedErr   error
	authErr        error
	user           *ghcli.User
	userErr        error
	releasesByRepo map[string][]ghcli.Release
	repoByName     map[string]*ghcli.Repo
	compareByKey   map[string]*ghcli.CompareResult
	tagExistsByKey map[string]bool
	tagCommitByKey map[string]string
	listCalls      int
	compareCalls   int
	nextReleaseID  int
}

func (f *fakeGHProvider) Installed(context.Context) error     { return f.installedErr }
func (f *fakeGHProvider) Authenticated(context.Context) error { return f.authErr }
func (f *fakeGHProvider) CurrentUser(context.Context) (*ghcli.User, error) {
	return f.user, f.userErr
}
func (f *fakeGHProvider) ListReleases(_ context.Context, owner string, repo string) ([]ghcli.Release, error) {
	f.listCalls++
	if f.releasesByRepo == nil {
		return nil, nil
	}
	return f.releasesByRepo[owner+"/"+repo], nil
}
func (f *fakeGHProvider) CompareCommits(_ context.Context, owner string, repo string, base string, head string) (*ghcli.CompareResult, error) {
	f.compareCalls++
	if f.compareByKey == nil {
		return &ghcli.CompareResult{}, nil
	}
	key := owner + "/" + repo + ":" + base + ":" + head
	if result, ok := f.compareByKey[key]; ok {
		return result, nil
	}
	return &ghcli.CompareResult{}, nil
}
func (f *fakeGHProvider) GetRepo(_ context.Context, owner string, repo string) (*ghcli.Repo, error) {
	if f.repoByName == nil {
		return &ghcli.Repo{FullName: owner + "/" + repo, Permissions: ghcli.RepoPermissions{Push: true}}, nil
	}
	if out, ok := f.repoByName[owner+"/"+repo]; ok {
		return out, nil
	}
	return &ghcli.Repo{FullName: owner + "/" + repo, Permissions: ghcli.RepoPermissions{}}, nil
}
func (f *fakeGHProvider) TagExists(_ context.Context, owner string, repo string, tag string) (bool, error) {
	if f.tagExistsByKey == nil {
		return false, nil
	}
	return f.tagExistsByKey[owner+"/"+repo+":"+tag], nil
}
func (f *fakeGHProvider) TagCommit(_ context.Context, owner string, repo string, tag string) (string, error) {
	if f.tagCommitByKey == nil {
		return "", context.DeadlineExceeded
	}
	value, ok := f.tagCommitByKey[owner+"/"+repo+":"+tag]
	if !ok {
		return "", context.DeadlineExceeded
	}
	return value, nil
}

func (f *fakeGHProvider) CreateDraftRelease(_ context.Context, owner string, repo string, tag string, _ string, title string, _ string) (*ghcli.Release, error) {
	if f.nextReleaseID == 0 {
		f.nextReleaseID = 1
	}
	if f.releasesByRepo == nil {
		f.releasesByRepo = make(map[string][]ghcli.Release)
	}
	rel := ghcli.Release{ID: f.nextReleaseID, TagName: tag, Name: title, URL: "https://example/releases/" + tag, Draft: true, PreRelease: false}
	f.nextReleaseID++
	key := owner + "/" + repo
	f.releasesByRepo[key] = append(f.releasesByRepo[key], rel)
	return &rel, nil
}
func (f *fakeGHProvider) UpdateDraftRelease(_ context.Context, owner string, repo string, releaseID int, title string, _ string) (*ghcli.Release, error) {
	key := owner + "/" + repo
	for i := range f.releasesByRepo[key] {
		if f.releasesByRepo[key][i].ID == releaseID {
			f.releasesByRepo[key][i].Name = title
			f.releasesByRepo[key][i].Draft = true
			f.releasesByRepo[key][i].PreRelease = false
			return &f.releasesByRepo[key][i], nil
		}
	}
	return nil, context.DeadlineExceeded
}
func (f *fakeGHProvider) PublishRelease(_ context.Context, owner string, repo string, releaseID int, tag string) (*ghcli.Release, error) {
	key := owner + "/" + repo
	for i := range f.releasesByRepo[key] {
		if f.releasesByRepo[key][i].ID == releaseID {
			f.releasesByRepo[key][i].TagName = tag
			f.releasesByRepo[key][i].Draft = false
			f.releasesByRepo[key][i].PreRelease = false
			return &f.releasesByRepo[key][i], nil
		}
	}
	return nil, context.DeadlineExceeded
}

func syncedFakeGit(cfg *Config) *fakeGitProvider {
	monorepoPath := cfg.MonorepoPath
	opGethPath := filepath.Clean(filepath.Join(monorepoPath, cfg.OpGeth.CheckoutPath))
	sha := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	return &fakeGitProvider{
		config: map[string]string{
			"user.name":  "Alice Example",
			"user.email": "alice@example.com",
		},
		remotesByPath: map[string]map[string]string{
			monorepoPath: {
				"origin": "git@github.com:ethereum-optimism/optimism.git",
			},
			opGethPath: {
				"origin": "git@github.com:ethereum-optimism/op-geth.git",
			},
		},
		currentBranchByPath: map[string]string{
			monorepoPath: "develop",
			opGethPath:   "optimism",
		},
		headSHA: sha,
		resolveRef: map[string]string{
			"origin/develop":  sha,
			"origin/optimism": sha,
		},
	}
}

func TestDoctorSuccess(t *testing.T) {
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), git, &fakeGHProvider{user: &ghcli.User{Login: "alice"}}, func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	})

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.False(t, report.Blocking())
	require.Equal(t, release.ReleaseManager{
		GHLogin:  "alice",
		GitName:  "Alice Example",
		GitEmail: "alice@example.com",
	}, report.ReleaseManager)
	require.Len(t, report.Checks, 8)
	for _, check := range report.Checks {
		require.Equal(t, workflow.StatusCompleted, check.Status)
	}
	require.Len(t, git.fetchedCalls, 2)
	require.Equal(t, cfg.MonorepoPath+":origin", git.fetchedCalls[0])
	require.Equal(t, filepath.Clean(filepath.Join(cfg.MonorepoPath, cfg.OpGeth.CheckoutPath))+":origin", git.fetchedCalls[1])
}

func TestDoctorFailureBlocksRun(t *testing.T) {
	app := NewWithProviders(DefaultConfig(), io.Discard, io.Discard, journal.NewStore(t.TempDir()), &fakeGitProvider{}, &fakeGHProvider{installedErr: context.DeadlineExceeded}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.True(t, report.Blocking())
	require.Empty(t, report.ReleaseManager.GHLogin)
	require.Equal(t, workflow.StatusFailed, doctorCheckByID(t, report, "doctor.gh-cli").Status)
}

func TestDoctorAllowsBaseBranchToLagOriginWhenResuming(t *testing.T) {
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	git.headSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	git.resolveRef["origin/develop"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	git.resolveRef["origin/optimism"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	git.ancestorByKey = map[string]bool{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": true,
	}
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), git, &fakeGHProvider{user: &ghcli.User{Login: "alice"}}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.False(t, report.Blocking())
	require.Equal(t, workflow.StatusCompleted, doctorCheckByID(t, report, "doctor.monorepo-base-branch-synced").Status)
}

func TestDoctorReportsDivergedBaseBranch(t *testing.T) {
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	git.headSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	git.resolveRef["origin/develop"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	git.resolveRef["origin/optimism"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	git.ancestorByKey = map[string]bool{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": false,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": false,
	}
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), git, &fakeGHProvider{user: &ghcli.User{Login: "alice"}}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.True(t, report.Blocking())
	check := doctorCheckByID(t, report, "doctor.monorepo-base-branch-synced")
	require.Equal(t, workflow.StatusFailed, check.Status)
	require.Contains(t, check.Detail, "diverges from origin/develop")
}

func TestDoctorUsesRemoteMatchingConfiguredTarget(t *testing.T) {
	cfg := DefaultConfig()
	git := syncedFakeGit(cfg)
	git.remotesByPath[cfg.MonorepoPath] = map[string]string{
		"origin":   "git@github.com:nonsense/optimism.git",
		"upstream": "git@github.com:ethereum-optimism/optimism.git",
	}
	git.remotesByPath[filepath.Clean(filepath.Join(cfg.MonorepoPath, cfg.OpGeth.CheckoutPath))] = map[string]string{
		"origin":   "git@github.com:nonsense/op-geth.git",
		"upstream": "git@github.com:ethereum-optimism/op-geth.git",
	}
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), git, &fakeGHProvider{user: &ghcli.User{Login: "alice"}}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.False(t, report.Blocking())
	require.Equal(t, cfg.MonorepoPath+":upstream", git.fetchedCalls[0])
	require.Equal(t, filepath.Clean(filepath.Join(cfg.MonorepoPath, cfg.OpGeth.CheckoutPath))+":upstream", git.fetchedCalls[1])
	require.Equal(t, "monorepo checkout on develop and releasable against upstream/develop", doctorCheckByID(t, report, "doctor.monorepo-base-branch-synced").Title)
	require.Equal(t, "op-geth checkout on optimism and releasable against upstream/optimism", doctorCheckByID(t, report, "doctor.op-geth-base-branch-synced").Title)
}

func TestDoctorReportsMissingPushAccess(t *testing.T) {
	cfg := DefaultConfig()
	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), syncedFakeGit(cfg), &fakeGHProvider{
		user: &ghcli.User{Login: "alice"},
		repoByName: map[string]*ghcli.Repo{
			"ethereum-optimism/optimism": {FullName: "ethereum-optimism/optimism", Permissions: ghcli.RepoPermissions{Push: false}},
			"ethereum-optimism/op-geth":  {FullName: "ethereum-optimism/op-geth", Permissions: ghcli.RepoPermissions{Push: true}},
		},
	}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.True(t, report.Blocking())
	check := doctorCheckByID(t, report, "doctor.repo-push-permissions")
	require.Equal(t, workflow.StatusFailed, check.Status)
	require.Contains(t, check.Detail, "ethereum-optimism/optimism")
	require.NotContains(t, check.Detail, "ethereum-optimism/op-geth; blocked")
}

func TestDoctorUsesConfiguredRepoTargets(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GitHub.Owner = "nonsense"
	cfg.GitHub.Repo = "optimism"
	cfg.OpGeth.Owner = "nonsense"
	cfg.OpGeth.Repo = "op-geth"

	git := syncedFakeGit(cfg)
	git.remotesByPath[cfg.MonorepoPath] = map[string]string{"origin": "git@github.com:nonsense/optimism.git"}
	git.remotesByPath[filepath.Clean(filepath.Join(cfg.MonorepoPath, cfg.OpGeth.CheckoutPath))] = map[string]string{"origin": "git@github.com:nonsense/op-geth.git"}

	app := NewWithProviders(cfg, io.Discard, io.Discard, journal.NewStore(t.TempDir()), git, &fakeGHProvider{
		user: &ghcli.User{Login: "alice"},
		repoByName: map[string]*ghcli.Repo{
			"nonsense/optimism": {FullName: "nonsense/optimism", Permissions: ghcli.RepoPermissions{Push: true}},
			"nonsense/op-geth":  {FullName: "nonsense/op-geth", Permissions: ghcli.RepoPermissions{Push: true}},
		},
	}, time.Now)

	report, err := app.Doctor(context.Background())
	require.NoError(t, err)
	require.False(t, report.Blocking())
	require.Equal(t, "push access to nonsense/optimism and nonsense/op-geth", doctorCheckByID(t, report, "doctor.repo-push-permissions").Title)
}

func doctorCheckByID(t *testing.T, report *DoctorReport, id string) DoctorCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("doctor check %s not found", id)
	return DoctorCheck{}
}

var (
	_ gitcli.Provider = (*fakeGitProvider)(nil)
	_ ghcli.Provider  = (*fakeGHProvider)(nil)
)
