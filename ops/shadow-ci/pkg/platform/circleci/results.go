package circleci

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform"
)

// Adapter implements platform.Platform for CircleCI.
type Adapter struct {
	client         *Client
	renderer       *Renderer
	artifactDir    string
	runners        map[string]string
	mainWorkflows  []string
	artifactGlobs  map[string]string
}

// NewAdapter creates a new CircleCI platform adapter.
func NewAdapter(cfg model.CircleCIConfig) *Adapter {
	return &Adapter{
		client:        NewClientFromEnv(),
		renderer:      NewRenderer(cfg.Runners),
		artifactDir:   "/tmp/shadow-ci-results",
		runners:       cfg.Runners,
		mainWorkflows: cfg.MainCI.Workflows,
		artifactGlobs: cfg.MainCI.ArtifactPatterns,
	}
}

func (a *Adapter) Render(plan model.TestPlan) ([]byte, error) {
	return a.renderer.Render(plan)
}

// FetchResults retrieves test results from a completed main CI pipeline.
func (a *Adapter) FetchResults(pipelineID string) ([]model.TestResult, error) {
	workflows, err := a.client.GetWorkflows(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("getting workflows: %w", err)
	}

	var allResults []model.TestResult

	for _, wf := range workflows {
		// Only look at main CI workflows.
		if !a.isMainWorkflow(wf.Name) {
			continue
		}

		jobs, err := a.client.GetWorkflowJobs(wf.ID)
		if err != nil {
			continue
		}

		for _, job := range jobs {
			artifacts, err := a.client.GetJobArtifacts(job.ProjectSlug, job.JobNumber)
			if err != nil {
				continue
			}

			for _, artifact := range artifacts {
				if !a.isTestArtifact(artifact.Path) {
					continue
				}

				data, err := a.client.DownloadArtifact(artifact.URL)
				if err != nil {
					continue
				}

				results, err := parseArtifact(data, artifact.Path)
				if err != nil {
					continue
				}

				allResults = append(allResults, results...)
			}
		}
	}

	return allResults, nil
}

func (a *Adapter) StoreArtifact(name string, data []byte) error {
	dir := a.artifactDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

func (a *Adapter) FetchArtifact(pipelineID string, name string) ([]byte, error) {
	// For local artifacts during same pipeline.
	path := filepath.Join(a.artifactDir, name)
	return os.ReadFile(path)
}

func (a *Adapter) CurrentPipeline() platform.PipelineInfo {
	return platform.PipelineInfo{
		ID:     os.Getenv("CIRCLE_PIPELINE_ID"),
		Branch: os.Getenv("CIRCLE_BRANCH"),
		Head:   os.Getenv("CIRCLE_SHA1"),
	}
}

func (a *Adapter) isMainWorkflow(name string) bool {
	for _, wf := range a.mainWorkflows {
		if wf == name {
			return true
		}
	}
	return false
}

func (a *Adapter) isTestArtifact(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "junit") ||
		strings.HasSuffix(lower, ".xml") ||
		strings.Contains(lower, "results")
}

// parseArtifact attempts to parse a test artifact into TestResults.
func parseArtifact(data []byte, path string) ([]model.TestResult, error) {
	if strings.HasSuffix(path, ".xml") {
		return parseJunitXML(data)
	}
	return nil, fmt.Errorf("unsupported artifact format: %s", path)
}

type junitSuites struct {
	XMLName xml.Name     `xml:"testsuites"`
	Suites  []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name  string      `xml:"name,attr"`
	Cases []junitCase `xml:"testcase"`
}

type junitCase struct {
	Name      string       `xml:"name,attr"`
	ClassName string       `xml:"classname,attr"`
	Time      float64      `xml:"time,attr"`
	Failure   *junitFail   `xml:"failure,omitempty"`
	Error     *junitErr    `xml:"error,omitempty"`
	Skipped   *junitSkip   `xml:"skipped,omitempty"`
}

type junitFail struct {
	Body string `xml:",chardata"`
}

type junitErr struct {
	Body string `xml:",chardata"`
}

type junitSkip struct{}

func parseJunitXML(data []byte) ([]model.TestResult, error) {
	var suites junitSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		return nil, err
	}

	var results []model.TestResult
	for _, suite := range suites.Suites {
		for _, tc := range suite.Cases {
			r := model.TestResult{
				Test: model.TestIdentifier{
					Name:    tc.Name,
					Package: tc.ClassName,
				},
				Duration: time.Duration(tc.Time * float64(time.Second)),
				Status:   model.StatusPass,
			}
			switch {
			case tc.Error != nil:
				r.Status = model.StatusError
				r.Output = tc.Error.Body
			case tc.Failure != nil:
				r.Status = model.StatusFail
				r.Output = tc.Failure.Body
			case tc.Skipped != nil:
				r.Status = model.StatusSkip
			}
			results = append(results, r)
		}
	}
	return results, nil
}
