package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

type reconciliationError struct {
	msg string
}

func (e *reconciliationError) Error() string {
	return e.msg
}

func reconciliationErrorf(format string, args ...any) error {
	return &reconciliationError{msg: fmt.Sprintf(format, args...)}
}

func (a *App) autoReconcileReadyTasks(run *release.Run, now time.Time) bool {
	now = now.UTC()
	changedAny := false
	for {
		changedThisPass := false
		for i := range run.Tasks {
			task := &run.Tasks[i]
			if task.Status != workflow.StatusReady && task.Status != workflow.StatusNeedsConfirmation {
				continue
			}
			result, handled, err := a.autoReconcileTask(run, task.ID)
			if err != nil {
				var recErr *reconciliationError
				if !errors.As(err, &recErr) {
					continue
				}
				result = taskResult{Status: workflow.StatusFailed, Summary: recErr.Error()}
				handled = true
			}
			if !handled {
				continue
			}
			if task.Status == result.Status && task.Reason == result.Summary {
				continue
			}
			task.Status = result.Status
			task.Reason = result.Summary
			task.UpdatedAt = now
			run.AddTimeline(now, fmt.Sprintf("auto-reconciled task %s -> %s", task.ID, result.Status), result.Summary)
			changedAny = true
			changedThisPass = true
		}
		if !changedThisPass {
			break
		}
		run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	}
	return changedAny
}

func (a *App) autoReconcileTask(run *release.Run, taskID string) (taskResult, bool, error) {
	switch {
	case strings.HasSuffix(taskID, ".local-tag"):
		return a.reconcileCreateTag(run, strings.TrimSuffix(taskID, ".local-tag"))
	case strings.HasSuffix(taskID, ".push-tag"):
		return a.reconcilePushTag(run, strings.TrimSuffix(taskID, ".push-tag"))
	case taskID == rolloutTaskID:
		return taskResult{}, false, nil
	case strings.HasSuffix(taskID, ".finalize-release"):
		return a.reconcileFinalizeRelease(run, strings.TrimSuffix(taskID, ".finalize-release"))
	default:
		return taskResult{}, false, nil
	}
}

func (a *App) reconcileCreateTag(run *release.Run, componentID string) (taskResult, bool, error) {
	proposal, spec, tagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	intendedCommit, err := a.git.HeadSHA(ctx, checkout)
	if err != nil {
		return taskResult{}, false, err
	}
	remoteExists, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if remoteExists {
		remoteCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
		if err != nil {
			return taskResult{}, false, err
		}
		if remoteCommit != intendedCommit {
			return taskResult{}, false, reconciliationErrorf("remote tag %s in %s/%s points to %s, expected %s", tagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteCommit), shortSHA(intendedCommit))
		}
		proposal.ProposedTagRemoteState = "exists"
		run.Versions[componentID] = proposal
		return taskResult{Status: workflow.StatusExternallySatisfied, Summary: fmt.Sprintf("remote tag already exists and matches HEAD: %s in %s/%s @ %s", tagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteCommit))}, true, nil
	}
	localExists, err := a.git.TagExists(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if !localExists {
		return taskResult{}, false, nil
	}
	localCommit, err := a.git.ResolveCommit(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if localCommit != intendedCommit {
		return taskResult{}, false, reconciliationErrorf("local tag %s points to %s, expected %s", tagName, shortSHA(localCommit), shortSHA(intendedCommit))
	}
	return taskResult{Status: workflow.StatusExternallySatisfied, Summary: fmt.Sprintf("local tag already exists and matches HEAD: %s @ %s", tagName, shortSHA(localCommit))}, true, nil
}

func (a *App) reconcilePushTag(run *release.Run, componentID string) (taskResult, bool, error) {
	proposal, spec, tagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	intendedCommit, err := a.git.HeadSHA(ctx, checkout)
	if err != nil {
		return taskResult{}, false, err
	}
	remoteExists, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if remoteExists {
		remoteCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
		if err != nil {
			return taskResult{}, false, err
		}
		if remoteCommit != intendedCommit {
			return taskResult{}, false, reconciliationErrorf("remote tag %s in %s/%s points to %s, expected %s", tagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteCommit), shortSHA(intendedCommit))
		}
		localExists, err := a.git.TagExists(ctx, checkout, tagName)
		if err != nil {
			return taskResult{}, false, err
		}
		if localExists {
			localCommit, err := a.git.ResolveCommit(ctx, checkout, tagName)
			if err != nil {
				return taskResult{}, false, err
			}
			if localCommit != remoteCommit {
				return taskResult{}, false, reconciliationErrorf("local tag %s points to %s but remote tag points to %s", tagName, shortSHA(localCommit), shortSHA(remoteCommit))
			}
		}
		proposal.ProposedTagRemoteState = "exists"
		run.Versions[componentID] = proposal
		return taskResult{Status: workflow.StatusExternallySatisfied, Summary: fmt.Sprintf("remote tag already exists and matches HEAD: %s in %s/%s @ %s", tagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteCommit))}, true, nil
	}
	localExists, err := a.git.TagExists(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if !localExists {
		return taskResult{}, false, nil
	}
	localCommit, err := a.git.ResolveCommit(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if localCommit != intendedCommit {
		return taskResult{}, false, reconciliationErrorf("local tag %s points to %s, expected %s", tagName, shortSHA(localCommit), shortSHA(intendedCommit))
	}
	return taskResult{}, false, nil
}

func (a *App) reconcileFinalizeRelease(run *release.Run, componentID string) (taskResult, bool, error) {
	proposal, spec, rcTagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	targetTagName := componentTagName(spec, proposal.TargetRelease)
	if strings.TrimSpace(targetTagName) == "" {
		return taskResult{}, false, fmt.Errorf("component %q has no target release tag", componentID)
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, false, err
	}
	rcCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, rcTagName)
	if err != nil {
		return taskResult{}, false, nil
	}
	localTargetExists, err := a.git.TagExists(ctx, checkout, targetTagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if localTargetExists {
		localTargetCommit, err := a.git.ResolveCommit(ctx, checkout, targetTagName)
		if err != nil {
			return taskResult{}, false, err
		}
		if localTargetCommit != rcCommit {
			return taskResult{}, false, reconciliationErrorf("local release tag %s points to %s, expected %s from RC %s", targetTagName, shortSHA(localTargetCommit), shortSHA(rcCommit), rcTagName)
		}
	}
	remoteTargetExists, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if !remoteTargetExists {
		return taskResult{}, false, nil
	}
	remoteTargetCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
	if err != nil {
		return taskResult{}, false, err
	}
	if remoteTargetCommit != rcCommit {
		return taskResult{}, false, reconciliationErrorf("remote release tag %s in %s/%s points to %s, expected %s from RC %s", targetTagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteTargetCommit), shortSHA(rcCommit), rcTagName)
	}
	releases, err := a.gh.ListReleases(ctx, spec.GitHubOwner, spec.GitHubRepo)
	if err != nil {
		return taskResult{}, false, err
	}
	for _, rel := range releases {
		if rel.TagName != targetTagName {
			continue
		}
		if rel.Draft {
			return taskResult{}, false, nil
		}
		proposal.TargetTagRemoteState = "exists"
		run.Versions[componentID] = proposal
		return taskResult{Status: workflow.StatusExternallySatisfied, Summary: fmt.Sprintf("release already published as %s in %s/%s", targetTagName, spec.GitHubOwner, spec.GitHubRepo)}, true, nil
	}
	return taskResult{}, false, nil
}
