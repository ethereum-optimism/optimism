package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

// mockCmdRunner is a mock implementation of cmdRunner
type mockCmdRunner struct {
	mock.Mock
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	runErr error // Error to return from Run()
}

func (m *mockCmdRunner) CombinedOutput() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockCmdRunner) SetOutput(stdout, stderr *bytes.Buffer) {
	m.Called(stdout, stderr)
	m.stdout = stdout
	m.stderr = stderr
}

func (m *mockCmdRunner) Run() error {
	m.Called()
	// Simulate writing output if configured
	if m.stdout != nil {
		m.stdout.WriteString("mock stdout output\n")
	}
	if m.stderr != nil {
		m.stderr.WriteString("mock stderr output\n")
	}
	return m.runErr // Return the pre-configured error
}

// mockCmdFactory is a mock implementation of cmdFactory
type mockCmdFactory struct {
	mock.Mock
}

func (m *mockCmdFactory) Create(name string, arg ...string) cmdRunner {
	args := m.Called(name, arg)
	return args.Get(0).(cmdRunner)
}

// mockDockerClient is a mock implementation of dockerClient
type mockDockerClient struct {
	mock.Mock
}

func (m *mockDockerClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	args := m.Called(ctx, imageID)
	return args.Get(0).(types.ImageInspect), args.Get(1).([]byte), args.Error(2)
}

func (m *mockDockerClient) ImageTag(ctx context.Context, source, target string) error {
	args := m.Called(ctx, source, target)
	return args.Error(0)
}

// mockDockerProvider is a mock implementation of dockerProvider
type mockDockerProvider struct {
	mock.Mock
}

func (m *mockDockerProvider) newClient() (dockerClient, error) {
	args := m.Called()
	// Return nil for the client if an error is configured
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(dockerClient), args.Error(1)
}

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
	imageID := "sha256:1234567890abcdef1234567890abcdef"
	shortID := "1234567890ab"
	finalTag := fmt.Sprintf("%s:%s", projectName, shortID)

	// --- Mock Setup ---
	mockRunner := &mockCmdRunner{}
	mockRunner.On("SetOutput", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("*bytes.Buffer")).Return()
	mockRunner.On("Run").Return(nil) // Simulate successful command execution

	mockFactory := &mockCmdFactory{}
	mockFactory.On("Create", "sh", []string{"-c", fmt.Sprintf("just %s-image %s", projectName, initialTag)}).Return(mockRunner)

	mockClient := &mockDockerClient{}
	mockClient.On("ImageInspectWithRaw", mock.Anything, initialTag).Return(types.ImageInspect{ID: imageID}, []byte{}, nil)
	mockClient.On("ImageTag", mock.Anything, initialTag, finalTag).Return(nil)

	mockProvider := &mockDockerProvider{}
	mockProvider.On("newClient").Return(mockClient, nil)

	// --- Builder Setup ---
	builder := NewDockerBuilder(
		WithDockerDryRun(false), // Ensure not dry run
		withCmdFactory(mockFactory.Create),
		withDockerProvider(mockProvider),
		WithDockerConcurrency(1), // Explicitly set concurrency
	)

	// --- Execute ---
	resultTag, err := builder.Build(projectName, initialTag)

	// --- Assertions ---
	require.NoError(t, err)
	assert.Equal(t, finalTag, resultTag)

	// Verify mocks were called
	mockFactory.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockClient.AssertExpectations(t)

	// Verify log output
	logs := logBuf.String()
	assert.Contains(t, logs, fmt.Sprintf("Build started for project: %s (tag: %s)", projectName, initialTag))
	assert.Contains(t, logs, fmt.Sprintf("Executing build command for %s", projectName))
	assert.Contains(t, logs, fmt.Sprintf("Build successful for project: %s. Tagged as: %s", projectName, finalTag))
	assert.NotContains(t, logs, "mock stdout output") // Output should be hidden on success
	assert.NotContains(t, logs, "mock stderr output")
	assert.NotContains(t, logs, "Build failed for") // Should not contain failure messages
	assert.NotContains(t, logs, "--- Start Output") // Should not contain detailed output logs
}

