package buildcache

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BuildCacheValidator determines if contract artifacts are up-to-date relative to source files
type BuildCacheValidator struct {
	contractsDir string
	logger       *log.Logger
	debugEnabled bool
}

// CacheStatus represents the current state of the build cache
type CacheStatus struct {
	IsValid          bool
	Reason           string
	NewestSource     time.Time
	OldestArtifact   time.Time
	MissingArtifacts bool
}

// NewBuildCacheValidator creates a new BuildCacheValidator instance
func NewBuildCacheValidator(contractsDir string, logger *log.Logger) *BuildCacheValidator {
	if logger == nil {
		logger = log.New(os.Stdout, "[BUILD-CACHE] ", log.LstdFlags)
	}

	// Enable debug logging if BUILD_CACHE_DEBUG environment variable is set
	debugEnabled := os.Getenv("BUILD_CACHE_DEBUG") == "true"

	return &BuildCacheValidator{
		contractsDir: contractsDir,
		logger:       logger,
		debugEnabled: debugEnabled,
	}
}

// IsCacheValid determines if the build cache is valid by comparing source and artifact timestamps
func (v *BuildCacheValidator) IsCacheValid(ctx context.Context) (bool, error) {
	status, err := v.GetCacheStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.IsValid, nil
}

// GetCacheStatus provides detailed information about the cache state
func (v *BuildCacheValidator) GetCacheStatus(ctx context.Context) (*CacheStatus, error) {
	v.debugLog("Starting cache validation for contracts directory: %s", v.contractsDir)
	status := &CacheStatus{
		IsValid: true, // Start with valid assumption, set to false if issues found
		Reason:  "all artifacts are up-to-date",
	}

	// Validate contracts directory exists and is accessible
	if err := v.validateContractsDirectory(); err != nil {
		v.logger.Printf("Contracts directory validation failed: %v", err)
		return nil, fmt.Errorf("contracts directory validation failed: %w", err)
	}

	// Check if artifacts directory exists
	artifactsDir := filepath.Join(v.contractsDir, "forge-artifacts")
	if err := v.validateArtifactsDirectory(artifactsDir, status); err != nil {
		if !status.IsValid {
			// This is an expected condition (missing artifacts), not an error
			v.debugLog("Artifacts directory validation resulted in invalid cache: %s", status.Reason)
			return status, nil
		}
		return nil, fmt.Errorf("artifacts directory validation failed: %w", err)
	}

	// If artifacts directory validation already marked cache as invalid, return early
	if !status.IsValid {
		return status, nil
	}

	// Check if artifacts directory has JSON files (indicating successful build)
	hasArtifacts, err := v.hasArtifactFiles(artifactsDir)
	if err != nil {
		v.logger.Printf("Error checking artifact files: %v", err)
		return nil, fmt.Errorf("failed to check artifact files: %w", err)
	}
	if !hasArtifacts {
		v.logger.Printf("forge-artifacts directory is empty or has no JSON files")
		status.IsValid = false
		status.Reason = "forge-artifacts directory is empty or has no JSON files"
		status.MissingArtifacts = true
		return status, nil
	}

	// Compare source vs artifact timestamps with comprehensive error handling
	newestSource, err := v.getNewestSourceTime()
	if err != nil {
		v.logger.Printf("Error getting newest source time: %v", err)
		return nil, fmt.Errorf("failed to get newest source time: %w", err)
	}
	status.NewestSource = newestSource
	v.debugLog("Newest source file timestamp: %v", newestSource.Format(time.RFC3339))

	oldestArtifact, err := v.getOldestArtifactTime(artifactsDir)
	if err != nil {
		v.logger.Printf("Error getting oldest artifact time: %v", err)
		return nil, fmt.Errorf("failed to get oldest artifact time: %w", err)
	}
	status.OldestArtifact = oldestArtifact
	v.debugLog("Oldest artifact timestamp: %v", oldestArtifact.Format(time.RFC3339))

	// If source files are newer than artifacts, cache is invalid
	if newestSource.After(oldestArtifact) {
		v.logger.Printf("Source files are newer than artifacts (source: %v, artifacts: %v)",
			newestSource.Format(time.RFC3339), oldestArtifact.Format(time.RFC3339))
		status.IsValid = false
		status.Reason = "source files are newer than artifacts"
		return status, nil
	} else {
		v.debugLog("Source files are not newer than artifacts (source: %v, artifacts: %v)",
			newestSource.Format(time.RFC3339), oldestArtifact.Format(time.RFC3339))
	}

	// Check if foundry.toml is newer than artifacts with error handling
	if err := v.validateFoundryConfig(status, oldestArtifact); err != nil {
		v.logger.Printf("Error validating foundry config: %v", err)
		return nil, fmt.Errorf("failed to validate foundry config: %w", err)
	}

	if !status.IsValid {
		return status, nil
	}

	v.logger.Printf("Build cache is valid")
	return status, nil
}

