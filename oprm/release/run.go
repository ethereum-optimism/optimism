package release

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/workflow"
)

// RunStatus is the overall lifecycle state of a release run.
type RunStatus string

const (
	RunStatusInProgress RunStatus = "in_progress"
	RunStatusBlocked    RunStatus = "blocked"
	RunStatusCompleted  RunStatus = "completed"
)

// ReleaseManager identifies the current operator for a release run.
type ReleaseManager struct {
	GHLogin  string `yaml:"gh_login,omitempty"`
	GitName  string `yaml:"git_name,omitempty"`
	GitEmail string `yaml:"git_email,omitempty"`
}

func (m ReleaseManager) String() string {
	switch {
	case m.GHLogin != "" && m.GitName != "" && m.GitEmail != "":
		return fmt.Sprintf("%s / %s <%s>", m.GHLogin, m.GitName, m.GitEmail)
	case m.GitName != "" && m.GitEmail != "":
		return fmt.Sprintf("%s <%s>", m.GitName, m.GitEmail)
	case m.GHLogin != "":
		return m.GHLogin
	default:
		return "unknown"
	}
}

// ConfigSnapshot records the subset of config that materially affects a run.
type ConfigSnapshot struct {
	RunsDir string `yaml:"runs_dir"`
}

type ReviewInfo struct {
	FromRef         string   `yaml:"from_ref,omitempty"`
	ToRef           string   `yaml:"to_ref,omitempty"`
	ToRefKind       string   `yaml:"to_ref_kind,omitempty"`
	CompareURL      string   `yaml:"compare_url,omitempty"`
	CommitSummaries []string `yaml:"commit_summaries,omitempty"`
}

// VersionProposal captures persisted planning state for a selected component.
type VersionProposal struct {
	LatestRelease          string     `yaml:"latest_release,omitempty"`
	LatestRC               string     `yaml:"latest_rc,omitempty"`
	LatestDraftRC          string     `yaml:"latest_draft_rc,omitempty"`
	ComparedRef            string     `yaml:"compared_ref,omitempty"`
	Changed                bool       `yaml:"changed,omitempty"`
	ChangeEvidence         []string   `yaml:"change_evidence,omitempty"`
	Review                 ReviewInfo `yaml:"review,omitempty"`
	Bump                   string     `yaml:"bump,omitempty"`
	TargetRelease          string     `yaml:"target_release,omitempty"`
	Proposed               string     `yaml:"proposed,omitempty"`
	TargetTagRemoteState   string     `yaml:"target_tag_remote_state,omitempty"`
	ProposedTagRemoteState string     `yaml:"proposed_tag_remote_state,omitempty"`
	ResumeDraft            bool       `yaml:"resume_draft,omitempty"`
	ManualOverride         bool       `yaml:"manual_override,omitempty"`
}

// TimelineEntry records a human-readable audit log entry.
type TimelineEntry struct {
	At      time.Time `yaml:"at"`
	Summary string    `yaml:"summary"`
	Details []string  `yaml:"details,omitempty"`
}

// Run is the persisted state for a release workflow execution.
type Run struct {
	RunID              string                     `yaml:"run_id"`
	Status             RunStatus                  `yaml:"status"`
	CreatedAt          time.Time                  `yaml:"created_at"`
	UpdatedAt          time.Time                  `yaml:"updated_at"`
	Repo               string                     `yaml:"repo"`
	BaseBranch         string                     `yaml:"base_branch"`
	ReleaseManager     ReleaseManager             `yaml:"release_manager,omitempty"`
	Config             ConfigSnapshot             `yaml:"config"`
	Candidates         []string                   `yaml:"candidates,omitempty"`
	Components         []string                   `yaml:"components,omitempty"`
	SelectionConfirmed bool                       `yaml:"selection_confirmed,omitempty"`
	Versions           map[string]VersionProposal `yaml:"versions,omitempty"`
	Tasks              []workflow.TaskState       `yaml:"tasks,omitempty"`
	Timeline           []TimelineEntry            `yaml:"timeline,omitempty"`
}

func NewRun(id string, now time.Time, repo string, baseBranch string, mgr ReleaseManager, runsDir string) *Run {
	return &Run{
		RunID:          id,
		Status:         RunStatusInProgress,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
		Repo:           repo,
		BaseBranch:     baseBranch,
		ReleaseManager: mgr,
		Config: ConfigSnapshot{
			RunsDir: runsDir,
		},
		Versions: make(map[string]VersionProposal),
	}
}

func (r *Run) Touch(now time.Time) {
	r.UpdatedAt = now.UTC()
}

func (r *Run) AddTimeline(now time.Time, summary string, details ...string) {
	r.Timeline = append(r.Timeline, TimelineEntry{
		At:      now.UTC(),
		Summary: summary,
		Details: details,
	})
	r.Touch(now)
}

func (r *Run) UpsertTask(task workflow.TaskState) {
	for i := range r.Tasks {
		if r.Tasks[i].ID == task.ID {
			r.Tasks[i] = task
			return
		}
	}
	r.Tasks = append(r.Tasks, task)
}

func (r *Run) FindTask(id string) *workflow.TaskState {
	for i := range r.Tasks {
		if r.Tasks[i].ID == id {
			return &r.Tasks[i]
		}
	}
	return nil
}