func TestDockerBuilder_Build_CommandFailure(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	projectName := "fail-project"
	initialTag := "fail-project:enclave1"
	expectedError := errors.New("command execution failed")

	// --- Mock Setup ---
	mockRunner := &mockCmdRunner{runErr: expectedError} // Simulate command failure
	mockRunner.On("SetOutput", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("*bytes.Buffer")).Return()
	mockRunner.On("Run").Return(expectedError) // Use the specific error

	mockFactory := &mockCmdFactory{}
	mockFactory.On("Create", "sh", []string{"-c", fmt.Sprintf("just %s-image %s", projectName, initialTag)}).Return(mockRunner)

	// Docker client/provider mocks are not strictly needed here as the failure happens before,
	// but we include the provider mock for completeness if NewDockerBuilder tried to create one early.
	mockProvider := &mockDockerProvider{}

	// --- Builder Setup ---
	builder := NewDockerBuilder(
		WithDockerDryRun(false),
		withCmdFactory(mockFactory.Create),
		withDockerProvider(mockProvider), // Pass the mock provider
		WithDockerConcurrency(1),
	)

	// --- Execute ---
	resultTag, err := builder.Build(projectName, initialTag)

	// --- Assertions ---
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build command failed")
	assert.Equal(t, "", resultTag) // No tag should be returned on failure

	// Verify mocks
	mockFactory.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
	// No Docker client calls should have happened
	mockProvider.AssertNotCalled(t, "newClient")

	// Verify log output
	logs := logBuf.String()
	assert.Contains(t, logs, fmt.Sprintf("Build started for project: %s", projectName))
	assert.Contains(t, logs, fmt.Sprintf("Executing build command for %s", projectName))
	assert.Contains(t, logs, fmt.Sprintf("Build failed for %s", projectName))
	assert.Contains(t, logs, expectedError.Error()) // Check if the specific error message is logged
	assert.Contains(t, logs, "--- Start Output (stdout) for failed fail-project ---")
	assert.Contains(t, logs, "mock stdout output") // Output SHOULD be visible on failure
	assert.Contains(t, logs, "--- End Output (stdout) for failed fail-project ---")
	assert.Contains(t, logs, "--- Start Output (stderr) for failed fail-project ---")
	assert.Contains(t, logs, "mock stderr output") // Output SHOULD be visible on failure
	assert.Contains(t, logs, "--- End Output (stderr) for failed fail-project ---")
	assert.NotContains(t, logs, "Build successful for project") // Should not log success
}

