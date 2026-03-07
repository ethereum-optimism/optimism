package golang

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

// GoRunner implements adapters.TestRunner using gotestsum.
type GoRunner struct {
	root string
}

// NewRunner creates a new Go test runner.
func NewRunner(root string) *GoRunner {
	return &GoRunner{root: root}
}

func (r *GoRunner) Language() string { return "go" }

// junitTestSuites is the top-level element of gotestsum junit output.
type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Time     float64         `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func (r *GoRunner) Run(targets []model.Target, config model.Configuration, opts adapters.RunOptions) ([]model.TestResult, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	junitFile := filepath.Join(os.TempDir(), fmt.Sprintf("shadow-ci-go-%d.xml", time.Now().UnixNano()))
	defer os.Remove(junitFile)

	pkgs := make([]string, 0, len(targets))
	for _, t := range targets {
		pkgs = append(pkgs, t.ID+"/...")
	}

	args := []string{
		"--format=testname",
		"--junitfile=" + junitFile,
		"--",
		"-count=1",
	}
	if opts.Timeout > 0 {
		args = append(args, fmt.Sprintf("-timeout=%ds", opts.Timeout))
	}
	args = append(args, pkgs...)

	cmd := exec.Command("gotestsum", args...)
	cmd.Dir = dir
	cmd.Env = buildEnv(config, opts)

	output, runErr := cmd.CombinedOutput()

	// Parse junit XML even if the test run "failed" (test failures are expected).
	results, parseErr := parseJunitFile(junitFile, string(output))
	if parseErr != nil {
		if runErr != nil {
			return nil, fmt.Errorf("test run failed and could not parse results: run=%w, parse=%v", runErr, parseErr)
		}
		return nil, fmt.Errorf("parsing junit: %w", parseErr)
	}

	return results, nil
}

func (r *GoRunner) RunOne(test model.TestIdentifier, config model.Configuration, opts adapters.RunOptions) (model.TestResult, error) {
	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	args := []string{
		"test",
		"-run", fmt.Sprintf("^%s$", test.Name),
		"-count=1",
		"-timeout=60s",
		"-v",
		test.Package + "/...",
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = buildEnv(config, opts)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	status := model.StatusPass
	if err != nil {
		status = model.StatusFail
	}

	return model.TestResult{
		Test:     test,
		Language: "go",
		Config:   config.Name,
		Status:   status,
		Duration: duration,
		Output:   truncateOutput(string(output), 4096),
	}, nil
}

func parseJunitFile(path, rawOutput string) ([]model.TestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading junit file: %w", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		return nil, fmt.Errorf("unmarshaling junit xml: %w", err)
	}

	var results []model.TestResult
	for _, suite := range suites.Suites {
		for _, tc := range suite.Cases {
			result := model.TestResult{
				Test: model.TestIdentifier{
					Name:    tc.Name,
					Package: tc.ClassName,
				},
				Language: "go",
				Duration: time.Duration(tc.Time * float64(time.Second)),
			}

			switch {
			case tc.Error != nil:
				result.Status = model.StatusError
				result.Output = truncateOutput(tc.Error.Body, 4096)
			case tc.Failure != nil:
				result.Status = model.StatusFail
				result.Output = truncateOutput(tc.Failure.Body, 4096)
			case tc.Skipped != nil:
				result.Status = model.StatusSkip
			default:
				result.Status = model.StatusPass
			}

			results = append(results, result)
		}
	}

	return results, nil
}

func buildEnv(config model.Configuration, opts adapters.RunOptions) []string {
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}
	for k, v := range opts.ExtraEnv {
		env = append(env, k+"="+v)
	}
	return env
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated]"
}

// TargetIDsFromPackages converts Go import paths to package patterns.
func TargetIDsFromPackages(targets []model.Target) string {
	ids := make([]string, len(targets))
	for i, t := range targets {
		ids[i] = t.ID
	}
	return strings.Join(ids, " ")
}
