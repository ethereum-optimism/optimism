package buildcache

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// MockCommandExecutor allows us to mock os/exec calls for testing
type MockCommandExecutor struct {
	commands  []MockCommand
	callLog   []string
	callIndex int
}

type MockCommand struct {
	name     string
	args     []string
	exitCode int
	err      error
}

func (m *MockCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	// Record the command call
	cmdStr := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	m.callLog = append(m.callLog, cmdStr)

	// Use sequential matching based on call order
	if m.callIndex < len(m.commands) {
		mockCmd := m.commands[m.callIndex]
		m.callIndex++

		// Verify the command matches what we expect
		if mockCmd.name == name && len(mockCmd.args) == len(args) {
			match := true
			for i, arg := range args {
				if mockCmd.args[i] != arg {
					match = false
					break
				}
			}
			if match {
				// Create a command that will simulate the expected behavior
				if mockCmd.err != nil {
					// Return a command that will fail
					return exec.CommandContext(ctx, "false") // 'false' command always exits with code 1
				}
				if mockCmd.exitCode != 0 {
					// Return a command that will exit with specific code
					return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("exit %d", mockCmd.exitCode))
				}
				// Return a successful command
				return exec.CommandContext(ctx, "true") // 'true' command always exits with code 0
			}
		}
	}

	// Default to successful command if no mock found or mismatch
	return exec.CommandContext(ctx, "true")
}

func (m *MockCommandExecutor) GetCallLog() []string {
	return m.callLog
}

func (m *MockCommandExecutor) Reset() {
	m.callLog = nil
	m.callIndex = 0
}

// TestableSmartBuildManager extends SmartBuildManager to allow command mocking
type TestableSmartBuildManager struct {
	*SmartBuildManager
	mockExecutor *MockCommandExecutor
}

func NewTestableSmartBuildManager(contractsDir string, logger *log.Logger) *TestableSmartBuildManager {
	manager := NewSmartBuildManager(contractsDir, logger)
	return &TestableSmartBuildManager{
		SmartBuildManager: manager,
		mockExecutor:      &MockCommandExecutor{},
	}
}

func (m *TestableSmartBuildManager) SetMockCommands(commands []MockCommand) {
	m.mockExecutor.commands = commands
	m.mockExecutor.Reset()
}

func (m *TestableSmartBuildManager) GetCommandLog() []string {
	return m.mockExecutor.GetCallLog()
}

// Override ExecuteBuild to use our mocked methods
func (m *TestableSmartBuildManager) ExecuteBuild(ctx context.Context) error {
	m.debugLog("Starting ExecuteBuild with contracts directory: %s", m.contractsDir)
	m.debugLog("Configuration - Force rebuild: %v, Skip check: %v", m.forceRebuild, m.skipCheck)

	// Validate build environment before proceeding
	if err := m.validateBuildEnvironment(); err != nil {
		m.logger.Printf("Build environment validation failed: %v, attempting graceful fallback", err)
		return m.performCleanBuildWithFallback(ctx)
	}

	// Check for force rebuild
	if m.forceRebuild {
		m.logger.Printf("Force rebuild requested, performing clean build...")
		return m.performCleanBuildWithFallback(ctx)
	}

	// Check if we should skip cache validation (useful for CI)
	if m.skipCheck {
		m.logger.Printf("Skipping build check, performing clean build...")
		return m.performCleanBuildWithFallback(ctx)
	}

	// Check cache validity with comprehensive error handling
	cacheValid, err := m.validateCacheWithFallback(ctx)
	if err != nil {
		m.logger.Printf("Cache validation failed with unrecoverable error: %v, falling back to clean build", err)
		return m.performCleanBuildWithFallback(ctx)
	}

	if cacheValid {
		m.logger.Printf("Build cache is valid, skipping rebuild...")
		return nil
	}

	m.logger.Printf("Build cache is stale, performing incremental rebuild...")
	return m.performIncrementalBuildWithFallback(ctx)
}

// Override the build methods to use mock executor
func (m *TestableSmartBuildManager) performCleanBuild(ctx context.Context) error {
	m.logger.Printf("Performing clean build...")

	// Execute forge clean
	cleanCmd := m.mockExecutor.CommandContext(ctx, "forge", "clean")
	cleanCmd.Dir = m.contractsDir
	if err := cleanCmd.Run(); err != nil {
		m.logger.Printf("Warning: forge clean failed: %v", err)
		// Continue with build even if clean fails
	}

	// Execute just forge-build
	return m.performBuild(ctx)
}