func TestDockerBuilder_Build_ConcurrencyLimit(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	concurrencyLimit := 2
	numBuilds := 5
	buildDuration := 50 * time.Millisecond // Short duration for test speed

	// --- Mock Setup ---
	mockRunners := make([]*mockCmdRunner, numBuilds)
	mockFactory := &mockCmdFactory{}
	mockClient := &mockDockerClient{}
	mockProvider := &mockDockerProvider{}
	mockProvider.On("newClient").Return(mockClient, nil) // Assume client creation works

	buildStartTimes := make([]time.Time, numBuilds)
	buildEndTimes := make([]time.Time, numBuilds)
	completionOrder := make([]int, 0, numBuilds)
	var mu sync.Mutex // Protect shared slices

	for i := 0; i < numBuilds; i++ {
		projectName := fmt.Sprintf("concurrent-project-%d", i)
		initialTag := fmt.Sprintf("%s:enclave1", projectName)
		imageID := fmt.Sprintf("sha256:conc%dabcdef1234567890abcdef", i)

		// --- Calculate expected final tag based on actual logic ---
		shortID := TruncateID(imageID)                         // Simulate truncation
		finalTag := fmt.Sprintf("%s:%s", projectName, shortID) // Correct expectation

		runner := &mockCmdRunner{}
		runnerIdx := i // Capture loop variable

		runner.On("SetOutput", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("*bytes.Buffer")).Return()
		runner.On("Run").Run(func(args mock.Arguments) {
			mu.Lock()
			buildStartTimes[runnerIdx] = time.Now()
			mu.Unlock()
			time.Sleep(buildDuration) // Simulate build time
			mu.Lock()
			buildEndTimes[runnerIdx] = time.Now()
			completionOrder = append(completionOrder, runnerIdx)
			mu.Unlock()
		}).Return(nil) // Simulate successful command execution

		mockRunners[i] = runner
		mockFactory.On("Create", "sh", []string{"-c", fmt.Sprintf("just %s-image %s", projectName, initialTag)}).Return(runner)
		mockClient.On("ImageInspectWithRaw", mock.Anything, initialTag).Return(types.ImageInspect{ID: imageID}, []byte{}, nil)
		mockClient.On("ImageTag", mock.Anything, initialTag, finalTag).Return(nil)
	}

	// --- Builder Setup ---
	builder := NewDockerBuilder(
		withCmdFactory(mockFactory.Create),
		withDockerProvider(mockProvider),
		WithDockerConcurrency(concurrencyLimit), // Set the limit
	)

	// --- Execute Concurrently ---
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

	wg.Wait() // Wait for all builds to complete
	totalDuration := time.Since(startTime)

	// --- Assertions ---
	mockFactory.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	for _, runner := range mockRunners {
		runner.AssertExpectations(t)
	}

	// Basic check: total time should be roughly (numBuilds / concurrencyLimit) * buildDuration
	minExpectedDuration := time.Duration(numBuilds/concurrencyLimit) * buildDuration
	assert.GreaterOrEqual(t, totalDuration, minExpectedDuration, "Total duration too short, indicates lack of proper concurrency limiting")

	// More detailed check: At no point should more than 'concurrencyLimit' builds be running simultaneously.
	maxConcurrent := 0
	for i := 0; i < numBuilds; i++ {
		concurrentCount := 0
		for j := 0; j < numBuilds; j++ {
			// Check if build j overlaps with the start time of build i
			if buildStartTimes[j].Before(buildStartTimes[i]) && buildEndTimes[j].After(buildStartTimes[i]) {
				concurrentCount++
			}
			// Also count builds starting exactly at the same time (within tolerance)
			if buildStartTimes[j].Equal(buildStartTimes[i]) && i <= j {
				concurrentCount++
			}
		}
		if concurrentCount > maxConcurrent {
			maxConcurrent = concurrentCount
		}
	}
	assert.LessOrEqual(t, maxConcurrent, concurrencyLimit, "More builds ran concurrently than the limit")

	// Verify logs indicate success for all
	logs := logBuf.String()
	for i := 0; i < numBuilds; i++ {
		projectName := fmt.Sprintf("concurrent-project-%d", i)
		assert.Contains(t, logs, fmt.Sprintf("Build successful for project: %s", projectName))
	}
	assert.NotContains(t, logs, "Build failed for")
}

func TestDockerBuilder_Build_DryRun(t *testing.T) {
	logBuf, cleanup := captureLogs(t)
	defer cleanup()

	projectName := "dry-run-project"
	initialTag := "dry-run-project:enclave-dry"

	// --- Mock Setup ---
	// No mocks for command execution or docker client should be needed/called in dry run
	mockFactory := &mockCmdFactory{}
	mockProvider := &mockDockerProvider{}

	// --- Builder Setup ---
	builder := NewDockerBuilder(
		WithDockerDryRun(true), // Enable dry run
		withCmdFactory(mockFactory.Create),
		withDockerProvider(mockProvider),
		WithDockerConcurrency(1),
	)

	// --- Execute ---
	resultTag, err := builder.Build(projectName, initialTag)

	// --- Assertions ---
	require.NoError(t, err)
	// In dry run mode, the builder currently returns the *initial* tag provided.
	assert.Equal(t, initialTag, resultTag)

	// Verify mocks were NOT called
	mockFactory.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	mockProvider.AssertNotCalled(t, "newClient")

	// Verify log output for dry run
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
	imageID := "sha256:dup1234567890abcdef1234567890abcdef"

	// --- Calculate expected final tag based on actual logic ---
	shortID := TruncateID(imageID)                         // Simulate truncation
	finalTag := fmt.Sprintf("%s:%s", projectName, shortID) // Correct expectation
	buildDuration := 30 * time.Millisecond

	// --- Mock Setup ---
	mockRunner := &mockCmdRunner{}
	mockRunner.On("SetOutput", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("*bytes.Buffer")).Return()
	// Simulate build time only on the first call to Run
	runCallCount := 0
	mockRunner.On("Run").Run(func(args mock.Arguments) {
		runCallCount++
		if runCallCount == 1 {
			time.Sleep(buildDuration)
		}
	}).Return(nil).Once() // Expect Run to be called ONLY ONCE

	mockFactory := &mockCmdFactory{}
	// Expect Create to be called ONLY ONCE
	mockFactory.On("Create", "sh", []string{"-c", fmt.Sprintf("just %s-image %s", projectName, initialTag)}).Return(mockRunner).Once()

	mockClient := &mockDockerClient{}
	// Expect Inspect and Tag to be called ONLY ONCE
	mockClient.On("ImageInspectWithRaw", mock.Anything, initialTag).Return(types.ImageInspect{ID: imageID}, []byte{}, nil).Once()
	mockClient.On("ImageTag", mock.Anything, initialTag, finalTag).Return(nil).Once()

	mockProvider := &mockDockerProvider{}
	mockProvider.On("newClient").Return(mockClient, nil).Once() // Expect newClient ONLY ONCE

	// --- Builder Setup ---
	builder := NewDockerBuilder(
		withCmdFactory(mockFactory.Create),
		withDockerProvider(mockProvider),
		WithDockerConcurrency(2), // Allow multiple goroutines to proceed to Build if needed
	)

	// --- Execute Concurrently ---
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

	wg.Wait() // Wait for all calls to Build to return

	// --- Assertions ---
	// Verify mocks were called exactly once, despite multiple calls to Build
	mockFactory.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
	mockClient.AssertExpectations(t)

	// Check results from all calls
	for i := 0; i < numCalls; i++ {
		require.NoError(t, errors[i], "Call %d returned an error", i)
		assert.Equal(t, finalTag, results[i], "Call %d returned wrong tag", i)
	}

	// Verify logs show only one build execution sequence
	logs := logBuf.String()
	assert.Equal(t, 1, strings.Count(logs, fmt.Sprintf("Build started for project: %s", projectName)), "Expected build start log only once")
	assert.Equal(t, 1, strings.Count(logs, fmt.Sprintf("Executing build command for %s", projectName)), "Expected command execution log only once")
	assert.Equal(t, 1, strings.Count(logs, fmt.Sprintf("Build successful for project: %s", projectName)), "Expected success log only once")
	assert.NotContains(t, logs, "Build failed")

	fmt.Println("Captured logs for duplicate calls test:\n", logs) // Optional: print logs for manual inspection if test fails
}

