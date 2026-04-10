package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

type taskResult struct {
	Status  workflow.Status
	Summary string
}

func (a *App) ExecuteTask(identifier string, taskID string) error {
	run, _, err := a.store.Load(identifier)
	if err != nil {
		return err
	}
	normalizeRunTasks(run.Tasks)
	taskID = canonicalTaskID(taskID)
	task := run.FindTask(taskID)
	if task == nil {
		return fmt.Errorf("task %q not found in run %q", taskID, run.RunID)
	}
	now := a.now().UTC()
	switch {
	case strings.HasSuffix(taskID, ".review-diff"):
		if task.Status != workflow.StatusNeedsConfirmation && task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for confirmation (status=%s)", taskID, task.Status)
		}
		task.Status = workflow.StatusCompleted
		task.Reason = "review diff confirmed"
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("confirmed task %s", taskID))
	case strings.HasSuffix(taskID, ".prepare-release-notes"):
		if task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for execution (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".prepare-release-notes")
		notesPath, err := a.writeReleaseNotes(run, componentID)
		if err != nil {
			return err
		}
		task.Status = workflow.StatusCompleted
		task.Reason = fmt.Sprintf("release notes written to %s", notesPath)
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("prepared release notes for %s", componentID), notesPath)
	case strings.HasSuffix(taskID, ".create-tag"):
		if task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for execution (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".create-tag")
		result, err := a.createTag(run, componentID)
		if err != nil {
			return err
		}
		task.Status = result.Status
		task.Reason = result.Summary
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("reconciled local tag for %s", componentID), result.Summary)
	case strings.HasSuffix(taskID, ".push-tag"):
		if task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for execution (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".push-tag")
		result, err := a.pushTag(run, componentID)
		if err != nil {
			return err
		}
		task.Status = result.Status
		task.Reason = result.Summary
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("reconciled pushed tag for %s", componentID), result.Summary)
	case strings.HasSuffix(taskID, ".github-draft-release"):
		if task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for execution (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".github-draft-release")
		summary, err := a.createOrUpdateDraftRelease(run, componentID)
		if err != nil {
			return err
		}
		task.Status = workflow.StatusCompleted
		task.Reason = summary
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("created or updated draft release for %s", componentID), summary)
	case strings.HasSuffix(taskID, ".manual-confirm-builds-ready"):
		if task.Status != workflow.StatusNeedsConfirmation && task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for confirmation (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".manual-confirm-builds-ready")
		proposal, ok := run.Versions[componentID]
		if !ok {
			return fmt.Errorf("component %q has no version proposal in run %q", componentID, run.RunID)
		}
		task.Status = workflow.StatusCompleted
		task.Reason = fmt.Sprintf("manual build readiness confirmed for %s", emptyReasonFallback(proposal.Proposed))
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("manually confirmed builds ready for %s", componentID), task.Reason)
	case strings.HasSuffix(taskID, ".rollout"):
		if task.Status != workflow.StatusNeedsConfirmation && task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for confirmation (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".rollout")
		proposal, ok := run.Versions[componentID]
		if !ok {
			return fmt.Errorf("component %q has no version proposal in run %q", componentID, run.RunID)
		}
		task.Status = workflow.StatusCompleted
		task.Reason = fmt.Sprintf("manual rollout confirmed for %s; trigger ./op rollout outside oprm", emptyReasonFallback(proposal.Proposed))
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("manually confirmed rollout for %s", componentID), task.Reason)
	case strings.HasSuffix(taskID, ".finalize-release"):
		if task.Status != workflow.StatusReady {
			return fmt.Errorf("task %q is not ready for execution (status=%s)", taskID, task.Status)
		}
		componentID := strings.TrimSuffix(taskID, ".finalize-release")
		result, err := a.finalizeRelease(run, componentID)
		if err != nil {
			return err
		}
		task.Status = result.Status
		task.Reason = result.Summary
		task.UpdatedAt = now
		run.AddTimeline(now, fmt.Sprintf("finalized release for %s", componentID), result.Summary)
	default:
		return fmt.Errorf("task execution not implemented for %q", taskID)
	}
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	_, err = a.store.Save(run)
	return err
}

func (a *App) releaseNotesPath(run *release.Run, componentID string, proposal release.VersionProposal) string {
	filenameVersion := proposal.Proposed
	if strings.TrimSpace(filenameVersion) == "" {
		filenameVersion = proposal.TargetRelease
	}
	if strings.TrimSpace(filenameVersion) == "" {
		filenameVersion = "draft"
	}
	filenameVersion = strings.ReplaceAll(filenameVersion, "/", "-")
	return filepath.Join(a.store.RunsDir(), run.RunID, "release-notes", fmt.Sprintf("%s-%s.md", componentID, filenameVersion))
}

