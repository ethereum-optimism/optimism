package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

type DoctorCheck struct {
	ID     string
	Title  string
	Status workflow.Status
	Detail string
}

type DoctorReport struct {
	ReleaseManager release.ReleaseManager
	Checks         []DoctorCheck
}

func (r *DoctorReport) Blocking() bool {
	for _, check := range r.Checks {
		if check.Status != workflow.StatusCompleted {
			return true
		}
	}
	return false
}

func (r *DoctorReport) TaskStates(now time.Time) []workflow.TaskState {
	out := make([]workflow.TaskState, 0, len(r.Checks))
	for _, check := range r.Checks {
		out = append(out, workflow.TaskState{
			ID:        check.ID,
			Title:     check.Title,
			Status:    check.Status,
			UpdatedAt: now.UTC(),
			Reason:    check.Detail,
		})
	}
	return out
}

func (a *App) Doctor(ctx context.Context) (*DoctorReport, error) {
	report := &DoctorReport{}

	add := func(id, title string, err error) {
		status := workflow.StatusCompleted
		detail := "ok"
		if err != nil {
			status = workflow.StatusFailed
			detail = err.Error()
		}
		report.Checks = append(report.Checks, DoctorCheck{
			ID:     id,
			Title:  title,
			Status: status,
			Detail: detail,
		})
	}

	ghInstalledErr := a.gh.Installed(ctx)
	add("doctor.gh-installed", "gh installed", ghInstalledErr)

	var ghAuthErr error
	if ghInstalledErr == nil {
		ghAuthErr = a.gh.Authenticated(ctx)
	} else {
		ghAuthErr = fmt.Errorf("blocked: gh is not installed")
	}
	add("doctor.gh-authenticated", "gh authenticated", ghAuthErr)

	gitInstalledErr := a.git.Installed(ctx)
	add("doctor.git-installed", "git installed", gitInstalledErr)

	var gitName, gitEmail string
	var gitConfiguredErr error
	if gitInstalledErr == nil {
		gitName, gitConfiguredErr = a.git.ConfigGet(ctx, "user.name")
		if gitConfiguredErr == nil {
			gitEmail, gitConfiguredErr = a.git.ConfigGet(ctx, "user.email")
		}
	} else {
		gitConfiguredErr = fmt.Errorf("blocked: git is not installed")
	}
	add("doctor.git-configured", "git configured", gitConfiguredErr)

	monorepoPath := a.MonorepoPath()
	opGethPath := a.OpGethPath()
	monorepoSpec, monorepoSpecErr := a.componentSpec("op-node")
	opGethSpec, opGethSpecErr := a.componentSpec("op-geth")

	var monorepoRemote string
	var monorepoFetchTagsErr error
	switch {
	case gitInstalledErr != nil:
		monorepoFetchTagsErr = fmt.Errorf("blocked: git is not installed")
	case monorepoSpecErr != nil:
		monorepoFetchTagsErr = monorepoSpecErr
	default:
		monorepoRemote, monorepoFetchTagsErr = a.configuredGitRemote(ctx, monorepoPath, monorepoSpec.GitHubOwner, monorepoSpec.GitHubRepo)
		if monorepoFetchTagsErr == nil {
			monorepoFetchTagsErr = a.git.FetchTags(ctx, monorepoPath, monorepoRemote)
		}
	}
	add("doctor.git-fetch-tags-monorepo", fmt.Sprintf("fetched monorepo tags from %s for %s/%s (%s)", emptyReasonFallback(monorepoRemote), a.Config.GitHub.Owner, a.Config.GitHub.Repo, monorepoPath), monorepoFetchTagsErr)

	var opGethRemote string
	var opGethFetchTagsErr error
	switch {
	case gitInstalledErr != nil:
		opGethFetchTagsErr = fmt.Errorf("blocked: git is not installed")
	case opGethSpecErr != nil:
		opGethFetchTagsErr = opGethSpecErr
	default:
		opGethRemote, opGethFetchTagsErr = a.configuredGitRemote(ctx, opGethPath, opGethSpec.GitHubOwner, opGethSpec.GitHubRepo)
		if opGethFetchTagsErr == nil {
			opGethFetchTagsErr = a.git.FetchTags(ctx, opGethPath, opGethRemote)
		}
	}
	add("doctor.git-fetch-tags-op-geth", fmt.Sprintf("fetched op-geth tags from %s for %s/%s (%s)", emptyReasonFallback(opGethRemote), a.Config.OpGeth.Owner, a.Config.OpGeth.Repo, opGethPath), opGethFetchTagsErr)

	var monorepoBranchSyncErr error
	switch {
	case gitInstalledErr != nil:
		monorepoBranchSyncErr = fmt.Errorf("blocked: git is not installed")
	case monorepoSpecErr != nil:
		monorepoBranchSyncErr = monorepoSpecErr
	case monorepoFetchTagsErr != nil:
		monorepoBranchSyncErr = fmt.Errorf("blocked: monorepo tags could not be fetched")
	default:
		monorepoBranchSyncErr = a.checkBranchSynced(ctx, monorepoPath, monorepoRemote, monorepoSpec.BaseBranch)
	}
	add("doctor.monorepo-base-branch-synced", fmt.Sprintf("monorepo checkout on %s and releasable against %s/%s", emptyReasonFallback(monorepoSpec.BaseBranch), emptyReasonFallback(monorepoRemote), emptyReasonFallback(monorepoSpec.BaseBranch)), monorepoBranchSyncErr)

	var opGethBranchSyncErr error
	switch {
	case gitInstalledErr != nil:
		opGethBranchSyncErr = fmt.Errorf("blocked: git is not installed")
	case opGethSpecErr != nil:
		opGethBranchSyncErr = opGethSpecErr
	case opGethFetchTagsErr != nil:
		opGethBranchSyncErr = fmt.Errorf("blocked: op-geth tags could not be fetched")
	default:
		opGethBranchSyncErr = a.checkBranchSynced(ctx, opGethPath, opGethRemote, opGethSpec.BaseBranch)
	}
	add("doctor.op-geth-base-branch-synced", fmt.Sprintf("op-geth checkout on %s and releasable against %s/%s", emptyReasonFallback(opGethSpec.BaseBranch), emptyReasonFallback(opGethRemote), emptyReasonFallback(opGethSpec.BaseBranch)), opGethBranchSyncErr)

	var ghLogin string
	var ghUserErr error
	if ghInstalledErr == nil && ghAuthErr == nil {
		user, err := a.gh.CurrentUser(ctx)
		if err != nil {
			ghUserErr = err
		} else {
			ghLogin = user.Login
		}
	} else {
		ghUserErr = fmt.Errorf("blocked: GitHub user cannot be resolved until gh is installed and authenticated")
	}

	report.ReleaseManager = release.ReleaseManager{
		GHLogin:  strings.TrimSpace(ghLogin),
		GitName:  strings.TrimSpace(gitName),
		GitEmail: strings.TrimSpace(gitEmail),
	}
	if report.ReleaseManager.GHLogin == "" || report.ReleaseManager.GitName == "" || report.ReleaseManager.GitEmail == "" {
		if ghUserErr == nil {
			ghUserErr = fmt.Errorf("release manager is incomplete")
		}
		add("doctor.release-manager-detected", "release manager detected", ghUserErr)
	} else {
		add("doctor.release-manager-detected", "release manager detected", nil)
	}

	var monorepoPushErr error
	if ghInstalledErr == nil && ghAuthErr == nil {
		repo, err := a.gh.GetRepo(ctx, a.Config.GitHub.Owner, a.Config.GitHub.Repo)
		if err != nil {
			monorepoPushErr = err
		} else if !repo.Permissions.Push {
			monorepoPushErr = fmt.Errorf("GitHub user does not have push access to %s/%s", a.Config.GitHub.Owner, a.Config.GitHub.Repo)
		}
	} else {
		monorepoPushErr = fmt.Errorf("blocked: cannot check push access until gh is installed and authenticated")
	}
	add("doctor.repo-push-monorepo", fmt.Sprintf("monorepo push access to %s/%s", a.Config.GitHub.Owner, a.Config.GitHub.Repo), monorepoPushErr)

	var opGethPushErr error
	if ghInstalledErr == nil && ghAuthErr == nil {
		repo, err := a.gh.GetRepo(ctx, a.Config.OpGeth.Owner, a.Config.OpGeth.Repo)
		if err != nil {
			opGethPushErr = err
		} else if !repo.Permissions.Push {
			opGethPushErr = fmt.Errorf("GitHub user does not have push access to %s/%s", a.Config.OpGeth.Owner, a.Config.OpGeth.Repo)
		}
	} else {
		opGethPushErr = fmt.Errorf("blocked: cannot check push access until gh is installed and authenticated")
	}
	add("doctor.repo-push-op-geth", fmt.Sprintf("push access to %s/%s", a.Config.OpGeth.Owner, a.Config.OpGeth.Repo), opGethPushErr)

	return report, nil
}

