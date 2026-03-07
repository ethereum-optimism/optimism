package model

// PipelineDecision is the shadow CI's decision about what the optimal pipeline looks like.
// Every job category in the CI maps to a CategoryDecision.
type PipelineDecision struct {
	ForceAll   bool                         `json:"force_all"`
	Branch     string                       `json:"branch"`
	Stage      Stage                        `json:"stage"`
	IsDevelop  bool                         `json:"is_develop"`
	IsSchedule bool                         `json:"is_schedule"`
	Categories map[string]*CategoryDecision `json:"categories"`
}

// CategoryDecision is the decision for a single job category.
type CategoryDecision struct {
	Needed       bool     `json:"needed"`
	Reason       string   `json:"reason"`
	Skipped      bool     `json:"skipped"`
	SkipWhy      string   `json:"skip_why,omitempty"`
	PlacedAt     Stage    `json:"placed_at,omitempty"`     // stage this category is placed at
	StageSkipped bool     `json:"stage_skipped,omitempty"` // true if skipped due to stage placement
	Targets      []string `json:"targets,omitempty"`       // affected targets (for graph-based categories)
	Packages     []string `json:"packages,omitempty"`      // affected packages (for fuzz routing)
	Features     []string `json:"features,omitempty"`      // needed features (for sol matrix)
	Configs      []string `json:"configs,omitempty"`       // needed configs (for rust e2e)
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
}

// FuzzPackage maps a fuzz package to its trigger paths.
type FuzzPackage struct {
	Package        string   `yaml:"package" json:"package"`
	TriggerPaths   []string `yaml:"trigger_paths" json:"trigger_paths"`
	NeedsArtifacts bool     `yaml:"needs_artifacts" json:"needs_artifacts"`
}