// hasArtifactFiles checks if the artifacts directory contains JSON files
func (v *BuildCacheValidator) hasArtifactFiles(artifactsDir string) (bool, error) {
	v.debugLog("Checking for artifact files in: %s", artifactsDir)

	// Check if the directory exists first
	if _, err := os.Stat(artifactsDir); err != nil {
		if os.IsNotExist(err) {
			v.debugLog("Artifacts directory does not exist: %s", artifactsDir)
			return false, nil // This is expected, not an error
		}
		if os.IsPermission(err) {
			v.logger.Printf("Warning: Permission denied accessing artifacts directory: %s", artifactsDir)
			return false, nil // Treat as no artifacts available
		}
		return false, fmt.Errorf("error accessing artifacts directory %s: %w", artifactsDir, err)
	}

	var hasJSON bool
	err := filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			v.debugLog("Error walking artifacts path %s: %v", path, err)
			if os.IsPermission(err) {
				v.logger.Printf("Warning: Permission denied accessing %s, skipping", path)
				return nil // Skip this file/directory but continue walking
			}
			return fmt.Errorf("error walking artifacts directory at %s: %w", path, err)
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			v.debugLog("Found artifact file: %s", path)
			hasJSON = true
			return filepath.SkipDir // We found at least one JSON file, no need to continue
		}
		return nil
	})

	v.debugLog("Artifact files check completed, has JSON files: %v", hasJSON)
	return hasJSON, err
}

// getNewestSourceTime finds the newest modification time among all source files
func (v *BuildCacheValidator) getNewestSourceTime() (time.Time, error) {
	v.debugLog("Getting newest source file timestamp")

	var newestTime time.Time
	srcDir := filepath.Join(v.contractsDir, "src")

	// Check if source directory exists first
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, fmt.Errorf("source directory does not exist: %s", srcDir)
		}
		if os.IsPermission(err) {
			return time.Time{}, fmt.Errorf("permission denied accessing source directory: %s", srcDir)
		}
		return time.Time{}, fmt.Errorf("error accessing source directory %s: %w", srcDir, err)
	}

	var sourceFileCount int
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			v.debugLog("Error walking path %s: %v", path, err)
			if os.IsPermission(err) {
				v.logger.Printf("Warning: Permission denied accessing %s, skipping", path)
				return nil // Skip this file/directory but continue walking
			}
			return fmt.Errorf("error walking source directory at %s: %w", path, err)
		}

		if !info.IsDir() && v.isSourceFile(info.Name()) {
			sourceFileCount++
			v.debugLog("Found source file: %s (modified: %v)", path, info.ModTime().Format(time.RFC3339))
			if info.ModTime().After(newestTime) {
				newestTime = info.ModTime()
			}
		}
		return nil
	})

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to walk source directory: %w", err)
	}

	if sourceFileCount == 0 {
		return time.Time{}, fmt.Errorf("no source files found in %s", srcDir)
	}

	v.debugLog("Found %d source files, newest timestamp: %v", sourceFileCount, newestTime.Format(time.RFC3339))
	return newestTime, nil
}

