package manager

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

var componentTaskTemplates = []struct {
	Suffix string
	Title  string
	Status workflow.Status
}{
	{Suffix: "review-diff", Title: "Review and confirm diff for %s", Status: workflow.StatusNeedsConfirmation},
	{Suffix: "local-tag", Title: "Create local tag for %s", Status: workflow.StatusReady},
	{Suffix: "push-tag", Title: "Push tag for %s", Status: workflow.StatusReady},
	{Suffix: "github-draft-release", Title: "GitHub draft release for %s", Status: workflow.StatusReady},
	{Suffix: "docker-build", Title: "Confirm docker build readiness for %s", Status: workflow.StatusNeedsConfirmation},
}

var finalizeTaskTemplate = struct {
	Suffix string
	Title  string
	Status workflow.Status
}{
	Suffix: "finalize-release",
	Title:  "Finalize release for %s",
	Status: workflow.StatusReady,
}

var rolloutTaskTemplate = struct {
	ID     string
	Title  string
	Status workflow.Status
}{
	ID:     "stack.rollout",
	Title:  "Trigger rollout workflow for the selected stack",
	Status: workflow.StatusNeedsConfirmation,
}

var taskIDSuffixAliases = map[string]string{
	"create-tag":                     "local-tag",
	"create-or-update-draft-release": "github-draft-release",
	"manual-confirm-builds-ready":    "docker-build",
}

const rolloutTaskID = "stack.rollout"

func canonicalTaskID(taskID string) string {
	componentID, suffix, ok := strings.Cut(taskID, ".")
	if !ok {
		return taskID
	}
	if suffix == "rollout" {
		return rolloutTaskID
	}
	if canonicalSuffix, ok := taskIDSuffixAliases[suffix]; ok {
		return componentID + "." + canonicalSuffix
	}
	return taskID
}

func normalizeRunTasks(runTasks []workflow.TaskState) bool {
	changed := false
	for i := range runTasks {
		canonicalID := canonicalTaskID(runTasks[i].ID)
		if canonicalID == runTasks[i].ID {
			continue
		}
		runTasks[i].ID = canonicalID
		changed = true
	}
	return changed
}

