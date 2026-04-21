package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CheckType describes a runnable pipeline step.
//
// Kinds ("test", "lint", "build", "check", "gen") are advisory — they
// bias policy priors and display, but the selector and executor treat
// every kind the same: if inputs/outputs say you're affected, you
// run, in dataflow-derived prereq order. "gen" is kept as a distinct
// kind because regenerators (mise-setup, gen-solidity-interfaces,
// gen-go-bindings) mutate the working tree; tooling can surface that
// distinction without it changing execution.
//
// Inputs and Outputs accept either path globs
// ("packages/contracts-bedrock/src/**/*.sol") or artifact refs
// ("artifact:forge-artifacts/**"). Tools is sugar: each entry expands
// into inputs: [artifact:toolchain/<tool>]. Selection and prereq
// ordering are both derived from the inputs/outputs dataflow chain.
type CheckType struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Kind            string   `yaml:"kind"`
	Language        string   `yaml:"language"`
	Command         string   `yaml:"command"`
	Scopeable       bool     `yaml:"scopeable"`
	ScopeFlag       string   `yaml:"scope_flag"`
	ScopeType       string   `yaml:"scope_type"`
	Inputs          []string `yaml:"inputs,omitempty"`
	Outputs         []string `yaml:"outputs,omitempty"`
	Tools           []string `yaml:"tools,omitempty"`
	Knobs           []Knob   `yaml:"knobs,omitempty"`
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
// feature flag combinations. Each profile sets a specific set of
// environment variables before running the test command.
//
// ActiveWhenSelected lists check IDs whose selection activates this
// profile. profileTriggerExpand clones each activating check's main-
// profile candidates under this profile.
type TestProfile struct {
	Name               string            `yaml:"name"`
	Env                map[string]string `yaml:"env,omitempty"`
	ActiveWhenSelected []string          `yaml:"active_when_selected,omitempty"`
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

// Validate checks for unique IDs, non-empty commands, and knob bounds.
// Dataflow-derived validation (cycle detection) is handled post-build
// by the graph package, not here, because Validate runs before the
// graph exists.
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
	for _, p := range c.Profiles {
		for _, id := range p.ActiveWhenSelected {
			if !seen[id] {
				return fmt.Errorf("profile %q references unknown check %q in active_when_selected", p.Name, id)
			}
		}
	}
	return nil
}

// ByID returns a check type by ID, or nil if not found.
func (c *Catalog) ByID(id string) *CheckType {
	return c.byID[id]
}

func (c *Catalog) buildIndex() {
	c.byID = make(map[string]*CheckType, len(c.CheckTypes))
	for i := range c.CheckTypes {
		c.byID[c.CheckTypes[i].ID] = &c.CheckTypes[i]
	}
}
