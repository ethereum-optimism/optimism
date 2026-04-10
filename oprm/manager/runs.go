package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

func NewRunID(now time.Time) string {
	return now.UTC().Format("20060102T150405Z")
}

func (a *App) CreateRun(ctx context.Context) (*release.Run, string, *DoctorReport, error) {
	report, err := a.Doctor(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	now := a.now().UTC()
	repo := fmt.Sprintf("%s/%s", a.Config.GitHub.Owner, a.Config.GitHub.Repo)
	run := release.NewRun(NewRunID(now), now, repo, a.Config.BaseBranch, report.ReleaseManager, a.Config.RunsDir)
	for _, task := range report.TaskStates(now) {
		run.UpsertTask(task)
	}
	run.AddTimeline(now, "initialized release run", fmt.Sprintf("release manager: %s", run.ReleaseManager.String()))
	if report.Blocking() {
		run.Status = release.RunStatusBlocked
		run.AddTimeline(now, "doctor detected blocking prerequisites")
	} else {
		run.AddTimeline(now, "doctor checks passed")
	}
	path, err := a.store.Save(run)
	if err != nil {
		return nil, "", nil, err
	}
	return run, path, report, nil
}

func (a *App) LoadRun(identifier string) (*release.Run, string, error) {
	run, path, err := a.store.Load(identifier)
	if err != nil {
		return nil, "", err
	}
	changed := normalizeRunTasks(run.Tasks)
	if a.autoReconcileReadyTasks(run, a.now()) {
		changed = true
	}
	if changed {
		path, err = a.store.Save(run)
		if err != nil {
			return nil, "", err
		}
	}
	return run, path, nil
}

func (a *App) RetryTask(identifier string, taskID string) (*release.Run, string, error) {
	return a.updateTask(identifier, taskID, workflow.StatusPending, "task reset for retry")
}

func (a *App) SkipTask(identifier string, taskID string, reason string) (*release.Run, string, error) {
	if reason == "" {
		return nil, "", fmt.Errorf("skip reason is required")
	}
	return a.updateTask(identifier, taskID, workflow.StatusSkipped, reason)
}

func (a *App) SatisfyTask(identifier string, taskID string, reason string) (*release.Run, string, error) {
	if reason == "" {
		return nil, "", fmt.Errorf("satisfy reason is required")
	}
	return a.updateTask(identifier, taskID, workflow.StatusExternallySatisfied, reason)
}

func (a *App) updateTask(identifier string, taskID string, status workflow.Status, reason string) (*release.Run, string, error) {
	run, path, err := a.store.Load(identifier)
	if err != nil {
		return nil, "", err
	}
	taskID = canonicalTaskID(taskID)
	task := run.FindTask(taskID)
	if task == nil {
		return nil, "", fmt.Errorf("task %q not found in run %q", taskID, run.RunID)
	}
	now := a.now().UTC()
	task.Status = status
	task.Reason = reason
	task.UpdatedAt = now
	run.AddTimeline(now, fmt.Sprintf("updated task %s -> %s", taskID, status), reason)
	run.Tasks = rebuildTasks(run.Tasks, run.Components, run.SelectionConfirmed, now)
	if _, err := a.store.Save(run); err != nil {
		return nil, "", err
	}
	return run, path, nil
}
