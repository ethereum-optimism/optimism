package workflow

import "time"

// Status describes the lifecycle state of a task in a release run.
type Status string

const (
	StatusPending             Status = "pending"
	StatusReady               Status = "ready"
	StatusBlocked             Status = "blocked"
	StatusNeedsConfirmation   Status = "needs-confirmation"
	StatusRunning             Status = "running"
	StatusCompleted           Status = "completed"
	StatusSkipped             Status = "skipped"
	StatusFailed              Status = "failed"
	StatusExternallySatisfied Status = "externally-satisfied"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending,
		StatusReady,
		StatusBlocked,
		StatusNeedsConfirmation,
		StatusRunning,
		StatusCompleted,
		StatusSkipped,
		StatusFailed,
		StatusExternallySatisfied:
		return true
	default:
		return false
	}
}

// TaskState records the persisted state of a task inside a release run journal.
type TaskState struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title,omitempty"`
	Status    Status    `yaml:"status"`
	UpdatedAt time.Time `yaml:"updated_at,omitempty"`
	Reason    string    `yaml:"reason,omitempty"`
}