// getOldestArtifactTime finds the oldest modification time among all artifact files
func (v *BuildCacheValidator) getOldestArtifactTime(artifactsDir string) (time.Time, error) {
	v.debugLog("Getting oldest artifact timestamp")

	var oldestTime time.Time
	first := true
	var artifactFileCount int

	err := filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			v.debugLog("Error walking artifacts path %s: %v", path, err)
			if os.IsPermission(err) {
				v.logger.Printf("Warning: Permission denied accessing %s, skipping", path)
				return nil // Skip this file/directory but continue walking
			}
			return fmt.Errorf("error walking artifacts directory at %s: %w", path, err)
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			artifactFileCount++
			v.debugLog("Found artifact file: %s (modified: %v)", path, info.ModTime().Format(time.RFC3339))
			if first || info.ModTime().Before(oldestTime) {
				oldestTime = info.ModTime()
				first = false
			}
		}
		return nil
	})

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to walk artifacts directory: %w", err)
	}

	if artifactFileCount == 0 {
		return time.Time{}, fmt.Errorf("no artifact files found in %s", artifactsDir)
	}

	v.debugLog("Found %d artifact files, oldest timestamp: %v", artifactFileCount, oldestTime.Format(time.RFC3339))
	return oldestTime, nil
}

// isSourceFile determines if a file is a Solidity source file
func (v *BuildCacheValidator) isSourceFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".sol"
}

// debugLog logs debug messages if debug logging is enabled
func (v *BuildCacheValidator) debugLog(format string, args ...interface{}) {
	if v.debugEnabled {
		v.logger.Printf("[DEBUG] "+format, args...)
	}
}

// validateContractsDirectory ensures the contracts directory exists and is accessible
func (v *BuildCacheValidator) validateContractsDirectory() error {
	v.debugLog("Validating contracts directory: %s", v.contractsDir)

	info, err := os.Stat(v.contractsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("contracts directory does not exist: %s", v.contractsDir)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing contracts directory: %s", v.contractsDir)
		}
		return fmt.Errorf("error accessing contracts directory %s: %w", v.contractsDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("contracts path is not a directory: %s", v.contractsDir)
	}

	v.debugLog("Contracts directory validation successful")
	return nil
}

// validateArtifactsDirectory checks if artifacts directory exists and is accessible
func (v *BuildCacheValidator) validateArtifactsDirectory(artifactsDir string, status *CacheStatus) error {
	v.debugLog("Validating artifacts directory: %s", artifactsDir)

	info, err := os.Stat(artifactsDir)
	if err != nil {
		if os.IsNotExist(err) {
			v.logger.Printf("forge-artifacts directory does not exist: %s", artifactsDir)
			status.IsValid = false
			status.Reason = "forge-artifacts directory does not exist"
			status.MissingArtifacts = true
			return nil // This is expected, not an error
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing artifacts directory: %s", artifactsDir)
		}
		return fmt.Errorf("error accessing artifacts directory %s: %w", artifactsDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("artifacts path is not a directory: %s", artifactsDir)
	}

	v.debugLog("Artifacts directory validation successful")
	return nil
}

// validateFoundryConfig checks if foundry.toml is newer than artifacts
func (v *BuildCacheValidator) validateFoundryConfig(status *CacheStatus, oldestArtifact time.Time) error {
	foundryPath := filepath.Join(v.contractsDir, "foundry.toml")
	v.debugLog("Checking foundry.toml at: %s", foundryPath)

	foundryInfo, err := os.Stat(foundryPath)
	if err != nil {
		if os.IsNotExist(err) {
			v.debugLog("foundry.toml not found, skipping config validation")
			return nil // foundry.toml is optional
		}
		if os.IsPermission(err) {
			v.logger.Printf("Warning: Permission denied accessing foundry.toml: %s", foundryPath)
			return nil // Treat as non-critical error
		}
		return fmt.Errorf("error accessing foundry.toml at %s: %w", foundryPath, err)
	}

	v.debugLog("foundry.toml timestamp: %v", foundryInfo.ModTime().Format(time.RFC3339))

	if foundryInfo.ModTime().After(oldestArtifact) {
		v.logger.Printf("foundry.toml is newer than artifacts (foundry: %v, artifacts: %v)",
			foundryInfo.ModTime().Format(time.RFC3339), oldestArtifact.Format(time.RFC3339))
		status.IsValid = false
		status.Reason = "foundry.toml is newer than artifacts"
	} else {
		v.debugLog("foundry.toml is not newer than artifacts (foundry: %v, artifacts: %v)",
			foundryInfo.ModTime().Format(time.RFC3339), oldestArtifact.Format(time.RFC3339))
	}

	return nil
}
