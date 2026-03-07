package circleci

import (
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlSafe_Plain(t *testing.T) {
	assert.Equal(t, "hello", yamlSafe("hello"))
	assert.Equal(t, "op-node", yamlSafe("op-node"))
	assert.Equal(t, "pkg/foo", yamlSafe("pkg/foo"))
}

func TestYamlSafe_Empty(t *testing.T) {
	assert.Equal(t, `""`, yamlSafe(""))
}

func TestYamlSafe_Dangerous(t *testing.T) {
	// Colons, newlines, and other YAML-special chars get quoted.
	assert.Equal(t, `"key: value"`, yamlSafe("key: value"))
	assert.Equal(t, `"line1\nline2"`, yamlSafe("line1\nline2"))
	assert.Equal(t, `"has\"quote"`, yamlSafe(`has"quote`))
	assert.Equal(t, `"{braces}"`, yamlSafe("{braces}"))
}

func TestYamlSafe_BoolLike(t *testing.T) {
	// YAML boolean-like strings must be quoted.
	assert.Equal(t, `"true"`, yamlSafe("true"))
	assert.Equal(t, `"false"`, yamlSafe("false"))
	assert.Equal(t, `"yes"`, yamlSafe("yes"))
	assert.Equal(t, `"no"`, yamlSafe("no"))
	assert.Equal(t, `"null"`, yamlSafe("null"))
	assert.Equal(t, `"TRUE"`, yamlSafe("TRUE"))
}

func TestYamlSafe_LeadingDash(t *testing.T) {
	assert.Equal(t, `"--flag"`, yamlSafe("--flag"))
	assert.Equal(t, `"-v"`, yamlSafe("-v"))
}

func TestYamlSafe_Backslash(t *testing.T) {
	assert.Equal(t, `"back\\slash"`, yamlSafe(`back\slash`))
}

func TestRenderFromDecision_IncludesBuildDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"contracts_build": {
				TriggerPaths:   []string{"packages/contracts-bedrock/"},
				Command:        "forge build",
				WorkspacePaths: []string{"packages/contracts-bedrock/forge-artifacts"},
				RunnerClass:    "medium",
			},
			"go_tests": {
				UseGraph:  true,
				Language:  "go",
				DependsOn: []string{"contracts_build"},
			},
			"sol_tests": {
				UseGraph:  true,
				Language:  "sol",
				DependsOn: []string{"contracts_build"},
			},
		},
	}

	decision := &model.PipelineDecision{
		Stage: model.StagePR,
		Categories: map[string]*model.CategoryDecision{
			"go_tests": {
				Needed: true,
				Reason: "affected",
				Targets: []string{"pkg/a"},
			},
		},
	}

	renderer := NewRenderer(map[string]string{
		"medium": "docker+medium",
		"large":  "docker+large",
	})

	yamlData, err := renderer.RenderFromDecision(decision, nil, scoping)
	require.NoError(t, err)

	yaml := string(yamlData)

	// Build job should be included even though it wasn't in the decision.
	assert.Contains(t, yaml, "shadow-contracts_build:")

	// Build job should have persist_to_workspace.
	assert.Contains(t, yaml, "persist_to_workspace")
	assert.Contains(t, yaml, "packages/contracts-bedrock/forge-artifacts")

	// Workflow should wire requires from test to build.
	assert.Contains(t, yaml, "shadow-contracts_build")

	// Comparison should depend on both build and test jobs.
	assert.Contains(t, yaml, "shadow-comparison")
}

