package buildcache_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/buildcache"
)

// ExampleBuildCacheValidator demonstrates how to use the BuildCacheValidator
func ExampleBuildCacheValidator() {
	// Create a logger (optional, will use default if nil)
	logger := log.New(os.Stdout, "[BUILD-CACHE] ", log.LstdFlags)

	// Create validator for contracts directory
	contractsDir := "/path/to/packages/contracts-bedrock"
	validator := buildcache.NewBuildCacheValidator(contractsDir, logger)

	// Check if cache is valid
	ctx := context.Background()
	isValid, err := validator.IsCacheValid(ctx)
	if err != nil {
		fmt.Printf("Error checking cache: %v\n", err)
		return
	}

	if isValid {
		fmt.Println("Build cache is valid, skipping rebuild")
	} else {
		fmt.Println("Build cache is stale, rebuild needed")
	}

	// Get detailed status information
	status, err := validator.GetCacheStatus(ctx)
	if err != nil {
		fmt.Printf("Error getting cache status: %v\n", err)
		return
	}

	fmt.Printf("Cache valid: %v\n", status.IsValid)
	fmt.Printf("Reason: %s\n", status.Reason)
	if !status.NewestSource.IsZero() {
		fmt.Printf("Newest source: %v\n", status.NewestSource)
	}
	if !status.OldestArtifact.IsZero() {
		fmt.Printf("Oldest artifact: %v\n", status.OldestArtifact)
	}
}

// ExampleSmartBuildManager demonstrates how to use the SmartBuildManager
func ExampleSmartBuildManager() {
	// Create a logger (optional, will use default if nil)
	logger := log.New(os.Stdout, "[BUILD-MANAGER] ", log.LstdFlags)

	// Create build manager for contracts directory
	contractsDir := "/path/to/packages/contracts-bedrock"
	manager := buildcache.NewSmartBuildManager(contractsDir, logger)

	// Execute smart build (will automatically determine the best strategy)
	ctx := context.Background()
	err := manager.ExecuteBuild(ctx)
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
		return
	}

	fmt.Println("Build completed successfully")
}

// ExampleSmartBuildManager_withConfiguration demonstrates configuration options
func ExampleSmartBuildManager_withConfiguration() {
	logger := log.New(os.Stdout, "[BUILD-MANAGER] ", log.LstdFlags)
	contractsDir := "/path/to/packages/contracts-bedrock"
	manager := buildcache.NewSmartBuildManager(contractsDir, logger)

	// Override settings programmatically
	manager.SetForceRebuild(true) // Force a clean rebuild
	manager.SetSkipCheck(false)   // Enable cache checking

	ctx := context.Background()

	// Environment variables can also be used:
	// FORCE_REBUILD=true - forces a clean rebuild
	// SKIP_BUILD_CHECK=true - skips cache validation (useful for CI)
	fmt.Println("Environment variables FORCE_REBUILD and SKIP_BUILD_CHECK are supported")

	// Get detailed cache status
	status, err := manager.GetCacheStatus(ctx)
	if err != nil {
		fmt.Printf("Error getting cache status: %v\n", err)
		return
	}

	fmt.Printf("Cache status: %s\n", status.Reason)
}
