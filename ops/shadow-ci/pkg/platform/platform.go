package platform

import "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"

// Platform abstracts a CI platform.
type Platform interface {
	// Render converts a TestPlan into platform-specific configuration.
	Render(plan model.TestPlan) ([]byte, error)

	// FetchResults retrieves test results from a completed pipeline run.
	FetchResults(pipelineID string) ([]model.TestResult, error)

	// StoreArtifact saves data as a pipeline artifact.
	StoreArtifact(name string, data []byte) error

	// FetchArtifact retrieves a previously stored artifact.
	FetchArtifact(pipelineID string, name string) ([]byte, error)

	// CurrentPipeline returns metadata about the currently executing pipeline.
	CurrentPipeline() PipelineInfo
}

// PipelineInfo holds metadata about a pipeline.
type PipelineInfo struct {
	ID     string `json:"id"`
	PR     int    `json:"pr"`
	Branch string `json:"branch"`
	Base   string `json:"base"`
	Head   string `json:"head"`
	Number int    `json:"number"`
}