func (m *TestableSmartBuildManager) performIncrementalBuild(ctx context.Context) error {
	m.logger.Printf("Performing incremental build...")
	return m.performBuild(ctx)
}

func (m *TestableSmartBuildManager) performBuild(ctx context.Context) error {
	return m.performBuildWithErrorHandling(ctx)
}

func (m *TestableSmartBuildManager) performBuildWithErrorHandling(ctx context.Context) error {
	m.debugLog("Executing build command with error handling")

	buildCmd := m.mockExecutor.CommandContext(ctx, "just", "forge-build")
	buildCmd.Dir = m.contractsDir

	if err := buildCmd.Run(); err != nil {
		// Provide more detailed error information
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("build command failed with exit code %d: %w", exitError.ExitCode(), err)
		}
		return fmt.Errorf("build command execution failed: %w", err)
	}

	m.logger.Printf("Build completed successfully")
	return nil
}

func (m *TestableSmartBuildManager) performCleanBuildWithFallback(ctx context.Context) error {
	m.debugLog("Performing clean build with fallback handling")

	// First attempt: try the normal clean build process
	if err := m.performCleanBuild(ctx); err != nil {
		m.logger.Printf("Clean build failed: %v", err)

		// Fallback strategy: try build without clean
		m.logger.Printf("Attempting fallback: build without clean...")
		if fallbackErr := m.performBuildWithErrorHandling(ctx); fallbackErr != nil {
			// If both attempts fail, return a comprehensive error
			return fmt.Errorf("clean build failed (%w) and fallback build also failed (%w)", err, fallbackErr)
		}

		m.logger.Printf("Fallback build succeeded after clean build failure")
		return nil
	}

	m.debugLog("Clean build completed successfully")
	return nil
}

func (m *TestableSmartBuildManager) performIncrementalBuildWithFallback(ctx context.Context) error {
	m.debugLog("Performing incremental build with fallback handling")

	// First attempt: try incremental build
	if err := m.performIncrementalBuild(ctx); err != nil {
		m.logger.Printf("Incremental build failed: %v", err)

		// Fallback strategy: try clean build
		m.logger.Printf("Attempting fallback: clean build after incremental build failure...")
		if fallbackErr := m.performCleanBuild(ctx); fallbackErr != nil {
			// If both attempts fail, return a comprehensive error
			return fmt.Errorf("incremental build failed (%w) and fallback clean build also failed (%w)", err, fallbackErr)
		}

		m.logger.Printf("Fallback clean build succeeded after incremental build failure")
		return nil
	}

	m.debugLog("Incremental build completed successfully")
	return nil
}

// Override validateBuildEnvironment to allow testing directory validation
func (m *TestableSmartBuildManager) validateBuildEnvironment() error {
	m.debugLog("Validating build environment")

	// Check if contracts directory exists and is accessible
	if _, err := os.Stat(m.contractsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("contracts directory does not exist: %s", m.contractsDir)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing contracts directory: %s", m.contractsDir)
		}
		return fmt.Errorf("error accessing contracts directory %s: %w", m.contractsDir, err)
	}

	// For testing, we'll skip the tool validation since we're mocking the commands
	m.debugLog("Build environment validation successful")
	return nil
}

