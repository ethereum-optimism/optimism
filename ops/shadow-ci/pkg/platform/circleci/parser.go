package circleci

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// ParseGotestsumJSON parses gotestsum JSON output into TestResults.
// Each line is a JSON event from `gotestsum --jsonfile`.
func ParseGotestsumJSON(data []byte) ([]model.TestResult, error) {
	var results []model.TestResult
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event struct {
			Action  string  `json:"Action"`
			Test    string  `json:"Test"`
			Package string  `json:"Package"`
			Elapsed float64 `json:"Elapsed"`
			Output  string  `json:"Output"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // skip unparseable lines
		}

		// Only care about pass/fail events with a test name.
		if event.Test == "" {
			continue
		}
		if event.Action != "pass" && event.Action != "fail" && event.Action != "skip" {
			continue
		}

		status := model.StatusPass
		switch event.Action {
		case "fail":
			status = model.StatusFail
		case "skip":
			status = model.StatusSkip
		}

		results = append(results, model.TestResult{
			Test: model.TestIdentifier{
				Package: event.Package,
				Name:    event.Test,
			},
			Language: "go",
			Status:   status,
			Duration: time.Duration(event.Elapsed * float64(time.Second)),
		})
	}

	return results, nil
}

// junitTestSuites is the root element of JUnit XML.
type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *struct{}     `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

// ParseJUnitXML parses JUnit XML test results into TestResults.
func ParseJUnitXML(data []byte) ([]model.TestResult, error) {
	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		// Try parsing as a single testsuite.
		var suite junitTestSuite
		if err2 := xml.Unmarshal(data, &suite); err2 != nil {
			return nil, fmt.Errorf("parsing JUnit XML: %w", err)
		}
		suites.Suites = []junitTestSuite{suite}
	}

	var results []model.TestResult
	for _, suite := range suites.Suites {
		for _, tc := range suite.TestCases {
			status := model.StatusPass
			var output string

			if tc.Skipped != nil {
				status = model.StatusSkip
			} else if tc.Failure != nil {
				status = model.StatusFail
				output = tc.Failure.Body
			}

			results = append(results, model.TestResult{
				Test: model.TestIdentifier{
					Package: tc.Classname,
					Name:    tc.Name,
				},
				Status:   status,
				Duration: time.Duration(tc.Time * float64(time.Second)),
				Output:   output,
			})
		}
	}

	return results, nil
}

// ParseForgeJSON parses forge test JSON output into TestResults.
func ParseForgeJSON(data []byte) ([]model.TestResult, error) {
	var raw struct {
		TestResults []struct {
			Contract string `json:"contract"`
			Test     string `json:"test"`
			Status   string `json:"status"`
			Duration int64  `json:"duration_ms"`
			Reason   string `json:"reason"`
		} `json:"test_results"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing forge JSON: %w", err)
	}

	var results []model.TestResult
	for _, tr := range raw.TestResults {
		status := model.StatusPass
		switch tr.Status {
		case "Failure", "failure":
			status = model.StatusFail
		case "Skip", "skip":
			status = model.StatusSkip
		}

		results = append(results, model.TestResult{
			Test: model.TestIdentifier{
				Package: tr.Contract,
				Name:    tr.Test,
			},
			Language: "sol",
			Status:   status,
			Duration: time.Duration(tr.Duration) * time.Millisecond,
			Output:   tr.Reason,
		})
	}

	return results, nil
}
