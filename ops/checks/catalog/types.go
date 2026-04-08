package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Check represents a single runnable check.
type Check struct {
	ID            string   `yaml:"id"`
	Name          string   `yaml:"name"`
	Kind          string   `yaml:"kind"`
	Language      string   `yaml:"language"`
	Command       string   `yaml:"command"`
	AvgDuration   int      `yaml:"avg_duration"`
	Packages      []string `yaml:"packages,omitempty"`
	Directories   []string `yaml:"directories,omitempty"`
	FilePatterns  []string `yaml:"file_patterns,omitempty"`
	Prerequisites []string `yaml:"prerequisites,omitempty"`
	Tags          []string `yaml:"tags,omitempty"`
}

// Catalog is the top-level checks manifest.
type Catalog struct {
	Checks []Check `yaml:"checks"`

	byID map[string]*Check
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

// Validate checks for: unique IDs, valid prerequisite references, non-empty commands.
func (c *Catalog) Validate() error {
	seen := make(map[string]bool)
	for _, ch := range c.Checks {
		if ch.ID == "" {
			return fmt.Errorf("check has empty ID")
		}
		if seen[ch.ID] {
			return fmt.Errorf("duplicate check ID: %q", ch.ID)
		}
		seen[ch.ID] = true
		if ch.Command == "" {
			return fmt.Errorf("check %q has empty command", ch.ID)
		}
	}
	// Validate prerequisites reference existing IDs
	for _, ch := range c.Checks {
		for _, prereq := range ch.Prerequisites {
			if !seen[prereq] {
				return fmt.Errorf("check %q has prerequisite %q which does not exist", ch.ID, prereq)
			}
		}
	}
	return nil
}

// ByID returns a check by ID, or nil if not found.
func (c *Catalog) ByID(id string) *Check {
	return c.byID[id]
}

// ByLanguage returns checks matching the given language.
func (c *Catalog) ByLanguage(lang string) []Check {
	var result []Check
	for _, ch := range c.Checks {
		if ch.Language == lang {
			result = append(result, ch)
		}
	}
	return result
}

func (c *Catalog) buildIndex() {
	c.byID = make(map[string]*Check, len(c.Checks))
	for i := range c.Checks {
		c.byID[c.Checks[i].ID] = &c.Checks[i]
	}
}
