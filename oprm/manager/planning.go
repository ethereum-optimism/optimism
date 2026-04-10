package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/components"
	"github.com/ethereum-optimism/optimism/oprm/providers/ghcli"
	"github.com/ethereum-optimism/optimism/oprm/release"
)

const maxChangeEvidence = 25

type PlanOptions struct {
	Bumps            map[string]release.BumpKind
	ManualTargets    map[string]string
	ConfirmSelection bool
}

type ComponentPlanResult struct {
	ComponentID string
	Proposal    release.VersionProposal
}

func (a *App) PlanRun(ctx context.Context, identifier string, selectedComponentIDs []string, options PlanOptions) (*release.Run, string, []ComponentPlanResult, error) {
	run, path, err := a.store.Load(identifier)
	if err != nil {
		return nil, "", nil, err
	}
	candidateIDs := selectedComponentIDs
	if len(candidateIDs) == 0 {
		candidateIDs = a.registry.IDs()
	}
	orderedCandidates := release.OrderedComponentIDs(candidateIDs)

	discovered, err := a.DiscoverComponentVersions(ctx, orderedCandidates)
	if err != nil {
		return nil, "", nil, err
	}
	compareCache := make(map[string]*ghcli.CompareResult)
	results := make([]ComponentPlanResult, 0, len(discovered))
	newVersions := make(map[string]release.VersionProposal, len(discovered))
	selected := make([]string, 0, len(discovered))
	for _, item := range discovered {
		proposal, err := a.planComponent(ctx, item, compareCache, options)
		if err != nil {
			return nil, "", nil, fmt.Errorf("plan %s: %w", item.Component.ID, err)
		}
		newVersions[item.Component.ID] = proposal
		if len(selectedComponentIDs) > 0 {
			selected = append(selected, item.Component.ID)
		} else if proposal.ResumeDraft || proposal.Changed {
			selected = append(selected, item.Component.ID)
		}
		results = append(results, ComponentPlanResult{ComponentID: item.Component.ID, Proposal: proposal})
	}

	selectionConfirmed := options.ConfirmSelection || len(selectedComponentIDs) > 0
	run.Candidates = orderedCandidates
	run.Components = release.OrderedComponentIDs(selected)
	run.SelectionConfirmed = selectionConfirmed
	run.Versions = newVersions
	now := a.now().UTC()
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	details := make([]string, 0, len(results))
	for _, item := range results {
		proposal := item.Proposal
		detail := fmt.Sprintf("%s: changed=%t", item.ComponentID, proposal.Changed)
		if proposal.Proposed != "" {
			detail += fmt.Sprintf(", target=%s, rc=%s", proposal.TargetRelease, proposal.Proposed)
		}
		details = append(details, detail)
	}
	run.AddTimeline(now, "updated release plan", details...)
	if _, err := a.store.Save(run); err != nil {
		return nil, "", nil, err
	}
	return run, path, results, nil
}

