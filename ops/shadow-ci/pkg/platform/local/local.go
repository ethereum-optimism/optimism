package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform"
)

// Adapter implements platform.Platform for local development and testing.
type Adapter struct {
	outputDir string
}

// NewAdapter creates a local platform adapter.
func NewAdapter(outputDir string) *Adapter {
	os.MkdirAll(outputDir, 0o755)
	return &Adapter{outputDir: outputDir}
}

// Render outputs the test plan as shell commands for local execution.
func (a *Adapter) Render(plan model.TestPlan) ([]byte, error) {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (a *Adapter) FetchResults(pipelineID string) ([]model.TestResult, error) {
	path := filepath.Join(a.outputDir, pipelineID, "results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading local results: %w", err)
	}

	var results []model.TestResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (a *Adapter) StoreArtifact(name string, data []byte) error {
	return os.WriteFile(filepath.Join(a.outputDir, name), data, 0o644)
}

func (a *Adapter) FetchArtifact(pipelineID string, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(a.outputDir, name))
}

func (a *Adapter) CurrentPipeline() platform.PipelineInfo {
	return platform.PipelineInfo{
		ID:     "local",
		Branch: "local",
	}
}