func (a *App) checkBranchSynced(ctx context.Context, repoPath string, remoteName string, expectedBranch string) error {
	branch, err := a.git.CurrentBranch(ctx, repoPath)
	if err != nil {
		return err
	}
	if branch != expectedBranch {
		return fmt.Errorf("current git branch %q does not match expected branch %q", branch, expectedBranch)
	}
	headSHA, err := a.git.HeadSHA(ctx, repoPath)
	if err != nil {
		return err
	}
	remoteRef := remoteName + "/" + expectedBranch
	remoteSHA, err := a.git.ResolveRef(ctx, repoPath, remoteRef)
	if err != nil {
		return err
	}
	if headSHA == remoteSHA {
		return nil
	}
	headReachableFromRemote, err := a.git.IsAncestor(ctx, repoPath, headSHA, remoteSHA)
	if err != nil {
		return err
	}
	if headReachableFromRemote {
		return nil
	}
	remoteReachableFromHead, err := a.git.IsAncestor(ctx, repoPath, remoteSHA, headSHA)
	if err != nil {
		return err
	}
	if remoteReachableFromHead {
		return fmt.Errorf("local HEAD %s is ahead of %s/%s %s", shortSHA(headSHA), remoteName, expectedBranch, shortSHA(remoteSHA))
	}
	return fmt.Errorf("local HEAD %s diverges from %s/%s %s", shortSHA(headSHA), remoteName, expectedBranch, shortSHA(remoteSHA))
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