func (a *App) writeReleaseNotes(run *release.Run, componentID string) (string, error) {
	spec, err := a.componentSpec(componentID)
	if err != nil {
		return "", err
	}
	proposal, ok := run.Versions[componentID]
	if !ok {
		return "", fmt.Errorf("component %q has no version proposal in run %q", componentID, run.RunID)
	}
	path := a.releaseNotesPath(run, componentID, proposal)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create release notes dir %q: %w", dir, err)
	}
	displayVersion := proposal.Proposed
	if strings.TrimSpace(displayVersion) == "" {
		displayVersion = proposal.TargetRelease
	}
	if strings.TrimSpace(displayVersion) == "" {
		displayVersion = "draft"
	}

	var out strings.Builder
	out.WriteString("# ")
	out.WriteString(componentID)
	out.WriteString(" ")
	out.WriteString(displayVersion)
	out.WriteString("\n\n")
	out.WriteString("- Repo: ")
	out.WriteString(spec.GitHubOwner)
	out.WriteString("/")
	out.WriteString(spec.GitHubRepo)
	out.WriteString("\n- Branch: ")
	out.WriteString(spec.BaseBranch)
	out.WriteString("\n- Latest release: ")
	out.WriteString(emptyReasonFallback(proposal.LatestRelease))
	out.WriteString("\n- Target release: ")
	out.WriteString(emptyReasonFallback(proposal.TargetRelease))
	out.WriteString("\n- Proposed RC: ")
	out.WriteString(emptyReasonFallback(proposal.Proposed))
	out.WriteString("\n")
	if proposal.Review.CompareURL != "" {
		out.WriteString("- Compare: ")
		out.WriteString(proposal.Review.CompareURL)
		out.WriteString("\n")
	}
	out.WriteString("\n## Review range\n\n")
	out.WriteString("- From: ")
	out.WriteString(emptyReasonFallback(proposal.Review.FromRef))
	out.WriteString("\n- To: ")
	out.WriteString(emptyReasonFallback(proposal.Review.ToRef))
	out.WriteString("\n")
	if len(proposal.ChangeEvidence) > 0 {
		out.WriteString("\n## Change evidence\n\n")
		for _, item := range proposal.ChangeEvidence {
			out.WriteString("- ")
			out.WriteString(item)
			out.WriteString("\n")
		}
	}
	if len(proposal.Review.CommitSummaries) > 0 {
		out.WriteString("\n## Commits\n\n")
		for _, item := range proposal.Review.CommitSummaries {
			out.WriteString("- ")
			out.WriteString(item)
			out.WriteString("\n")
		}
	}
	if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
		return "", fmt.Errorf("write release notes %q: %w", path, err)
	}
	return path, nil
}

func emptyReasonFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<none>"
	}
	return value
}

func (a *App) createTag(run *release.Run, componentID string) (taskResult, error) {
	proposal, _, tagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, err
	}
	if result, handled, err := a.reconcileCreateTag(run, componentID); err != nil {
		return taskResult{}, err
	} else if handled {
		return result, nil
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, err
	}
	if err := a.git.CreateAnnotatedTag(ctx, checkout, tagName, fmt.Sprintf("%s %s", componentID, emptyReasonFallback(proposal.Proposed))); err != nil {
		return taskResult{}, err
	}
	return taskResult{Status: workflow.StatusCompleted, Summary: fmt.Sprintf("created local tag %s", tagName)}, nil
}

func (a *App) pushTag(run *release.Run, componentID string) (taskResult, error) {
	proposal, spec, tagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, err
	}
	if result, handled, err := a.reconcilePushTag(run, componentID); err != nil {
		return taskResult{}, err
	} else if handled {
		return result, nil
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, err
	}
	localExists, err := a.git.TagExists(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, err
	}
	if !localExists {
		return taskResult{}, fmt.Errorf("local tag %s does not exist; create it before pushing", tagName)
	}
	localCommit, err := a.git.ResolveCommit(ctx, checkout, tagName)
	if err != nil {
		return taskResult{}, err
	}
	intendedCommit, err := a.git.HeadSHA(ctx, checkout)
	if err != nil {
		return taskResult{}, err
	}
	if localCommit != intendedCommit {
		return taskResult{}, reconciliationErrorf("local tag %s points to %s, expected %s", tagName, shortSHA(localCommit), shortSHA(intendedCommit))
	}
	remoteName, err := a.configuredGitRemote(ctx, checkout, spec.GitHubOwner, spec.GitHubRepo)
	if err != nil {
		return taskResult{}, err
	}
	if err := a.git.PushTag(ctx, checkout, remoteName, tagName); err != nil {
		return taskResult{}, err
	}
	verifiedRemote, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
	if err != nil {
		return taskResult{}, err
	}
	if !verifiedRemote {
		return taskResult{}, fmt.Errorf("tag %s was pushed but is not visible on remote %s/%s", tagName, spec.GitHubOwner, spec.GitHubRepo)
	}
	remoteCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
	if err != nil {
		return taskResult{}, err
	}
	if remoteCommit != localCommit {
		return taskResult{}, reconciliationErrorf("remote tag %s points to %s but local tag points to %s", tagName, shortSHA(remoteCommit), shortSHA(localCommit))
	}
	proposal.ProposedTagRemoteState = "exists"
	run.Versions[componentID] = proposal
	return taskResult{Status: workflow.StatusCompleted, Summary: fmt.Sprintf("pushed tag %s to %s/%s", tagName, spec.GitHubOwner, spec.GitHubRepo)}, nil
}

