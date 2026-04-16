package coverage

import (
	"encoding/json"
	"fmt"
	"os"
)

// Report is the language-agnostic coverage output format.
// One report per (test unit, profile) pair.
type Report struct {
	Test     string              `json:"test"`              // test identifier (file path, package, function)
	Language string              `json:"language"`           // "solidity", "go", "rust"
	Profile  string              `json:"profile,omitempty"` // e.g. "main", "custom_gas_token". Empty = default.
	Covers   map[string][][2]int `json:"covers"`             // source file → list of [startLine, endLine] ranges
}

// LoadReport reads a coverage report from a JSON file.
func LoadReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading coverage report: %w", err)
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing coverage report: %w", err)
	}
	return &r, nil
}

// LoadReports reads all coverage reports from a directory.
func LoadReports(dir string) ([]*Report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading coverage directory: %w", err)
	}
	var reports []*Report
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := dir + "/" + e.Name()
		r, err := LoadReport(path)
		if err != nil {
			continue // skip invalid files
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// SaveReport writes a coverage report to a JSON file.
func SaveReport(r *Report, path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Profile is a named configuration for running tests.
// Different profiles can produce different coverage for the same test
// (feature flags gate different code paths).
type Profile struct {
	Name string            // e.g. "main", "custom_gas_token"
	Env  map[string]string // environment variables to set
}

// Collector produces coverage reports for a given language.
type Collector interface {
	// Language returns the language this collector handles.
	Language() string

	// Collect runs coverage for a single test unit under a profile.
	// An empty Profile{} means default (no extra env vars).
	Collect(rootDir string, testPath string, profile Profile) (*Report, error)
}
