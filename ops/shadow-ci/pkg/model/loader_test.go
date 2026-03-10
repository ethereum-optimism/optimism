package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "adapters.yaml"), []byte(`
go:
  enabled: true
  root: "."
  special_paths:
    - "go.mod"
sol:
  enabled: true
  root: "packages/contracts-bedrock"
  source_dirs:
    - "src/"
    - "test/"
  features:
    - name: main
      env:
        FOUNDRY_PROFILE: ci
      always: true
rust:
  enabled: false
  root: "rust"
`), 0o644)

	os.WriteFile(filepath.Join(dir, "scoping.yaml"), []byte(`
confidence_threshold: 0.9
always_run_graduation_weeks: 8
force_all_paths:
  - ".circleci/"
always_run:
  go:
    - "github.com/ethereum-optimism/optimism/op-e2e/system"
activation:
  mode: shadow
  languages:
    go: true
    sol: true
    rust: false
`), 0o644)

	os.WriteFile(filepath.Join(dir, "platform.yaml"), []byte(`
platform: circleci
circleci:
  runners:
    small: "docker+medium"
    large: "latitude-1"
  main_ci:
    workflows:
      - main
events:
  store: local
  local:
    dir: /tmp/test-events
`), 0o644)

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)

	// Adapters.
	assert.True(t, cfg.Adapters.Go.Enabled)
	assert.Equal(t, ".", cfg.Adapters.Go.Root)
	assert.Contains(t, cfg.Adapters.Go.SpecialPaths, "go.mod")

	assert.True(t, cfg.Adapters.Sol.Enabled)
	assert.Equal(t, "packages/contracts-bedrock", cfg.Adapters.Sol.Root)
	assert.Len(t, cfg.Adapters.Sol.Features, 1)
	assert.Equal(t, "main", cfg.Adapters.Sol.Features[0].Name)

	assert.False(t, cfg.Adapters.Rust.Enabled)

	// Scoping.
	assert.Equal(t, 0.9, cfg.Scoping.ConfidenceThreshold)
	assert.Contains(t, cfg.Scoping.ForceAllPaths, ".circleci/")
	assert.Equal(t, "shadow", cfg.Scoping.Activation.Mode)

	// Platform.
	assert.Equal(t, "circleci", cfg.Platform.Platform)
	assert.Equal(t, "latitude-1", cfg.Platform.CircleCI.Runners["large"])
	assert.Equal(t, "local", cfg.Platform.Events.Store)
}

func writeMinimalConfig(t *testing.T, dir string, scopingYAML string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "adapters.yaml"), []byte("go:\n  enabled: true\n  root: \".\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "scoping.yaml"), []byte(scopingYAML), 0o644)
	os.WriteFile(filepath.Join(dir, "platform.yaml"), []byte("platform: circleci\nevents:\n  store: local\n  local:\n    dir: /tmp/test\n"), 0o644)
}

func TestLoadConfig_RejectsOverlappingWorkspacePaths(t *testing.T) {
	dir := t.TempDir()
	writeMinimalConfig(t, dir, `
confidence_threshold: 0.9
activation:
  mode: shadow
job_categories:
  cat_a:
    group: build
    trigger_paths: ["src/"]
    workspace_paths: ["build/output"]
    command: "make build-a"
    description: "Build A"
  cat_b:
    group: build
    trigger_paths: ["lib/"]
    workspace_paths: ["build/output"]
    command: "make build-b"
    description: "Build B"
`)
	_, err := LoadConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace_path")
	assert.Contains(t, err.Error(), "build/output")
}

func TestLoadConfig_AllowsSamePathDifferentGroups(t *testing.T) {
	dir := t.TempDir()
	writeMinimalConfig(t, dir, `
confidence_threshold: 0.9
activation:
  mode: shadow
job_categories:
  cat_a:
    group: build
    trigger_paths: ["src/"]
    workspace_paths: ["build/output"]
    command: "make build"
    description: "Build"
  cat_b:
    group: go
    trigger_paths: ["src/"]
    workspace_paths: ["build/output"]
    command: "make test"
    description: "Test"
`)
	_, err := LoadConfig(dir)
	require.NoError(t, err)
}