func (a *App) planComponent(ctx context.Context, item ComponentVersionDiscovery, compareCache map[string]*ghcli.CompareResult, options PlanOptions) (release.VersionProposal, error) {
	var latestReleaseVersion string
	var latestReleaseTag string
	if item.LatestRelease != nil {
		latestReleaseVersion = item.LatestRelease.Version
		latestReleaseTag = item.LatestRelease.TagName
	}
	var latestRCVersion string
	if item.LatestRC != nil {
		latestRCVersion = item.LatestRC.Version
	}
	var latestDraftRCVersion string
	if item.LatestDraftRC != nil {
		latestDraftRCVersion = item.LatestDraftRC.Version
	}

	reviewToRef := item.Component.BaseBranch
	reviewToRefKind := "branch"
	if item.LatestDraftRC != nil {
		reviewToRef = item.LatestDraftRC.TagName
		reviewToRefKind = "draft-rc"
	}

	proposal := release.VersionProposal{
		LatestRelease: latestReleaseVersion,
		LatestRC:      latestRCVersion,
		LatestDraftRC: latestDraftRCVersion,
		ComparedRef:   reviewToRef,
		Review: release.ReviewInfo{
			FromRef:   latestReleaseTag,
			ToRef:     reviewToRef,
			ToRefKind: reviewToRefKind,
		},
	}
	if latestReleaseTag == "" {
		proposal.Changed = true
		proposal.ChangeEvidence = []string{"no previous stable release found"}
		proposal.Review.CommitSummaries = []string{"no previous stable release found; review starts from current branch state"}
	} else {
		cacheKey := strings.Join([]string{item.Component.GitHubOwner, item.Component.GitHubRepo, latestReleaseTag, reviewToRef}, ":")
		compare := compareCache[cacheKey]
		if compare == nil {
			var err error
			compare, err = a.gh.CompareCommits(ctx, item.Component.GitHubOwner, item.Component.GitHubRepo, latestReleaseTag, reviewToRef)
			if err != nil {
				return release.VersionProposal{}, err
			}
			compareCache[cacheKey] = compare
		}
		proposal.Review.CompareURL = compare.HTMLURL
		proposal.Review.CommitSummaries = summarizeCommitEvidence(compare.Commits, maxChangeEvidence)
		changedFiles := make([]string, 0, len(compare.Files))
		for _, file := range compare.Files {
			changedFiles = append(changedFiles, file.Filename)
		}
		detection, err := release.DetectComponentChanges(item.Component, reviewToRef, changedFiles)
		if err != nil {
			return release.VersionProposal{}, err
		}
		proposal.Changed = detection.Changed
		proposal.ChangeEvidence = summarizeChangeEvidence(detection.MatchedFiles, maxChangeEvidence)
		if len(proposal.ChangeEvidence) == 0 {
			proposal.ChangeEvidence = []string{fmt.Sprintf("no in-scope changes detected between %s and %s", latestReleaseTag, reviewToRef)}
		}
		if len(proposal.Review.CommitSummaries) == 0 {
			proposal.Review.CommitSummaries = []string{fmt.Sprintf("no commits found between %s and %s", latestReleaseTag, reviewToRef)}
		}
	}

	componentBump := options.Bumps[item.Component.ID]
	manualTarget := options.ManualTargets[item.Component.ID]
	computed, err := release.ProposalForComponent(latestReleaseVersion, latestRCVersion, latestDraftRCVersion, proposal.Changed, componentBump, manualTarget)
	if err != nil {
		return release.VersionProposal{}, err
	}
	proposal.Bump = computed.Bump
	proposal.TargetRelease = computed.TargetRelease
	proposal.Proposed = computed.Proposed
	proposal.ResumeDraft = computed.ResumeDraft
	proposal.ManualOverride = computed.ManualOverride
	if proposal.TargetRelease != "" {
		exists, err := a.gh.TagExists(ctx, item.Component.GitHubOwner, item.Component.GitHubRepo, componentTagName(item.Component, proposal.TargetRelease))
		if err != nil {
			return release.VersionProposal{}, err
		}
		if exists {
			proposal.TargetTagRemoteState = "exists"
		} else {
			proposal.TargetTagRemoteState = "missing"
		}
	}
	if proposal.Proposed != "" {
		exists, err := a.gh.TagExists(ctx, item.Component.GitHubOwner, item.Component.GitHubRepo, componentTagName(item.Component, proposal.Proposed))
		if err != nil {
			return release.VersionProposal{}, err
		}
		if exists {
			proposal.ProposedTagRemoteState = "exists"
		} else {
			proposal.ProposedTagRemoteState = "missing"
		}
	}
	return proposal, nil
}

func componentTagName(spec components.ComponentSpec, version string) string {
	if strings.TrimSpace(version) == "" {
		return ""
	}
	if spec.Kind == components.KindExternalGo {
		return version
	}
	return spec.TagPrefix + "/" + version
}

func summarizeChangeEvidence(files []string, limit int) []string {
	if len(files) <= limit || limit <= 0 {
		return append([]string(nil), files...)
	}
	summary := append([]string(nil), files[:limit]...)
	summary = append(summary, fmt.Sprintf("... and %d more matching files", len(files)-limit))
	return summary
}

func summarizeCommitEvidence(commits []ghcli.CompareCommit, limit int) []string {
	if limit <= 0 || len(commits) <= limit {
		return formatCommitSummaries(commits)
	}
	summary := formatCommitSummaries(commits[:limit])
	summary = append(summary, fmt.Sprintf("... and %d more commits", len(commits)-limit))
	return summary
}

func formatCommitSummaries(commits []ghcli.CompareCommit) []string {
	out := make([]string, 0, len(commits))
	for _, commit := range commits {
		sha := commit.SHA
		if len(sha) > 8 {
			sha = sha[:8]
		}
		message := strings.TrimSpace(strings.Split(commit.Commit.Message, "\n")[0])
		if message == "" {
			message = "<no commit message>"
		}
		out = append(out, fmt.Sprintf("%s %s", sha, message))
	}
	return out
}