func (a *App) createOrUpdateDraftRelease(run *release.Run, componentID string) (string, error) {
	proposal, spec, tagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return "", err
	}
	body, err := a.releaseNotesBody(run, componentID, proposal)
	if err != nil {
		return "", err
	}
	title := tagName
	ctx := context.Background()
	tagExists, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName)
	if err != nil {
		return "", err
	}
	if !tagExists {
		return "", fmt.Errorf("cannot create draft release for %s because remote tag %s does not exist yet", componentID, tagName)
	}
	releases, err := a.gh.ListReleases(ctx, spec.GitHubOwner, spec.GitHubRepo)
	if err != nil {
		return "", err
	}
	for _, rel := range releases {
		if rel.TagName != tagName {
			continue
		}
		if !rel.Draft {
			return "", fmt.Errorf("release for tag %s already exists and is not a draft: %s", tagName, rel.URL)
		}
		updated, err := a.gh.UpdateDraftRelease(ctx, spec.GitHubOwner, spec.GitHubRepo, rel.ID, title, body)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("updated draft release %s", emptyReasonFallback(updated.URL)), nil
	}
	created, err := a.gh.CreateDraftRelease(ctx, spec.GitHubOwner, spec.GitHubRepo, tagName, spec.BaseBranch, title, body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("created draft release %s", emptyReasonFallback(created.URL)), nil
}

