package model

// PipelineDecision is the shadow CI's decision about what the optimal pipeline looks like.
// Every job category in the CI maps to a CategoryDecision.
type PipelineDecision struct {
	ForceAll       bool                         `json:"force_all"`
	Branch         string                       `json:"branch"`
	Stage          Stage                        `json:"stage"`
	IsDevelop      bool                         `json:"is_develop"`
	IsSchedule     bool                         `json:"is_schedule"`
	Categories     map[string]*CategoryDecision `json:"categories"`
	RequiredBuilds []string                     `json:"required_builds,omitempty"` // build categories needed for selected tests
}

// CategoryDecision is the decision for a single job category.
type CategoryDecision struct {
	Needed         bool                   `json:"needed"`
	Reason         string                 `json:"reason"`
	Skipped        bool                   `json:"skipped"`
	SkipWhy        string                 `json:"skip_why,omitempty"`
	PlacedAt       Stage                  `json:"placed_at,omitempty"`     // stage this category is placed at
	StageSkipped   bool                   `json:"stage_skipped,omitempty"` // true if skipped due to stage placement
	Targets        []string               `json:"targets,omitempty"`       // affected targets (for graph-based categories)
	Packages       []string               `json:"packages,omitempty"`      // affected packages (for fuzz routing)
	Features       []string               `json:"features,omitempty"`      // needed features (for sol matrix)
	Configs        []string               `json:"configs,omitempty"`       // needed configs (for rust e2e)
	Tests          []TestPlacement        `json:"tests,omitempty"`         // per-test placements
	ShadowDeferral *ShadowDeferralSummary `json:"shadow_deferral,omitempty"`
}

// TestPlacement is a per-test stage assignment.
type TestPlacement struct {
	TestKey        string  `json:"test_key"`                      // TestIdentifier.Key()
	AssignedStage  Stage   `json:"assigned_stage"`                // which stage this test should run at
	Reason         string  `json:"reason"`                        // human-readable explanation
	Confidence     float64 `json:"confidence"`                    // 0-1
	WouldDefer     bool    `json:"would_defer"`                   // shadow deferral flag
	DeferTo        Stage   `json:"defer_to,omitempty"`            // stage it would be deferred to
	CorrelatedWith string  `json:"correlated_with,omitempty"`     // faster correlated test
}

// ShadowDeferralSummary summarizes what would be deferred for a category.
type ShadowDeferralSummary struct {
	WouldDefer int     `json:"would_defer"`  // count of tests that would be deferred
	EstSavings int64   `json:"est_savings"`  // estimated time saved in milliseconds
	MissRisk   float64 `json:"miss_risk"`    // estimated probability of missing a failure
}

// JobCategoryConfig defines a CI job category in scoping.yaml.
type JobCategoryConfig struct {
	TriggerPaths    []string         `yaml:"trigger_paths" json:"trigger_paths"`
	UseGraph        bool             `yaml:"use_graph" json:"use_graph"`
	Language        string           `yaml:"language" json:"language"`
	AlwaysOnDevelop bool             `yaml:"always_on_develop" json:"always_on_develop"`
	DevelopOnly     bool             `yaml:"develop_only" json:"develop_only"`
	ScheduleOnly    bool             `yaml:"schedule_only" json:"schedule_only"`
	TagOnly         bool             `yaml:"tag_only" json:"tag_only"`
	Always          bool             `yaml:"always" json:"always"`
	FeatureMatrix   []string         `yaml:"feature_matrix" json:"feature_matrix,omitempty"`
	FuzzPackages    []FuzzPackage    `yaml:"fuzz_packages" json:"fuzz_packages,omitempty"`
	Checks          []string         `yaml:"checks" json:"checks,omitempty"`
	Configs         []string         `yaml:"configs" json:"configs,omitempty"`
	Description     string           `yaml:"description" json:"description"`

	// Execution group — determines which CI job runs this category.
	// Values: "build", "go", "sol", "rust", "misc". Categories with no group
	// are decision-only (shadow CI decides run/skip but doesn't execute).
	Group string `yaml:"group" json:"group,omitempty"`

	// Dependency and build fields for dynamic pipeline generation.
	DependsOn      []string `yaml:"depends_on" json:"depends_on,omitempty"`       // categories this one requires
	Command        string   `yaml:"command" json:"command,omitempty"`             // full command (used when targets empty or force-all)
	TargetCommand  string   `yaml:"target_command" json:"target_command,omitempty"` // targeted command; {{targets}} replaced with affected targets
	WorkspacePaths []string `yaml:"workspace_paths" json:"workspace_paths,omitempty"` // paths to persist in workspace
	RunnerClass    string   `yaml:"runner" json:"runner,omitempty"`               // resource class override

	// Content-addressed build cache fields.
	CacheInputs []string `yaml:"cache_inputs" json:"cache_inputs,omitempty"` // paths hashed for cache key (defaults to trigger_paths)
}

// FuzzPackage maps a fuzz package to its trigger paths.
type FuzzPackage struct {
	Package        string   `yaml:"package" json:"package"`
	TriggerPaths   []string `yaml:"trigger_paths" json:"trigger_paths"`
	NeedsArtifacts bool     `yaml:"needs_artifacts" json:"needs_artifacts"`
}
