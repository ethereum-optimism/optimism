package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CheckType describes a runnable check and its available configuration knobs.
type CheckType struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Kind            string   `yaml:"kind"`              // "test", "lint", "build", "check"
	Language        string   `yaml:"language"`           // "go", "solidity", "rust", "*"
	Command         string   `yaml:"command"`            // base command
	Scopeable       bool     `yaml:"scopeable"`          // can this check be scoped to a subset?
	ScopeFlag       string   `yaml:"scope_flag"`         // CLI flag for scope ("--match-path", "" for positional)
	ScopeType       string   `yaml:"scope_type"`         // "packages", "paths", "tests"
	Triggers        []string `yaml:"triggers,omitempty"` // file patterns that trigger this (for non-scopeable checks)
	// Produces lists file globs whose freshness this check is responsible
	// for. The builder emits `produces` edges from check:<id> to every
	// matching source node. When the selector's reverse walk passes
	// through one of those nodes, this check becomes a per-file
	// prerequisite of the downstream consumer Candidates — separate from
	// the coarse catalog-level Prerequisites.
	Produces        []string `yaml:"produces,omitempty"`
	// Inputs / Outputs / Tools: pipeline-model fields (see
	// ops/checks/docs/pipeline-model.md). Optional during Phase A of
	// the migration — when set, the builder emits consumes/produces
	// edges to/from source and artifact nodes. Replace Triggers /
	// Prerequisites / Produces during Phase B migration.
	//
	// Inputs and Outputs accept either path globs
	// ("packages/contracts-bedrock/src/**/*.sol") or artifact refs
	// ("artifact:forge-artifacts/**"). Tools is sugar: each entry
	// expands into inputs: [artifact:toolchain/<tool>].
	Inputs  []string `yaml:"inputs,omitempty"`
	Outputs []string `yaml:"outputs,omitempty"`
	Tools   []string `yaml:"tools,omitempty"`
	Knobs           []Knob   `yaml:"knobs,omitempty"`
	Prerequisites   []string `yaml:"prerequisites,omitempty"`
	AvgDuration     int      `yaml:"avg_duration"`
	PerUnitDuration int      `yaml:"per_unit_duration,omitempty"`

	// CIJobNames maps this check to one or more CircleCI job names.
	// Profile-variant jobs (e.g. forge-test under custom_gas_token) all
	// aggregate into this check's CI-history outcomes unless split
	// into a dedicated check type. Empty means no CI-history mapping.
	CIJobNames []string `yaml:"ci_job_names,omitempty"`
}

// Knob is a configurable parameter for a check type.
type Knob struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`              // "int", "bool", "string", "enum"
	Flag    string   `yaml:"flag,omitempty"`    // CLI flag ("--fuzz-runs", "-short")
	EnvVar  string   `yaml:"env_var,omitempty"` // env var alternative
	Default any      `yaml:"default"`
	Min     int      `yaml:"min,omitempty"`
	Max     int      `yaml:"max,omitempty"`
	Choices []string `yaml:"choices,omitempty"`
}

// TestProfile is a named configuration for running tests under different
// feature flag combinations. Each profile sets a specific set of environment
// variables before running the test command.
//
// Triggers let a profile force itself into the candidate set when the
// diff touches relevant paths. Covers the case where coverage data
// wasn't collected under this profile (so no coverage edge carries the
// profile tag) but the profile still needs to run — e.g. feature-flag
// profiles that only activate specific code paths.
type TestProfile struct {
	Name     string            `yaml:"name"`
	Env      map[string]string `yaml:"env,omitempty"`
	Triggers []string          `yaml:"triggers,omitempty"`
}

// MatchesTriggers reports whether any of the given file paths match
// this profile's triggers. Mirrors CheckType.MatchesTriggers.
func (p *TestProfile) MatchesTriggers(filePaths []string) bool {
	if len(p.Triggers) == 0 {
		return false
	}
	for _, f := range filePaths {
		for _, pattern := range p.Triggers {
			if matched, _ := filepath.Match(pattern, f); matched {
				return true
			}
			if strings.HasSuffix(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(f, prefix) {
					return true
				}
			}
		}
	}
	return false
}

// Catalog is the top-level manifest of available check types.
type Catalog struct {
	CheckTypes []CheckType   `yaml:"check_types"`
	Profiles   []TestProfile `yaml:"profiles,omitempty"`
	// UniversalInputs is wired as consumes edges from every check_type
	// at build time. Diffs touching these paths fan out to every check
	// via dataflow, replacing the former policy.blast_radius_patterns.
	UniversalInputs []string `yaml:"universal_inputs,omitempty"`
	byID            map[string]*CheckType
}

// ProfileByName returns a profile by name, or nil if not found.
func (c *Catalog) ProfileByName(name string) *TestProfile {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i]
		}
	}
	return nil
}

// Load reads a catalog from a YAML file.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses catalog YAML data.
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing catalog: %w", err)
	}
	c.buildIndex()
	return &c, nil
}

// Validate checks for unique IDs, valid prerequisites, non-empty commands, and knob bounds.
func (c *Catalog) Validate() error {
	seen := make(map[string]bool)
	for _, ct := range c.CheckTypes {
		if ct.ID == "" {
			return fmt.Errorf("check type has empty ID")
		}
		if seen[ct.ID] {
			return fmt.Errorf("duplicate check type ID: %q", ct.ID)
		}
		seen[ct.ID] = true
		if ct.Command == "" {
			return fmt.Errorf("check type %q has empty command", ct.ID)
		}
		for _, k := range ct.Knobs {
			if k.Type == "int" && k.Min > k.Max && k.Max != 0 {
				return fmt.Errorf("check type %q knob %q: min (%d) > max (%d)", ct.ID, k.Name, k.Min, k.Max)
			}
			if k.Type == "enum" && len(k.Choices) == 0 {
				return fmt.Errorf("check type %q knob %q: enum type requires choices", ct.ID, k.Name)
			}
		}
	}
	for _, ct := range c.CheckTypes {
		for _, prereq := range ct.Prerequisites {
			if !seen[prereq] {
				return fmt.Errorf("check type %q has prerequisite %q which does not exist", ct.ID, prereq)
			}
		}
	}
	return nil
}

// ByID returns a check type by ID, or nil if not found.
func (c *Catalog) ByID(id string) *CheckType {
	return c.byID[id]
}

// MatchesTriggers returns true if any of the given file paths match this check type's triggers.
func (ct *CheckType) MatchesTriggers(filePaths []string) bool {
	if len(ct.Triggers) == 0 {
		return false
	}
	for _, f := range filePaths {
		for _, pattern := range ct.Triggers {
			if matched, _ := filepath.Match(pattern, f); matched {
				return true
			}
			// Also try matching with ** prefix stripped for directory globs
			if strings.HasSuffix(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(f, prefix) {
					return true
				}
			}
		}
	}
	return false
}

func (c *Catalog) buildIndex() {
	c.byID = make(map[string]*CheckType, len(c.CheckTypes))
	for i := range c.CheckTypes {
		c.byID[c.CheckTypes[i].ID] = &c.CheckTypes[i]
	}
}
