package buildcache

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// SmartBuildManager orchestrates the build process based on cache validity and configuration
type SmartBuildManager struct {
	validator    *BuildCacheValidator
	contractsDir string
	forceRebuild bool
	skipCheck    bool
	logger       *log.Logger
	debugEnabled bool
}

// NewSmartBuildManager creates a new SmartBuildManager instance
func NewSmartBuildManager(contractsDir string, logger *log.Logger) *SmartBuildManager {
	if logger == nil {
		logger = log.New(os.Stdout, "[BUILD-MANAGER] ", log.LstdFlags)
	}

	validator := NewBuildCacheValidator(contractsDir, logger)

	// Enable debug logging if BUILD_CACHE_DEBUG environment variable is set
	debugEnabled := os.Getenv("BUILD_CACHE_DEBUG") == "true"

	return &SmartBuildManager{
		validator:    validator,
		contractsDir: contractsDir,
		forceRebuild: os.Getenv("FORCE_REBUILD") == "true",
		skipCheck:    os.Getenv("SKIP_BUILD_CHECK") == "true",
		logger:       logger,
		debugEnabled: debugEnabled,
	}
}

// ExecuteBuild implements a three-tier build strategy:
// 1. Force rebuild: When explicitly requested, performs full clean + build
// 2. Cache hit: When artifacts are up-to-date, skips build entirely
// 3. Incremental rebuild: When cache is stale, uses forge build without clean
func (m *SmartBuildManager) ExecuteBuild(ctx context.Context) error {
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

// shouldSkipBuild determines if the build should be skipped entirely
func (m *SmartBuildManager) shouldSkipBuild(ctx context.Context) (bool, error) {
	// Never skip if force rebuild is requested
	if m.forceRebuild {
		return false, nil
	}

	// Never skip if cache check is disabled
	if m.skipCheck {
		return false, nil
	}

	// Check cache validity
	return m.validator.IsCacheValid(ctx)
}

// performCleanBuild executes a full clean and rebuild process
func (m *SmartBuildManager) performCleanBuild(ctx context.Context) error {
	m.logger.Printf("Performing clean build...")

	// Execute forge clean
	cleanCmd := exec.CommandContext(ctx, "forge", "clean")
	cleanCmd.Dir = m.contractsDir
	cleanCmd.Stdout = os.Stdout
	cleanCmd.Stderr = os.Stderr

	if err := cleanCmd.Run(); err != nil {
		m.logger.Printf("Warning: forge clean failed: %v", err)
		// Continue with build even if clean fails
	}

	return m.performBuild(ctx)
}

// performIncrementalBuild executes an incremental rebuild without cleaning
func (m *SmartBuildManager) performIncrementalBuild(ctx context.Context) error {
	m.logger.Printf("Performing incremental build...")
	return m.performBuild(ctx)
}

// performBuild executes the actual build command
func (m *SmartBuildManager) performBuild(ctx context.Context) error {
	return m.performBuildWithErrorHandling(ctx)
}

// SetForceRebuild allows overriding the force rebuild setting
func (m *SmartBuildManager) SetForceRebuild(force bool) {
	m.forceRebuild = force
}

// SetSkipCheck allows overriding the skip check setting
func (m *SmartBuildManager) SetSkipCheck(skip bool) {
	m.skipCheck = skip
}

// GetCacheStatus returns the current cache status
func (m *SmartBuildManager) GetCacheStatus(ctx context.Context) (*CacheStatus, error) {
	return m.validator.GetCacheStatus(ctx)
}

// debugLog logs debug messages if debug logging is enabled
func (m *SmartBuildManager) debugLog(format string, args ...interface{}) {
	if m.debugEnabled {
		m.logger.Printf("[DEBUG] "+format, args...)
	}
}

// validateBuildEnvironment checks if required build tools are available
func (m *SmartBuildManager) validateBuildEnvironment() error {
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

	// Check if required build tools are available
	if err := m.checkBuildTool("forge"); err != nil {
		return fmt.Errorf("forge tool validation failed: %w", err)
	}

	if err := m.checkBuildTool("just"); err != nil {
		return fmt.Errorf("just tool validation failed: %w", err)
	}

	m.debugLog("Build environment validation successful")
	return nil
}

// checkBuildTool verifies that a required build tool is available
func (m *SmartBuildManager) checkBuildTool(tool string) error {
	m.debugLog("Checking availability of build tool: %s", tool)

	_, err := exec.LookPath(tool)
	if err != nil {
		return fmt.Errorf("required build tool '%s' not found in PATH: %w", tool, err)
	}

	m.debugLog("Build tool %s is available", tool)
	return nil
}

// validateCacheWithFallback performs cache validation with graceful error handling
func (m *SmartBuildManager) validateCacheWithFallback(ctx context.Context) (bool, error) {
	m.debugLog("Validating cache with fallback handling")

	cacheValid, err := m.validator.IsCacheValid(ctx)
	if err != nil {
		m.logger.Printf("Cache validation encountered error: %v", err)

		// Try to get more detailed status information for better error reporting
		if status, statusErr := m.validator.GetCacheStatus(ctx); statusErr == nil {
			m.debugLog("Cache status details - Valid: %v, Reason: %s, Missing artifacts: %v",
				status.IsValid, status.Reason, status.MissingArtifacts)
		}

		// Return the original error - caller will decide on fallback strategy
		return false, fmt.Errorf("cache validation failed: %w", err)
	}

	m.debugLog("Cache validation completed successfully, valid: %v", cacheValid)
	return cacheValid, nil
}

// performCleanBuildWithFallback executes a clean build with comprehensive error handling
func (m *SmartBuildManager) performCleanBuildWithFallback(ctx context.Context) error {
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

// performIncrementalBuildWithFallback executes an incremental build with comprehensive error handling
func (m *SmartBuildManager) performIncrementalBuildWithFallback(ctx context.Context) error {
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

// performBuildWithErrorHandling executes the build command with comprehensive error handling
// It runs "just forge-build"
func (m *SmartBuildManager) performBuildWithErrorHandling(ctx context.Context) error {
	m.debugLog("Executing build command with error handling")

	buildCmd := exec.CommandContext(ctx, "just", "forge-build")
	buildCmd.Dir = m.contractsDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

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