func rebuildTasks(existing []workflow.TaskState, componentIDs []string, selectionConfirmed bool, now time.Time) []workflow.TaskState {
	selectedSet := make(map[string]struct{}, len(componentIDs))
	for _, id := range componentIDs {
		selectedSet[id] = struct{}{}
	}
	byID := make(map[string]workflow.TaskState, len(existing))
	for _, task := range existing {
		task.ID = canonicalTaskID(task.ID)
		if strings.HasPrefix(task.ID, "doctor.") || task.ID == rolloutTaskID {
			byID[task.ID] = task
			continue
		}
		prefix := strings.SplitN(task.ID, ".", 2)[0]
		if _, ok := selectedSet[prefix]; ok {
			byID[task.ID] = task
		}
	}
	if !selectionConfirmed {
		delete(byID, rolloutTaskID)
		result := make([]workflow.TaskState, 0, len(byID))
		for _, task := range byID {
			result = append(result, task)
		}
		return sortTasks(result)
	}

	allDockerBuildsComplete := len(componentIDs) > 0
	for _, componentID := range componentIDs {
		predecessorComplete := true
		for _, template := range componentTaskTemplates {
			taskID := fmt.Sprintf("%s.%s", componentID, template.Suffix)
			task := byID[taskID]
			if task.ID == "" {
				task = workflow.TaskState{ID: taskID, Title: fmt.Sprintf(template.Title, componentID), UpdatedAt: now.UTC()}
			}
			if task.Status == "" || !task.Status.Valid() {
				task.Status = workflow.StatusPending
			}
			switch {
			case isTerminalTaskStatus(task.Status), task.Status == workflow.StatusFailed, task.Status == workflow.StatusRunning:
				// Preserve explicit or in-flight states.
			case predecessorComplete:
				task.Status = template.Status
			case task.Status != workflow.StatusPending:
				task.Status = workflow.StatusPending
			}
			byID[taskID] = task
			predecessorComplete = isTerminalTaskStatus(task.Status)
		}
		if dockerBuild := byID[fmt.Sprintf("%s.docker-build", componentID)]; !isTerminalTaskStatus(dockerBuild.Status) {
			allDockerBuildsComplete = false
		}
	}

	rolloutTask := byID[rolloutTaskID]
	if rolloutTask.ID == "" {
		rolloutTask = workflow.TaskState{ID: rolloutTaskID, Title: rolloutTaskTemplate.Title, UpdatedAt: now.UTC()}
	}
	if rolloutTask.Status == "" || !rolloutTask.Status.Valid() {
		rolloutTask.Status = workflow.StatusPending
	}
	switch {
	case rolloutTask.Status == workflow.StatusFailed || rolloutTask.Status == workflow.StatusRunning || isTerminalTaskStatus(rolloutTask.Status):
		// Preserve explicit or in-flight states.
	case allDockerBuildsComplete:
		rolloutTask.Status = rolloutTaskTemplate.Status
	case rolloutTask.Status != workflow.StatusPending:
		rolloutTask.Status = workflow.StatusPending
	}
	byID[rolloutTaskID] = rolloutTask
	rolloutComplete := isTerminalTaskStatus(rolloutTask.Status)

	for _, componentID := range componentIDs {
		taskID := fmt.Sprintf("%s.%s", componentID, finalizeTaskTemplate.Suffix)
		task := byID[taskID]
		if task.ID == "" {
			task = workflow.TaskState{ID: taskID, Title: fmt.Sprintf(finalizeTaskTemplate.Title, componentID), UpdatedAt: now.UTC()}
		}
		if task.Status == "" || !task.Status.Valid() {
			task.Status = workflow.StatusPending
		}
		switch {
		case isTerminalTaskStatus(task.Status), task.Status == workflow.StatusFailed, task.Status == workflow.StatusRunning:
			// Preserve explicit or in-flight states.
		case rolloutComplete:
			task.Status = finalizeTaskTemplate.Status
		case task.Status != workflow.StatusPending:
			task.Status = workflow.StatusPending
		}
		byID[taskID] = task
	}

	result := make([]workflow.TaskState, 0, len(byID))
	for _, task := range byID {
		result = append(result, task)
	}
	return sortTasks(result)
}

func isTerminalTaskStatus(status workflow.Status) bool {
	switch status {
	case workflow.StatusCompleted, workflow.StatusSkipped, workflow.StatusExternallySatisfied:
		return true
	default:
		return false
	}
}

func sortTasks(tasks []workflow.TaskState) []workflow.TaskState {
	orderKey := func(taskID string) string {
		parts := strings.SplitN(taskID, ".", 2)
		if len(parts) != 2 {
			return "99:" + taskID
		}
		prefix, suffix := parts[0], parts[1]
		if prefix == "doctor" {
			doctorWeight := "99"
			switch suffix {
			case "git":
				doctorWeight = "00"
			case "gh-cli":
				doctorWeight = "01"
			case "git-fetch-tags-monorepo":
				doctorWeight = "02"
			case "git-fetch-tags-op-geth":
				doctorWeight = "03"
			case "monorepo-base-branch-synced":
				doctorWeight = "04"
			case "op-geth-base-branch-synced":
				doctorWeight = "05"
			case "release-manager-detected":
				doctorWeight = "06"
			case "repo-push-permissions":
				doctorWeight = "07"
			}
			return "00:" + doctorWeight + ":doctor"
		}
		weight := "99"
		switch suffix {
		case "review-diff":
			weight = "10"
		case "local-tag":
			weight = "11"
		case "push-tag":
			weight = "12"
		case "github-draft-release":
			weight = "13"
		case "docker-build":
			weight = "14"
		case "rollout":
			weight = "15"
		case "finalize-release":
			weight = "16"
		}
		if taskID == rolloutTaskID {
			return weight + ":zzzz:rollout"
		}
		return weight + ":" + prefix + ":" + suffix
	}
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if orderKey(tasks[j].ID) < orderKey(tasks[i].ID) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
	return tasks
}
