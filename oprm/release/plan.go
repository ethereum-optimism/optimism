package release

import (
	"fmt"
	"sort"
)

type PlanComponent struct {
	ComponentID string
	Proposal    VersionProposal
}

type Plan struct {
	Components []PlanComponent
}

func OrderedComponentIDs(selected []string) []string {
	seen := make(map[string]struct{}, len(selected))
	ordered := make([]string, 0, len(selected))
	for _, id := range selected {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	weight := func(id string) int {
		if id == "op-geth" {
			return 0
		}
		return 1
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		wi, wj := weight(ordered[i]), weight(ordered[j])
		if wi != wj {
			return wi < wj
		}
		return ordered[i] < ordered[j]
	})
	return ordered
}

func InferBumpKind(latestRelease string, targetRelease string) BumpKind {
	if latestRelease == "" || targetRelease == "" {
		return ""
	}
	base, err := ValidateManualReleaseVersion(latestRelease)
	if err != nil {
		return ""
	}
	target, err := ValidateManualReleaseVersion(targetRelease)
	if err != nil {
		return ""
	}
	switch {
	case target.parsed.Major() != base.parsed.Major():
		return BumpMajor
	case target.parsed.Minor() != base.parsed.Minor():
		return BumpMinor
	case target.parsed.Patch() != base.parsed.Patch():
		return BumpPatch
	default:
		return ""
	}
}

func ProposalForComponent(latestRelease string, latestRC string, latestDraftRC string, changed bool, bump BumpKind, manualTarget string) (VersionProposal, error) {
	proposal := VersionProposal{
		LatestRelease: latestRelease,
		LatestRC:      latestRC,
		LatestDraftRC: latestDraftRC,
	}
	if manualTarget != "" {
		rc, err := ProposeNextRCFromManualTarget(manualTarget, latestRC)
		if err != nil {
			return VersionProposal{}, err
		}
		proposal.Bump = string(BumpManual)
		proposal.TargetRelease = MustParseVersion(manualTarget).String()
		proposal.Proposed = rc.String()
		proposal.ManualOverride = true
		return proposal, nil
	}
	if latestDraftRC != "" {
		draft, err := ParseVersion(latestDraftRC)
		if err != nil {
			return VersionProposal{}, fmt.Errorf("parse latest draft rc: %w", err)
		}
		if !draft.IsRC() {
			return VersionProposal{}, fmt.Errorf("latest draft rc %q is not an rc version", latestDraftRC)
		}
		proposal.Bump = string(InferBumpKind(latestRelease, draft.Stable().String()))
		proposal.TargetRelease = draft.Stable().String()
		proposal.Proposed = draft.String()
		proposal.ResumeDraft = true
		return proposal, nil
	}
	if bump == "" {
		if !changed {
			return proposal, nil
		}
		bump = BumpPatch
	}
	if latestRelease == "" {
		if latestRC == "" && latestDraftRC == "" {
			proposal.Bump = string(BumpManual)
			proposal.TargetRelease = "v0.0.1"
			proposal.Proposed = "v0.0.1-rc.1"
			return proposal, nil
		}
		return VersionProposal{}, fmt.Errorf("cannot compute a %s bump without a latest stable release; use a manual target version", bump)
	}
	target, err := NextReleaseVersion(latestRelease, bump)
	if err != nil {
		return VersionProposal{}, err
	}
	rc, err := NextRCVersion(target.String(), latestRC)
	if err != nil {
		return VersionProposal{}, err
	}
	proposal.Bump = string(bump)
	proposal.TargetRelease = target.String()
	proposal.Proposed = rc.String()
	return proposal, nil
}
