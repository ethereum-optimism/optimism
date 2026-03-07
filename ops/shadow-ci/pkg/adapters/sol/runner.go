package sol

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

// SolRunner implements adapters.TestRunner using forge test.
type SolRunner struct {
	root string // e.g., "packages/contracts-bedrock"
}

// NewRunner creates a new Solidity test runner.
func NewRunner(root string) *SolRunner {
	return &SolRunner{root: root}
}

func (r *SolRunner) Language() string { return "sol" }

func (r *SolRunner) Run(targets []model.Target, config model.Configuration, opts adapters.RunOptions) ([]model.TestResult, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	// Build match-path filter from targets.
	paths := make([]string, 0, len(targets))
	for _, t := range targets {
		paths = append(paths, t.ID)
	}
	matchPath := strings.Join(paths, "|")

	junitFile := filepath.Join(os.TempDir(), fmt.Sprintf("shadow-ci-sol-%d.xml", time.Now().UnixNano()))
	defer os.Remove(junitFile)

	args := []string{"test", "--match-path", matchPath, "--junit"}

	cmd := exec.Command("forge", args...)
	cmd.Dir = dir
	cmd.Env = buildSolEnv(config, opts)

	output, _ := cmd.CombinedOutput()

	// forge test --junit outputs junit XML to stdout.
	results, err := parseSolJunit(output)
	if err != nil {
		return nil, fmt.Errorf("parsing forge junit output: %w (output: %s)", err, truncate(string(output), 500))
	}

	for i := range results {
		results[i].Config = config.Name
	}

	return results, nil
}

func (r *SolRunner) RunOne(test model.TestIdentifier, config model.Configuration, opts adapters.RunOptions) (model.TestResult, error) {
	dir := opts.WorkDir
	if dir == "" {
		dir = r.root
	}

	args := []string{
		"test",
		"--match-test", fmt.Sprintf("^%s$", test.Name),
		"--match-path", test.Package,
		"--junit",
	}

	cmd := exec.Command("forge", args...)
	cmd.Dir = dir
	cmd.Env = buildSolEnv(config, opts)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	status := model.StatusPass
	if err != nil {
		status = model.StatusFail
	}

	return model.TestResult{
		Test:     test,
		Language: "sol",
		Config:   config.Name,
		Status:   status,
		Duration: duration,
		Output:   truncate(string(output), 4096),
	}, nil
}

// forgeJunitSuites mirrors forge's junit XML output structure.
type forgeJunitSuites struct {
	XMLName xml.Name          `xml:"testsuites"`
	Suites  []forgeJunitSuite `xml:"testsuite"`
}

type forgeJunitSuite struct {
	Name  string           `xml:"name,attr"`
	Tests int              `xml:"tests,attr"`
	Time  float64          `xml:"time,attr"`
	Cases []forgeJunitCase `xml:"testcase"`
}

type forgeJunitCase struct {
	Name      string             `xml:"name,attr"`
	ClassName string             `xml:"classname,attr"`
	Time      float64            `xml:"time,attr"`
	Failure   *forgeJunitFailure `xml:"failure,omitempty"`
}

type forgeJunitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func parseSolJunit(data []byte) ([]model.TestResult, error) {
	var suites forgeJunitSuites
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
				Language: "sol",
				Duration: time.Duration(tc.Time * float64(time.Second)),
				Status:   model.StatusPass,
			}
			if tc.Failure != nil {
				result.Status = model.StatusFail
				result.Output = truncate(tc.Failure.Body, 4096)
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func buildSolEnv(config model.Configuration, opts adapters.RunOptions) []string {
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}
	for k, v := range opts.ExtraEnv {
		env = append(env, k+"="+v)
	}
	return env
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated]"
}