// --- Test Default Command Runner ---
// This test ensures the real exec.Command interaction works as expected
// It's more of an integration test for the defaultCmdRunner wrapper.
func TestDefaultCmdRunner_Run_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command runner test in short mode.")
	}

	cmdRunner := defaultCmdFactory("echo", "hello world")
	var stdout, stderr bytes.Buffer
	cmdRunner.SetOutput(&stdout, &stderr)

	err := cmdRunner.Run()
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestDefaultCmdRunner_Run_Failure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command runner test in short mode.")
	}

	// Use a command guaranteed to fail and write to stderr
	cmdRunner := defaultCmdFactory("sh", "-c", "echo 'stdout message' && >&2 echo 'stderr message' && exit 1")
	var stdout, stderr bytes.Buffer
	cmdRunner.SetOutput(&stdout, &stderr)

	err := cmdRunner.Run()
	require.Error(t, err)
	// Check if it's an ExitError which is expected for command failures
	var exitErr *exec.ExitError
	assert.ErrorAs(t, err, &exitErr, "Error should be an exec.ExitError")
	assert.Equal(t, "stdout message\n", stdout.String())
	assert.Equal(t, "stderr message\n", stderr.String())
}

func TestDefaultCmdRunner_CombinedOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command runner test in short mode.")
	}

	// Test when SetOutput hasn't been called (internal buffering)
	cmdRunner1 := defaultCmdFactory("sh", "-c", "echo 'stdout' && >&2 echo 'stderr'")
	output1, err1 := cmdRunner1.CombinedOutput()
	require.NoError(t, err1)
	// Order isn't guaranteed, but both should be present
	assert.Contains(t, string(output1), "stdout")
	assert.Contains(t, string(output1), "stderr")

	// Test when SetOutput *has* been called
	cmdRunner2 := defaultCmdFactory("sh", "-c", "echo 'stdout2' && >&2 echo 'stderr2'")
	var stdout, stderr bytes.Buffer
	cmdRunner2.SetOutput(&stdout, &stderr)
	output2, err2 := cmdRunner2.CombinedOutput() // This should now call Run() internally and combine buffers
	require.NoError(t, err2)
	assert.Equal(t, "stdout2\n", stdout.String()) // Buffers should be populated
	assert.Equal(t, "stderr2\n", stderr.String())
	combined := "stdout2\n" + "stderr2\n"
	assert.Equal(t, combined, string(output2)) // Combined output should match concatenated buffers
}
