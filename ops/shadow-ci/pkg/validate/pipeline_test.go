package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestValidatePipeline_Valid(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  setup:
    docker:
      - image: cimg/base:2024.01
    steps:
      - checkout
      - persist_to_workspace:
          root: /tmp
          paths:
            - results/
  test:
    docker:
      - image: cimg/base:2024.01
    steps:
      - checkout
      - attach_workspace:
          at: /tmp
      - run:
          name: Test
          command: echo ok
workflows:
  main:
    jobs:
      - setup
      - test:
          requires:
            - setup
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	assert.Empty(t, issues)
}

func TestValidatePipeline_DualWorkspaceRoots(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  setup:
    docker:
      - image: cimg/base:2024.01
    steps:
      - checkout
      - persist_to_workspace:
          root: .
          paths:
            - bin/
      - persist_to_workspace:
          root: /tmp
          paths:
            - results/
workflows:
  main:
    jobs:
      - setup
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "error", issues[0].Severity)
	assert.Contains(t, issues[0].Message, "multiple persist_to_workspace roots")
}

func TestValidatePipeline_DualAttachWorkspace(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  test:
    docker:
      - image: cimg/base:2024.01
    steps:
      - attach_workspace:
          at: .
      - attach_workspace:
          at: /tmp
      - run:
          name: Test
          command: echo ok
workflows:
  main:
    jobs:
      - test
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "multiple attach_workspace roots")
}

func TestValidatePipeline_CrossWorkflowRequires(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo build
  test:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo test
workflows:
  main:
    jobs:
      - build
  shadow:
    jobs:
      - test:
          requires:
            - build
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "error", issues[0].Severity)
	assert.Contains(t, issues[0].Message, `requires "build"`)
	assert.Contains(t, issues[0].Message, "not in this workflow")
}

func TestValidatePipeline_RequiresNonexistent(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  test:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo test
workflows:
  main:
    jobs:
      - test:
          requires:
            - nonexistent-job
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, `requires "nonexistent-job"`)
}

func TestValidatePipeline_MatrixCommas(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  sol-test:
    docker:
      - image: cimg/base:2024.01
    parameters:
      features:
        type: string
    steps:
      - run: echo test
workflows:
  main:
    jobs:
      - sol-test:
          name: sol-<<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features:
                - main
                - OPCM_V2
                - "OPCM_V2,CUSTOM_GAS_TOKEN"
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "warning", issues[0].Severity)
	assert.Contains(t, issues[0].Message, "comma")
	assert.Contains(t, issues[0].Message, "OPCM_V2,CUSTOM_GAS_TOKEN")
}

func TestValidatePipeline_PythonYamlImport(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  check:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run:
          name: Check
          command: |
            python3 -c "
            import json
            import yaml
            print('hello')
            "
workflows:
  main:
    jobs:
      - check
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "error", issues[0].Severity)
	assert.Contains(t, issues[0].Message, "import yaml")
}

func TestValidatePipeline_MatrixExpansionInRequires(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  setup:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo setup
  fuzz:
    docker:
      - image: cimg/base:2024.01
    parameters:
      package:
        type: string
    steps:
      - run: echo fuzz
  compare:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo compare
workflows:
  main:
    jobs:
      - setup
      - fuzz:
          name: fuzz-<<matrix.package>>
          package: <<matrix.package>>
          requires:
            - setup
          matrix:
            parameters:
              package:
                - op-node
                - cannon
      - compare:
          requires:
            - fuzz-op-node
            - fuzz-cannon
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	assert.Empty(t, issues)
}

func TestValidatePipeline_MatrixExpansionMismatch(t *testing.T) {
	path := writeYAML(t, `
version: 2.1
jobs:
  fuzz:
    docker:
      - image: cimg/base:2024.01
    parameters:
      package:
        type: string
    steps:
      - run: echo fuzz
  compare:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo compare
workflows:
  main:
    jobs:
      - fuzz:
          name: fuzz-<<matrix.package>>
          package: <<matrix.package>>
          matrix:
            parameters:
              package:
                - op-node
                - cannon
      - compare:
          requires:
            - fuzz-op-node
            - fuzz-cannon
            - fuzz-op-batcher
`)
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, `requires "fuzz-op-batcher"`)
}

func TestValidatePipeline_CurrentShadowCI(t *testing.T) {
	// Validate the actual shadow-ci.yml we ship.
	path := "../../../../.circleci/continue/shadow-ci.yml"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("shadow-ci.yml not found — run from worktree")
	}
	issues, err := ValidatePipeline(path)
	require.NoError(t, err)
	for _, issue := range issues {
		if issue.Severity == "error" {
			t.Errorf("validation error in shadow-ci.yml: %s", issue)
		} else {
			t.Logf("validation warning: %s", issue)
		}
	}
}
