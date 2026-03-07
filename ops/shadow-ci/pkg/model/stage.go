package model

import "strings"

// Stage represents a CI pipeline stage with distinct cost profiles.
type Stage string

const (
	StagePR         Stage = "pr"
	StageMergeQueue Stage = "merge_queue"
	StagePostMerge  Stage = "post_merge"
	StageNightly    Stage = "nightly"
)

// AllStages in order of increasing tolerance for failure.
var AllStages = []Stage{StagePR, StageMergeQueue, StagePostMerge, StageNightly}

// DetermineStage infers the current CI stage from the branch name and schedule flag.
func DetermineStage(branch string, isSchedule bool) Stage {
	if isSchedule {
		return StageNightly
	}
	if branch == "develop" {
		return StagePostMerge
	}
	if isReadonlyQueueBranch(branch) {
		return StageMergeQueue
	}
	return StagePR
}

// isReadonlyQueueBranch checks for GitHub merge queue branch pattern:
// gh-readonly-queue/{base_branch}/pr-{number}-{sha}
func isReadonlyQueueBranch(branch string) bool {
	return strings.HasPrefix(branch, "gh-readonly-queue/")
}

// StageIndex returns the numeric index of a stage (lower = earlier).
func StageIndex(s Stage) int {
	switch s {
	case StagePR:
		return 0
	case StageMergeQueue:
		return 1
	case StagePostMerge:
		return 2
	case StageNightly:
		return 3
	default:
		return 0
	}
}

// ShouldRunAtStage returns true if a category placed at placedAt should run
// at the current stage. A test placed at "pr" runs at every stage. A test
// placed at "nightly" only runs at nightly.
func ShouldRunAtStage(placedAt, current Stage) bool {
	return StageIndex(current) >= StageIndex(placedAt)
}
