package model

import "time"

// TestPlan is the core artifact. A platform-agnostic, language-agnostic
// description of what to test for a specific change.
type TestPlan struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Trigger      Trigger  `json:"trigger"`
	ChangedFiles []string `json:"changed_files"`

	// What to test — computed by the AffectedComputer.
	Jobs []PlannedJob `json:"jobs"`

	// Dependencies between jobs (job name → depends on).
	Dependencies map[string][]string `json:"dependencies,omitempty"`

	Summary PlanSummary `json:"summary"`
}

// Trigger describes what caused this plan to be created.
type Trigger struct {
	Type   string `json:"type"` // "pr", "push", "merge_queue", "nightly", "manual"
	PR     int    `json:"pr,omitempty"`
	Branch string `json:"branch"`
	Base   string `json:"base"`
	Head   string `json:"head"`
}

// PlannedJob describes a single test execution job.
type PlannedJob struct {
	Name     string `json:"name"`
	Language string `json:"language"`

	Targets        []Target        `json:"targets"`
	Configurations []Configuration `json:"configurations"`

	Resources Resources `json:"resources"`

	// Why these targets were selected.
	SelectionReason string `json:"selection_reason"`
}

// Resources describes the compute requirements for a job.
type Resources struct {
	Parallelism int           `json:"parallelism"`
	Runner      string        `json:"runner"`
	Timeout     time.Duration `json:"timeout"`
}

// PlanSummary provides aggregate statistics for the plan.
type PlanSummary struct {
	ByLanguage map[string]LanguageSummary `json:"by_language"`
}

// LanguageSummary provides per-language statistics.
type LanguageSummary struct {
	SelectedTargets int           `json:"selected_targets"`
	TotalTargets    int           `json:"total_targets"`
	SkipRate        float64       `json:"skip_rate"`
	AlwaysRunCount  int           `json:"always_run_count"`
	Configurations  int           `json:"configurations"`
	EstimatedTime   time.Duration `json:"estimated_time"`
}