func (a *App) finalizeRelease(run *release.Run, componentID string) (taskResult, error) {
	proposal, spec, rcTagName, err := a.releaseTaskContext(run, componentID)
	if err != nil {
		return taskResult{}, err
	}
	targetTagName := componentTagName(spec, proposal.TargetRelease)
	if strings.TrimSpace(targetTagName) == "" {
		return taskResult{}, fmt.Errorf("component %q has no target release tag", componentID)
	}
	if result, handled, err := a.reconcileFinalizeRelease(run, componentID); err != nil {
		return taskResult{}, err
	} else if handled {
		return result, nil
	}
	ctx := context.Background()
	checkout, err := a.checkoutPath(componentID)
	if err != nil {
		return taskResult{}, err
	}
	rcCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, rcTagName)
	if err != nil {
		return taskResult{}, fmt.Errorf("resolve validated RC tag %s in %s/%s: %w", rcTagName, spec.GitHubOwner, spec.GitHubRepo, err)
	}
	localTargetExists, err := a.git.TagExists(ctx, checkout, targetTagName)
	if err != nil {
		return taskResult{}, err
	}
	if localTargetExists {
		localTargetCommit, err := a.git.ResolveCommit(ctx, checkout, targetTagName)
		if err != nil {
			return taskResult{}, err
		}
		if localTargetCommit != rcCommit {
			return taskResult{}, reconciliationErrorf("local release tag %s points to %s, expected %s from RC %s", targetTagName, shortSHA(localTargetCommit), shortSHA(rcCommit), rcTagName)
		}
	} else {
		message := fmt.Sprintf("%s %s", componentID, emptyReasonFallback(proposal.TargetRelease))
		if err := a.git.CreateAnnotatedTagAtRef(ctx, checkout, targetTagName, rcCommit, message); err != nil {
			return taskResult{}, err
		}
	}
	remoteTargetExists, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
	if err != nil {
		return taskResult{}, err
	}
	if remoteTargetExists {
		remoteTargetCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
		if err != nil {
			return taskResult{}, err
		}
		if remoteTargetCommit != rcCommit {
			return taskResult{}, reconciliationErrorf("remote release tag %s in %s/%s points to %s, expected %s from RC %s", targetTagName, spec.GitHubOwner, spec.GitHubRepo, shortSHA(remoteTargetCommit), shortSHA(rcCommit), rcTagName)
		}
	} else {
		remoteName, err := a.configuredGitRemote(ctx, checkout, spec.GitHubOwner, spec.GitHubRepo)
		if err != nil {
			return taskResult{}, err
		}
		if err := a.git.PushTag(ctx, checkout, remoteName, targetTagName); err != nil {
			return taskResult{}, err
		}
		verifiedRemote, err := a.gh.TagExists(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
		if err != nil {
			return taskResult{}, err
		}
		if !verifiedRemote {
			return taskResult{}, fmt.Errorf("release tag %s was pushed but is not visible on remote %s/%s", targetTagName, spec.GitHubOwner, spec.GitHubRepo)
		}
		remoteTargetCommit, err := a.gh.TagCommit(ctx, spec.GitHubOwner, spec.GitHubRepo, targetTagName)
		if err != nil {
			return taskResult{}, err
		}
		if remoteTargetCommit != rcCommit {
			return taskResult{}, reconciliationErrorf("remote release tag %s points to %s, expected %s from RC %s", targetTagName, shortSHA(remoteTargetCommit), shortSHA(rcCommit), rcTagName)
		}
	}
	releases, err := a.gh.ListReleases(ctx, spec.GitHubOwner, spec.GitHubRepo)
	if err != nil {
		return taskResult{}, err
	}
	for _, rel := range releases {
		if rel.TagName == targetTagName {
			if !rel.Draft {
				proposal.TargetTagRemoteState = "exists"
				run.Versions[componentID] = proposal
				return taskResult{Status: workflow.StatusExternallySatisfied, Summary: fmt.Sprintf("release already published as %s in %s/%s", targetTagName, spec.GitHubOwner, spec.GitHubRepo)}, nil
			}
			published, err := a.gh.PublishRelease(ctx, spec.GitHubOwner, spec.GitHubRepo, rel.ID, targetTagName)
			if err != nil {
				return taskResult{}, err
			}
			proposal.TargetTagRemoteState = "exists"
			run.Versions[componentID] = proposal
			return taskResult{Status: workflow.StatusCompleted, Summary: fmt.Sprintf("published release %s", emptyReasonFallback(published.URL))}, nil
		}
	}
	for _, rel := range releases {
		if rel.TagName != rcTagName {
			continue
		}
		if !rel.Draft {
			return taskResult{}, reconciliationErrorf("release for RC tag %s already exists and is not a draft: %s", rcTagName, emptyReasonFallback(rel.URL))
		}
		published, err := a.gh.PublishRelease(ctx, spec.GitHubOwner, spec.GitHubRepo, rel.ID, targetTagName)
		if err != nil {
			return taskResult{}, err
		}
		proposal.TargetTagRemoteState = "exists"
		run.Versions[componentID] = proposal
		return taskResult{Status: workflow.StatusCompleted, Summary: fmt.Sprintf("published release %s", emptyReasonFallback(published.URL))}, nil
	}
	return taskResult{}, fmt.Errorf("cannot finalize %s because no matching draft release exists for RC tag %s in %s/%s", componentID, rcTagName, spec.GitHubOwner, spec.GitHubRepo)
}

func (a *App) releaseNotesBody(run *release.Run, componentID string, proposal release.VersionProposal) (string, error) {
	notesPath := a.releaseNotesPath(run, componentID, proposal)
	bodyBytes, err := os.ReadFile(notesPath)
	if err != nil {
		return "", fmt.Errorf("read release notes artifact for %s: %w", componentID, err)
	}
	return string(bodyBytes), nil
}

func (a *App) releaseTaskContext(run *release.Run, componentID string) (release.VersionProposal, components.ComponentSpec, string, error) {
	proposal, ok := run.Versions[componentID]
	if !ok {
		return release.VersionProposal{}, components.ComponentSpec{}, "", fmt.Errorf("component %q has no version proposal in run %q", componentID, run.RunID)
	}
	spec, err := a.componentSpec(componentID)
	if err != nil {
		return release.VersionProposal{}, components.ComponentSpec{}, "", err
	}
	tagName := componentTagName(spec, proposal.Proposed)
	if strings.TrimSpace(tagName) == "" {
		return release.VersionProposal{}, components.ComponentSpec{}, "", fmt.Errorf("component %q has no proposed rc tag", componentID)
	}
	return proposal, spec, tagName, nil
}

func (a *App) UpdateSelection(identifier string, selected []string, confirmed bool) (*release.Run, string, error) {
	run, path, err := a.store.Load(identifier)
	if err != nil {
		return nil, "", err
	}
	ordered := make([]string, 0, len(selected))
	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
	}
	for _, id := range run.Candidates {
		if _, ok := selectedSet[id]; ok {
			ordered = append(ordered, id)
		}
	}
	now := a.now().UTC()
	run.Components = ordered
	run.SelectionConfirmed = confirmed
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	run.AddTimeline(now, "updated component selection", fmt.Sprintf("selected: %s", strings.Join(run.Components, ", ")))
	if _, err := a.store.Save(run); err != nil {
		return nil, "", err
	}
	return run, path, nil
}