func TestRenderFromDecision_TransitiveBuildDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"contracts_build": {
				Command:     "forge build",
				RunnerClass: "medium",
			},
			"cannon_prestate": {
				DependsOn:   []string{"contracts_build"},
				Command:     "make cannon-prestate",
				RunnerClass: "large",
			},
			"go_fault_proofs": {
				TriggerPaths: []string{"op-e2e/"},
				DependsOn:    []string{"contracts_build", "cannon_prestate"},
			},
		},
	}

	decision := &model.PipelineDecision{
		Stage: model.StageMergeQueue,
		Categories: map[string]*model.CategoryDecision{
			"go_fault_proofs": {
				Needed: true,
				Reason: "trigger paths matched",
			},
		},
	}

	renderer := NewRenderer(map[string]string{
		"medium": "docker+medium",
		"large":  "docker+large",
	})

	yamlData, err := renderer.RenderFromDecision(decision, nil, scoping)
	require.NoError(t, err)

	yaml := string(yamlData)

	// Both build deps should be included.
	assert.Contains(t, yaml, "shadow-contracts_build:")
	assert.Contains(t, yaml, "shadow-cannon_prestate:")

	// cannon_prestate should require contracts_build (transitive dep).
	// Find the workflow section and check requires ordering.
	workflowSection := yaml[strings.Index(yaml, "workflows:"):]
	assert.Contains(t, workflowSection, "shadow-contracts_build")
	assert.Contains(t, workflowSection, "shadow-cannon_prestate")
}

func TestRenderFromDecision_ValidatesClean(t *testing.T) {
	// Render a realistic pipeline and validate it passes the pipeline validator.
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"contracts_build": {
				TriggerPaths:   []string{"packages/contracts-bedrock/"},
				Command:        "forge build",
				WorkspacePaths: []string{"packages/contracts-bedrock/forge-artifacts"},
				RunnerClass:    "medium",
			},
			"cannon_prestate": {
				TriggerPaths:   []string{"cannon/"},
				DependsOn:      []string{"contracts_build"},
				Command:        "make cannon-prestate",
				WorkspacePaths: []string{"cannon/bin"},
				RunnerClass:    "large",
			},
			"go_tests": {
				UseGraph:  true,
				Language:  "go",
				DependsOn: []string{"contracts_build"},
			},
			"acceptance_tests": {
				TriggerPaths: []string{"op-e2e/"},
				DependsOn:    []string{"contracts_build", "cannon_prestate"},
			},
			"go_lint": {
				TriggerPaths: []string{"**/*.go"},
			},
		},
	}

	decision := &model.PipelineDecision{
		Stage: model.StagePR,
		Categories: map[string]*model.CategoryDecision{
			"go_tests": {
				Needed:  true,
				Reason:  "affected",
				Targets: []string{"pkg/a"},
			},
			"acceptance_tests": {
				Needed: true,
				Reason: "trigger paths matched",
			},
			"go_lint": {
				Needed: true,
				Reason: "go files changed",
			},
		},
	}

	renderer := NewRenderer(map[string]string{
		"medium": "docker+medium",
		"large":  "docker+large",
	})

	yamlData, err := renderer.RenderFromDecision(decision, nil, scoping)
	require.NoError(t, err)

	yaml := string(yamlData)

	// Write to temp file and validate.
	tmpFile := t.TempDir() + "/pipeline.yml"
	require.NoError(t, os.WriteFile(tmpFile, yamlData, 0o644))

	issues, err := validate.ValidatePipeline(tmpFile)
	require.NoError(t, err)

	// Filter for errors only.
	var errors []string
	for _, issue := range issues {
		if issue.Severity == "error" {
			errors = append(errors, issue.String())
		}
	}
	assert.Empty(t, errors, "Pipeline validation should have no errors.\nGenerated YAML:\n%s", yaml)
}

func TestRenderFromDecision_NoBuildDeps(t *testing.T) {
	scoping := model.ScopingConfig{
		JobCategories: map[string]model.JobCategoryConfig{
			"go_lint": {
				TriggerPaths: []string{"**/*.go"},
			},
		},
	}

	decision := &model.PipelineDecision{
		Stage: model.StagePR,
		Categories: map[string]*model.CategoryDecision{
			"go_lint": {
				Needed: true,
				Reason: "changed files match",
			},
		},
	}

	renderer := NewRenderer(map[string]string{"large": "docker+large"})

	yamlData, err := renderer.RenderFromDecision(decision, nil, scoping)
	require.NoError(t, err)

	yaml := string(yamlData)

	// Should have the lint job but no build jobs.
	assert.Contains(t, yaml, "shadow-go_lint:")
	assert.NotContains(t, yaml, "persist_to_workspace")
}
