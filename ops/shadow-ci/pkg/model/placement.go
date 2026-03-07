package model

// PlacementConfig defines per-category stage placement.
type PlacementConfig struct {
	// Constraints are hard rules that the optimizer cannot override.
	Constraints []PlacementConstraint `yaml:"constraints" json:"constraints"`

	// Assignments are current stage placements (static defaults + optimizer output).
	Assignments map[string]CategoryPlacement `yaml:"assignments" json:"assignments"`

	// DefaultStage for categories without explicit assignments.
	DefaultStage Stage `yaml:"default_stage" json:"default_stage"`
}

// PlacementConstraint is a hard rule the optimizer must respect.
type PlacementConstraint struct {
	Category string `yaml:"category" json:"category"`
	// MinStage: category must run at this stage or earlier.
	MinStage Stage `yaml:"min_stage,omitempty" json:"min_stage,omitempty"`
	// MaxStage: category must not run earlier than this stage.
	MaxStage Stage `yaml:"max_stage,omitempty" json:"max_stage,omitempty"`
	// PinnedStage: category must run at exactly this stage.
	PinnedStage Stage `yaml:"pinned_stage,omitempty" json:"pinned_stage,omitempty"`
	Reason      string `yaml:"reason" json:"reason"`
}

// CategoryPlacement is the current stage assignment for a category.
type CategoryPlacement struct {
	Stage     Stage  `yaml:"stage" json:"stage"`
	Source    string `yaml:"source" json:"source"` // "default", "static", "optimizer"
	Reason    string `yaml:"reason,omitempty" json:"reason,omitempty"`
	UpdatedAt string `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// GetCategoryStage returns the effective stage for a category, respecting
// constraints (pinned overrides assignment) and falling back to defaults.
func (pc *PlacementConfig) GetCategoryStage(category string) Stage {
	// Pinned constraint overrides everything.
	for _, c := range pc.Constraints {
		if c.Category == category && c.PinnedStage != "" {
			return c.PinnedStage
		}
	}
	// Explicit assignment.
	if a, ok := pc.Assignments[category]; ok {
		return a.Stage
	}
	// Default.
	if pc.DefaultStage != "" {
		return pc.DefaultStage
	}
	return StagePR
}