func TestSmartBuildManager_ExecuteBuild_Integration(t *testing.T) {
	tests := []struct {
		name             string
		setupFunc        func(t *testing.T, tempDir string)
		forceRebuild     bool
		skipCheck        bool
		envVars          map[string]string
		mockCommands     []MockCommand
		expectedCommands []string
		expectError      bool
		errorContains    string
	}{
		{
			name: "force rebuild executes clean build workflow",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild: true,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
			expectError:      false,
		},
		{
			name: "skip check executes clean build workflow",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild: false,
			skipCheck:    true,
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
			expectError:      false,
		},
		{
			name: "valid cache skips build entirely",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild:     false,
			skipCheck:        false,
			mockCommands:     []MockCommand{},
			expectedCommands: []string{}, // No commands should be executed
			expectError:      false,
		},
		{
			name: "invalid cache triggers incremental build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupInvalidCache(t, tempDir)
			},
			forceRebuild: false,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"just forge-build"}, // No clean command
			expectError:      false,
		},
		{
			name: "missing artifacts triggers incremental build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupMissingArtifacts(t, tempDir)
			},
			forceRebuild: false,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"just forge-build"},
			expectError:      false,
		},
		{
			name: "build failure triggers fallback to clean build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupInvalidCache(t, tempDir)
			},
			forceRebuild: false,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "just", args: []string{"forge-build"}, exitCode: 1}, // First build fails
				{name: "forge", args: []string{"clean"}, exitCode: 0},      // Fallback clean
				{name: "just", args: []string{"forge-build"}, exitCode: 0}, // Fallback build succeeds
			},
			expectedCommands: []string{"just forge-build", "forge clean", "just forge-build"},
			expectError:      false,
		},
		{
			name: "clean build failure with successful fallback",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild: true,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 1}, // Clean build fails
				{name: "just", args: []string{"forge-build"}, exitCode: 0}, // Fallback succeeds
			},
			expectedCommands: []string{"forge clean", "just forge-build", "just forge-build"},
			expectError:      false,
		},
		{
			name: "both build and fallback fail",
			setupFunc: func(t *testing.T, tempDir string) {
				setupInvalidCache(t, tempDir)
			},
			forceRebuild: false,
			skipCheck:    false,
			mockCommands: []MockCommand{
				{name: "just", args: []string{"forge-build"}, exitCode: 1}, // First build fails
				{name: "forge", args: []string{"clean"}, exitCode: 0},      // Fallback clean
				{name: "just", args: []string{"forge-build"}, exitCode: 1}, // Fallback build also fails
			},
			expectedCommands: []string{"just forge-build", "forge clean", "just forge-build"},
			expectError:      true,
			errorContains:    "incremental build failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "integration_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Setup test scenario
			tt.setupFunc(t, tempDir)

			// Create testable manager
			logger := log.New(os.Stdout, "[INTEGRATION-TEST] ", log.LstdFlags)
			manager := NewTestableSmartBuildManager(tempDir, logger)
			manager.SetForceRebuild(tt.forceRebuild)
			manager.SetSkipCheck(tt.skipCheck)
			manager.SetMockCommands(tt.mockCommands)

			// Execute build
			ctx := context.Background()
			err = manager.ExecuteBuild(ctx)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check executed commands
			actualCommands := manager.GetCommandLog()
			if len(actualCommands) != len(tt.expectedCommands) {
				t.Errorf("Expected %d commands, got %d: %v", len(tt.expectedCommands), len(actualCommands), actualCommands)
			} else {
				for i, expected := range tt.expectedCommands {
					if actualCommands[i] != expected {
						t.Errorf("Command %d: expected '%s', got '%s'", i, expected, actualCommands[i])
					}
				}
			}
		})
	}
}

func TestSmartBuildManager_EnvironmentVariableIntegration(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		setupFunc        func(t *testing.T, tempDir string)
		mockCommands     []MockCommand
		expectedCommands []string
	}{
		{
			name: "FORCE_REBUILD=true triggers clean build",
			envVars: map[string]string{
				"FORCE_REBUILD": "true",
			},
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
		},
		{
			name: "SKIP_BUILD_CHECK=true triggers clean build",
			envVars: map[string]string{
				"SKIP_BUILD_CHECK": "true",
			},
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
		},
		{
			name: "BUILD_CACHE_DEBUG=true enables debug logging",
			envVars: map[string]string{
				"BUILD_CACHE_DEBUG": "true",
			},
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands:     []MockCommand{},
			expectedCommands: []string{}, // Valid cache, no commands
		},
		{
			name: "Multiple environment variables work together",
			envVars: map[string]string{
				"FORCE_REBUILD":     "true",
				"BUILD_CACHE_DEBUG": "true",
			},
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "env_integration_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			tt.setupFunc(t, tempDir)

			// Create testable manager (environment variables are read in constructor)
			logger := log.New(os.Stdout, "[ENV-INTEGRATION-TEST] ", log.LstdFlags)
			manager := NewTestableSmartBuildManager(tempDir, logger)
			manager.SetMockCommands(tt.mockCommands)

			// Execute build
			ctx := context.Background()
			err = manager.ExecuteBuild(ctx)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check executed commands
			actualCommands := manager.GetCommandLog()
			if len(actualCommands) != len(tt.expectedCommands) {
				t.Errorf("Expected %d commands, got %d: %v", len(tt.expectedCommands), len(actualCommands), actualCommands)
			} else {
				for i, expected := range tt.expectedCommands {
					if actualCommands[i] != expected {
						t.Errorf("Command %d: expected '%s', got '%s'", i, expected, actualCommands[i])
					}
				}
			}
		})
	}
}

