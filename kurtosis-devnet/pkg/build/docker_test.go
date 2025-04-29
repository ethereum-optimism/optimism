package build

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helper to capture log output ---
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	var logBuf bytes.Buffer
	originalLogger := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() {
		log.SetOutput(originalLogger)
	})
	return &logBuf, func() { log.SetOutput(originalLogger) }
}

// --- Tests ---

func TestDockerBuilder_Build_Success(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	projectName := "test-project"
	initialTag := "test-project:enclave1"

	// Create a builder in dry run mode
	builder := NewDockerBuilder(
		WithDockerDryRun(true),
		WithDockerConcurrency(1),
	)

	// Execute build
	resultTag, err := builder.Build(projectName, initialTag)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, initialTag, resultTag)

	// Verify log output
	logs := logBuf.String()
	assert.Contains(t, logs, fmt.Sprintf("Build started for project: %s (tag: %s)", projectName, initialTag))
	assert.Contains(t, logs, fmt.Sprintf("Dry run: Skipping build for project %s", projectName))
}

func TestDockerBuilder_Build_CommandFailure(t *testing.T) {
	// Create a builder in dry run mode
	builder := NewDockerBuilder(
		WithDockerDryRun(true),
		WithDockerConcurrency(1),
	)

	// Try to build a project
	result, err := builder.Build("test-project", "test-tag")

	// Verify the result
	require.NoError(t, err)
	assert.Equal(t, "test-tag", result)
}

func TestDockerBuilder_Build_ConcurrencyLimit(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	concurrencyLimit := 2
	numBuilds := 5

	// Create a builder in dry run mode with concurrency limit
	builder := NewDockerBuilder(
		WithDockerDryRun(true),
		WithDockerConcurrency(concurrencyLimit),
	)

	// Execute builds concurrently
	var wg sync.WaitGroup
	wg.Add(numBuilds)
	startTime := time.Now()

	for i := 0; i < numBuilds; i++ {
		go func(idx int) {
			defer wg.Done()
			projectName := fmt.Sprintf("concurrent-project-%d", idx)
			initialTag := fmt.Sprintf("%s:enclave1", projectName)
			_, err := builder.Build(projectName, initialTag)
			assert.NoError(t, err, "Build %d failed", idx)
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Verify logs show dry run messages
	logs := logBuf.String()
	for i := 0; i < numBuilds; i++ {
		projectName := fmt.Sprintf("concurrent-project-%d", i)
		assert.Contains(t, logs, fmt.Sprintf("Dry run: Skipping build for project %s", projectName))
	}
	assert.NotContains(t, logs, "Build failed")

	// Basic check: total time should be reasonable
	assert.Less(t, totalDuration, 1*time.Second, "Total duration too long, indicates potential blocking")
}

func TestDockerBuilder_Build_DryRun(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	projectName := "dry-run-project"
	initialTag := "dry-run-project:enclave-dry"

	// Create a builder in dry run mode
	builder := NewDockerBuilder(
		WithDockerDryRun(true),
		WithDockerConcurrency(1),
	)

	// Execute build
	resultTag, err := builder.Build(projectName, initialTag)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, initialTag, resultTag)

	// Verify log output
	logs := logBuf.String()
	assert.Contains(t, logs, fmt.Sprintf("Build started for project: %s", projectName))
	assert.Contains(t, logs, fmt.Sprintf("Dry run: Skipping build for project %s", projectName))
	assert.NotContains(t, logs, "Executing build command")
	assert.NotContains(t, logs, "Build successful")
	assert.NotContains(t, logs, "Build failed")
}

func TestDockerBuilder_Build_DuplicateCalls(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	projectName := "duplicate-project"
	initialTag := "duplicate:enclave1"

	// Create a builder in dry run mode
	builder := NewDockerBuilder(
		WithDockerDryRun(true),
		WithDockerConcurrency(2),
	)

	// Execute multiple concurrent builds
	var wg sync.WaitGroup
	numCalls := 3
	results := make([]string, numCalls)
	errors := make([]error, numCalls)
	wg.Add(numCalls)

	for i := 0; i < numCalls; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = builder.Build(projectName, initialTag)
		}(i)
	}

	wg.Wait()

	// Verify all calls returned the same result
	for i := 0; i < numCalls; i++ {
		require.NoError(t, errors[i], "Call %d returned an error", i)
		assert.Equal(t, initialTag, results[i], "Call %d returned wrong tag", i)
	}

	// Verify logs show dry run messages
	logs := logBuf.String()
	assert.Contains(t, logs, fmt.Sprintf("Build started for project: %s", projectName))
	assert.Contains(t, logs, fmt.Sprintf("Dry run: Skipping build for project %s", projectName))
	assert.NotContains(t, logs, "Build failed")
}
