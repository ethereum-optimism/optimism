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
	{Suffix: "prepare-release-notes", Title: "Prepare release notes for %s", Status: workflow.StatusReady},
	{Suffix: "create-tag", Title: "Create local tag for %s", Status: workflow.StatusReady},
	{Suffix: "push-tag", Title: "Push tag for %s", Status: workflow.StatusReady},
	{Suffix: "github-draft-release", Title: "GitHub draft release for %s", Status: workflow.StatusReady},
	{Suffix: "docker-build", Title: "Confirm docker build readiness for %s", Status: workflow.StatusNeedsConfirmation},
	{Suffix: "rollout", Title: "Trigger rollout workflow for %s", Status: workflow.StatusNeedsConfirmation},
	{Suffix: "finalize-release", Title: "Finalize release for %s", Status: workflow.StatusReady},
}

var taskIDSuffixAliases = map[string]string{
	"create-or-update-draft-release": "github-draft-release",
	"manual-confirm-builds-ready":    "docker-build",
}

func canonicalTaskID(taskID string) string {
	componentID, suffix, ok := strings.Cut(taskID, ".")
	if !ok {
		return taskID
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
		if strings.HasPrefix(task.ID, "doctor.") {
			byID[task.ID] = task
			continue
		}
		prefix := strings.SplitN(task.ID, ".", 2)[0]
		if _, ok := selectedSet[prefix]; ok {
			byID[task.ID] = task
		}
	}
	if !selectionConfirmed {
		result := make([]workflow.TaskState, 0, len(byID))
		for _, task := range byID {
			result = append(result, task)
		}
		return sortTasks(result)
	}

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
			return taskID
		}
		suffixWeight := "0"
		switch parts[1] {
		case "review-diff":
			suffixWeight = "0"
		case "prepare-release-notes":
			suffixWeight = "1"
		case "create-tag":
			suffixWeight = "2"
		case "push-tag":
			suffixWeight = "3"
		case "github-draft-release":
			suffixWeight = "4"
		case "docker-build":
			suffixWeight = "5"
		case "rollout":
			suffixWeight = "6"
		case "finalize-release":
			suffixWeight = "7"
		}
		return parts[0] + ":" + suffixWeight + ":" + parts[1]
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