func TestSmartBuildManager_BuildEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, tempDir string)
		mockCommands  []MockCommand
		expectError   bool
		errorContains string
	}{
		{
			name: "missing contracts directory fails validation",
			setupFunc: func(t *testing.T, tempDir string) {
				// Don't create the contracts directory - use a non-existent path
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 1}, // Fallback should also fail
				{name: "just", args: []string{"forge-build"}, exitCode: 1},
			},
			expectError:   true,
			errorContains: "clean build failed",
		},
		{
			name: "valid contracts directory passes validation",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "validation_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// For the missing directory test, use a non-existent subdirectory
			contractsDir := tempDir
			if strings.Contains(tt.name, "missing contracts directory") {
				contractsDir = filepath.Join(tempDir, "nonexistent")
			}

			// Setup test scenario
			tt.setupFunc(t, tempDir)

			// Create testable manager
			logger := log.New(os.Stdout, "[VALIDATION-TEST] ", log.LstdFlags)
			manager := NewTestableSmartBuildManager(contractsDir, logger)
			manager.SetMockCommands(tt.mockCommands)

			// Force rebuild to trigger validation
			manager.SetForceRebuild(true)

			// Execute build
			ctx := context.Background()
			err = manager.ExecuteBuild(ctx)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSmartBuildManager_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name             string
		setupFunc        func(t *testing.T, tempDir string)
		mockCommands     []MockCommand
		expectedCommands []string
		expectError      bool
	}{
		{
			name: "forge clean fails but build succeeds",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 1},      // Clean fails
				{name: "just", args: []string{"forge-build"}, exitCode: 0}, // Build succeeds
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
			expectError:      false, // Clean failure is non-fatal
		},
		{
			name: "missing source directory causes cache validation to fail and falls back to clean build",
			setupFunc: func(t *testing.T, tempDir string) {
				// Create artifacts but no source directory to cause validation failure
				artifactsDir := filepath.Join(tempDir, "forge-artifacts")
				if err := os.MkdirAll(artifactsDir, 0755); err != nil {
					t.Fatal(err)
				}
				artifactPath := filepath.Join(artifactsDir, "test.json")
				if err := os.WriteFile(artifactPath, []byte(`{"artifact": "data"}`), 0644); err != nil {
					t.Fatal(err)
				}
				// Don't create src directory - this will cause getNewestSourceTime to fail
			},
			mockCommands: []MockCommand{
				{name: "forge", args: []string{"clean"}, exitCode: 0},
				{name: "just", args: []string{"forge-build"}, exitCode: 0},
			},
			expectedCommands: []string{"forge clean", "just forge-build"},
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "complex_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			tt.setupFunc(t, tempDir)

			// Create testable manager
			logger := log.New(os.Stdout, "[COMPLEX-TEST] ", log.LstdFlags)
			manager := NewTestableSmartBuildManager(tempDir, logger)
			manager.SetMockCommands(tt.mockCommands)

			// For the first test, force rebuild to trigger clean
			if strings.Contains(tt.name, "forge clean fails") {
				manager.SetForceRebuild(true)
			}

			// Execute build
			ctx := context.Background()
			err = manager.ExecuteBuild(ctx)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check executed commands
			actualCommands := manager.GetCommandLog()
			if len(actualCommands) != len(tt.expectedCommands) {
				t.Errorf("Expected %d commands, got %d: %v", len(tt.expectedCommands), len(actualCommands), actualCommands)
			} else {
				for i, expected := range tt.expectedCommands {
					if actualCommands[i] != expected {
						t.Errorf("Command %d: expected '%s', got '%s'", i, expected, actualCommands[i])
					}
				}
			}
		})
	}
}
