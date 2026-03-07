package model

// Config is the top-level configuration loaded from YAML files.
type Config struct {
	Adapters  AdaptersConfig  `yaml:"adapters" json:"adapters"`
	Scoping   ScopingConfig   `yaml:"scoping" json:"scoping"`
	Platform  PlatformConfig  `yaml:"platform" json:"platform"`
	Placement PlacementConfig `yaml:"placement" json:"placement"`
}

// AdaptersConfig configures language adapters.
type AdaptersConfig struct {
	Go   *LanguageAdapterConfig `yaml:"go,omitempty" json:"go,omitempty"`
	Sol  *SolAdapterConfig      `yaml:"sol,omitempty" json:"sol,omitempty"`
	Rust *LanguageAdapterConfig `yaml:"rust,omitempty" json:"rust,omitempty"`
}

// LanguageAdapterConfig is the base config for a language adapter.
type LanguageAdapterConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	Root          string   `yaml:"root" json:"root"`
	SpecialPaths  []string `yaml:"special_paths" json:"special_paths"`
	CacheKeyFiles []string `yaml:"cache_key_files" json:"cache_key_files"`
}

// SolAdapterConfig extends LanguageAdapterConfig with Solidity-specific config.
type SolAdapterConfig struct {
	LanguageAdapterConfig `yaml:",inline"`
	SourceDirs            []string       `yaml:"source_dirs" json:"source_dirs"`
	RemappingsFile        string         `yaml:"remappings_file" json:"remappings_file"`
	Features              []FeatureRule  `yaml:"features" json:"features"`
}

// FeatureRule defines a Solidity feature matrix entry.
type FeatureRule struct {
	Name         string            `yaml:"name" json:"name"`
	Env          map[string]string `yaml:"env" json:"env"`
	Always       bool              `yaml:"always" json:"always"`
	TriggerPaths []string          `yaml:"trigger_paths" json:"trigger_paths"`
}

// ScopingConfig controls target scoping and confidence.
type ScopingConfig struct {
	ConfidenceThreshold      float64                          `yaml:"confidence_threshold" json:"confidence_threshold"`
	AlwaysRunGraduationWeeks int                              `yaml:"always_run_graduation_weeks" json:"always_run_graduation_weeks"`
	ForceAllPaths            []string                         `yaml:"force_all_paths" json:"force_all_paths"`
	AlwaysRun                map[string][]string               `yaml:"always_run" json:"always_run"`
	JobCategories            map[string]JobCategoryConfig      `yaml:"job_categories" json:"job_categories"`
	AcceptanceGates          map[string]AcceptanceGate         `yaml:"acceptance_gates" json:"acceptance_gates"`
	Activation               ActivationConfig                  `yaml:"activation" json:"activation"`
}

// AcceptanceGate defines an acceptance test gate.
type AcceptanceGate struct {
	TriggerPaths []string `yaml:"trigger_paths" json:"trigger_paths"`
	Always       bool     `yaml:"always" json:"always"`
}

// ActivationConfig controls what's enabled.
type ActivationConfig struct {
	Mode      string            `yaml:"mode" json:"mode"` // "shadow", "belt-and-suspenders", "primary"
	Languages map[string]bool   `yaml:"languages" json:"languages"`
	Agents    map[string]bool   `yaml:"agents" json:"agents"`
	Comparison ComparisonConfig `yaml:"comparison" json:"comparison"`
}

// ComparisonConfig controls comparison behavior.
type ComparisonConfig struct {
	Required bool `yaml:"required" json:"required"`
}

// PlatformConfig configures the CI platform.
type PlatformConfig struct {
	Platform string          `yaml:"platform" json:"platform"`
	CircleCI CircleCIConfig  `yaml:"circleci" json:"circleci"`
	Events   EventStoreConfig `yaml:"events" json:"events"`
}

// CircleCIConfig configures CircleCI-specific settings.
type CircleCIConfig struct {
	Runners         map[string]string `yaml:"runners" json:"runners"`
	MainCI          MainCIConfig      `yaml:"main_ci" json:"main_ci"`
	MaxConcurrency  map[string]int    `yaml:"max_concurrency" json:"max_concurrency"`
}

// MainCIConfig identifies the main CI pipeline for comparison.
type MainCIConfig struct {
	Workflows        []string          `yaml:"workflows" json:"workflows"`
	ArtifactPatterns map[string]string `yaml:"artifact_patterns" json:"artifact_patterns"`
}

// EventStoreConfig configures the event store.
type EventStoreConfig struct {
	Store string    `yaml:"store" json:"store"` // "local", "gcs"
	Local LocalConfig `yaml:"local,omitempty" json:"local,omitempty"`
	GCS   GCSConfig   `yaml:"gcs,omitempty" json:"gcs,omitempty"`
}

// LocalConfig configures local file event storage.
type LocalConfig struct {
	Dir string `yaml:"dir" json:"dir"`
}

// GCSConfig configures GCS event storage.
type GCSConfig struct {
	Bucket  string `yaml:"bucket" json:"bucket"`
	Project string `yaml:"project" json:"project"`
}
