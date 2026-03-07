package rust

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// RustRunner implements adapters.TestRunner using cargo nextest.
type RustRunner struct {
	root string // e.g., "rust"
}

// NewRunner creates a new Rust test runner.
func NewRunner(root string) *RustRunner {
	return &RustRunner{root: root}
}

func (r *RustRunner) Language() string { return "rust" }

func (r *RustRunner) Run(targets []model.Target, config model.Configuration, opts adapters.RunOptions) ([]model.TestResult, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	junitFile := filepath.Join(os.TempDir(), fmt.Sprintf("shadow-ci-rust-%d.xml", time.Now().UnixNano()))
	defer os.Remove(junitFile)

	args := []string{"nextest", "run"}
	for _, t := range targets {
		args = append(args, "--package", t.ID)
	}
	args = append(args, "--message-format", "libtest-json")

	// Use junit output for structured results.
	args = append(args, "--", "--format=junit")

	cmd := exec.Command("cargo", args...)
	cmd.Dir = dir
	cmd.Env = buildRustEnv(config, opts)

	output, _ := cmd.CombinedOutput()

	// Try to parse as junit XML from nextest.
	results, err := parseNextestJunit(output)
	if err != nil {
		// Fallback: treat the entire run as a single result.
		status := model.StatusPass
		if strings.Contains(string(output), "FAILED") || strings.Contains(string(output), "error") {
			status = model.StatusFail
		}
		return []model.TestResult{{
			Test:     model.TestIdentifier{Name: "all", Package: "rust"},
			Language: "rust",
			Config:   config.Name,
			Status:   status,
			Output:   truncateStr(string(output), 4096),
		}}, nil
	}

	for i := range results {
		results[i].Config = config.Name
	}
	return results, nil
}

func (r *RustRunner) RunOne(test model.TestIdentifier, config model.Configuration, opts adapters.RunOptions) (model.TestResult, error) {
	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	args := []string{
		"nextest", "run",
		"--package", test.Package,
		"-E", fmt.Sprintf("test(%s)", test.Name),
	}

	cmd := exec.Command("cargo", args...)
	cmd.Dir = dir
	cmd.Env = buildRustEnv(config, opts)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	status := model.StatusPass
	if err != nil {
		status = model.StatusFail
	}

	return model.TestResult{
		Test:     test,
		Language: "rust",
		Config:   config.Name,
		Status:   status,
		Duration: duration,
		Output:   truncateStr(string(output), 4096),
	}, nil
}

type nextestJunitSuites struct {
	XMLName xml.Name            `xml:"testsuites"`
	Suites  []nextestJunitSuite `xml:"testsuite"`
}

type nextestJunitSuite struct {
	Name  string              `xml:"name,attr"`
	Cases []nextestJunitCase  `xml:"testcase"`
}

type nextestJunitCase struct {
	Name      string               `xml:"name,attr"`
	ClassName string               `xml:"classname,attr"`
	Time      float64              `xml:"time,attr"`
	Failure   *nextestJunitFailure `xml:"failure,omitempty"`
}

type nextestJunitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func parseNextestJunit(data []byte) ([]model.TestResult, error) {
	var suites nextestJunitSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		return nil, err
	}

	var results []model.TestResult
	for _, suite := range suites.Suites {
		for _, tc := range suite.Cases {
			result := model.TestResult{
				Test: model.TestIdentifier{
					Name:    tc.Name,
					Package: tc.ClassName,
				},
				Language: "rust",
				Duration: time.Duration(tc.Time * float64(time.Second)),
				Status:   model.StatusPass,
			}
			if tc.Failure != nil {
				result.Status = model.StatusFail
				result.Output = truncateStr(tc.Failure.Body, 4096)
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func buildRustEnv(config model.Configuration, opts adapters.RunOptions) []string {
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}
	for k, v := range opts.ExtraEnv {
		env = append(env, k+"="+v)
	}
	return env
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated]"
}
