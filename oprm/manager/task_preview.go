package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/release"
)

func (a *App) PreviewTaskCommands(run *release.Run, taskID string) ([]string, error) {
	if run == nil {
		return nil, nil
	}
	switch {
	case strings.HasSuffix(taskID, ".create-tag"):
		componentID := strings.TrimSuffix(taskID, ".create-tag")
		proposal, _, tagName, err := a.releaseTaskContext(run, componentID)
		if err != nil {
			return nil, err
		}
		checkout, err := a.checkoutPath(componentID)
		if err != nil {
			return nil, err
		}
		message := fmt.Sprintf("%s %s", componentID, emptyReasonFallback(proposal.Proposed))
		return []string{formatCommand("git", "-C", checkout, "tag", "-a", tagName, "-m", message)}, nil
	case strings.HasSuffix(taskID, ".push-tag"):
		componentID := strings.TrimSuffix(taskID, ".push-tag")
		_, spec, tagName, err := a.releaseTaskContext(run, componentID)
		if err != nil {
			return nil, err
		}
		checkout, err := a.checkoutPath(componentID)
		if err != nil {
			return nil, err
		}
		remoteName, err := a.configuredGitRemote(context.Background(), checkout, spec.GitHubOwner, spec.GitHubRepo)
		if err != nil {
			return nil, err
		}
		return []string{formatCommand("git", "-C", checkout, "push", remoteName, tagName)}, nil
	case strings.HasSuffix(taskID, ".github-draft-release"):
		componentID := strings.TrimSuffix(taskID, ".github-draft-release")
		proposal, spec, tagName, err := a.releaseTaskContext(run, componentID)
		if err != nil {
			return nil, err
		}
		notesPath := a.releaseNotesPath(run, componentID, proposal)
		title := tagName
		if proposal.ResumeDraft {
			return []string{formatCommand("gh", "api", fmt.Sprintf("repos/%s/%s/releases/<matching-draft-release-id>", spec.GitHubOwner, spec.GitHubRepo), "--method", "PATCH", "-f", "name="+title, "-F", "draft=true", "-F", "prerelease=false", "-f", "body=@"+notesPath)}, nil
		}
		return []string{formatCommand("gh", "api", fmt.Sprintf("repos/%s/%s/releases", spec.GitHubOwner, spec.GitHubRepo), "--method", "POST", "-f", "tag_name="+tagName, "-f", "target_commitish="+spec.BaseBranch, "-f", "name="+title, "-F", "draft=true", "-F", "prerelease=false", "-f", "body=@"+notesPath)}, nil
	case strings.HasSuffix(taskID, ".rollout"):
		componentID := strings.TrimSuffix(taskID, ".rollout")
		proposal, _, _, err := a.releaseTaskContext(run, componentID)
		if err != nil {
			return nil, err
		}
		return []string{formatCommand("./op", "rollout", componentID, emptyReasonFallback(proposal.TargetRelease))}, nil
	case strings.HasSuffix(taskID, ".finalize-release"):
		componentID := strings.TrimSuffix(taskID, ".finalize-release")
		proposal, spec, rcTagName, err := a.releaseTaskContext(run, componentID)
		if err != nil {
			return nil, err
		}
		targetTagName := componentTagName(spec, proposal.TargetRelease)
		if strings.TrimSpace(targetTagName) == "" {
			return nil, fmt.Errorf("component %q has no target release tag", componentID)
		}
		checkout, err := a.checkoutPath(componentID)
		if err != nil {
			return nil, err
		}
		remoteName, err := a.configuredGitRemote(context.Background(), checkout, spec.GitHubOwner, spec.GitHubRepo)
		if err != nil {
			return nil, err
		}
		return []string{
			formatCommand("git", "-C", checkout, "tag", "-a", targetTagName, rcTagName, "-m", fmt.Sprintf("%s %s", componentID, emptyReasonFallback(proposal.TargetRelease))),
			formatCommand("git", "-C", checkout, "push", remoteName, targetTagName),
			formatCommand("gh", "api", fmt.Sprintf("repos/%s/%s/releases/<matching-rc-draft-release-id>", spec.GitHubOwner, spec.GitHubRepo), "--method", "PATCH", "-f", "tag_name="+targetTagName, "-F", "draft=false", "-F", "prerelease=false"),
		}, nil
	default:
		return nil, nil
	}
}

func formatCommand(command string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(command))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	safe := true
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', '/', ':', '@', '=':
			continue
		default:
			safe = false
		}
	}
	if safe {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
