package buildcache

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSmartBuildManager_ExecuteBuild(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tempDir string)
		forceRebuild   bool
		skipCheck      bool
		expectedAction string // "skip", "incremental", "clean"
	}{
		{
			name: "force rebuild ignores cache state",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild:   true,
			skipCheck:      false,
			expectedAction: "clean",
		},
		{
			name: "skip check performs clean build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild:   false,
			skipCheck:      true,
			expectedAction: "clean",
		},
		{
			name: "valid cache skips build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupValidCache(t, tempDir)
			},
			forceRebuild:   false,
			skipCheck:      false,
			expectedAction: "skip",
		},
		{
			name: "invalid cache triggers incremental build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupInvalidCache(t, tempDir)
			},
			forceRebuild:   false,
			skipCheck:      false,
			expectedAction: "incremental",
		},
		{
			name: "missing artifacts triggers incremental build",
			setupFunc: func(t *testing.T, tempDir string) {
				setupMissingArtifacts(t, tempDir)
			},
			forceRebuild:   false,
			skipCheck:      false,
			expectedAction: "incremental",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			tt.setupFunc(t, tempDir)

			// Create manager
			logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
			manager := NewSmartBuildManager(tempDir, logger)
			manager.SetForceRebuild(tt.forceRebuild)
			manager.SetSkipCheck(tt.skipCheck)

			// Test shouldSkipBuild to verify logic
			ctx := context.Background()
			shouldSkip, err := manager.shouldSkipBuild(ctx)
			if err != nil {
				t.Fatalf("shouldSkipBuild() error = %v", err)
			}

			expectedSkip := tt.expectedAction == "skip"
			if shouldSkip != expectedSkip {
				t.Errorf("shouldSkipBuild() = %v, want %v", shouldSkip, expectedSkip)
			}

			// Note: We don't test ExecuteBuild() directly because it would try to run
			// actual forge and just commands. In a real integration test environment,
			// we would mock these commands or use a test environment with the tools installed.
		})
	}
}

func TestSmartBuildManager_EnvironmentVariables(t *testing.T) {
	// Test FORCE_REBUILD environment variable
	t.Run("FORCE_REBUILD=true", func(t *testing.T) {
		os.Setenv("FORCE_REBUILD", "true")
		defer os.Unsetenv("FORCE_REBUILD")

		tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		manager := NewSmartBuildManager(tempDir, logger)

		if !manager.forceRebuild {
			t.Error("Expected forceRebuild to be true when FORCE_REBUILD=true")
		}
	})

	// Test SKIP_BUILD_CHECK environment variable
	t.Run("SKIP_BUILD_CHECK=true", func(t *testing.T) {
		os.Setenv("SKIP_BUILD_CHECK", "true")
		defer os.Unsetenv("SKIP_BUILD_CHECK")

		tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		manager := NewSmartBuildManager(tempDir, logger)

		if !manager.skipCheck {
			t.Error("Expected skipCheck to be true when SKIP_BUILD_CHECK=true")
		}
	})

	// Test default values when environment variables are not set
	t.Run("default values", func(t *testing.T) {
		os.Unsetenv("FORCE_REBUILD")
		os.Unsetenv("SKIP_BUILD_CHECK")

		tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		manager := NewSmartBuildManager(tempDir, logger)

		if manager.forceRebuild {
			t.Error("Expected forceRebuild to be false by default")
		}
		if manager.skipCheck {
			t.Error("Expected skipCheck to be false by default")
		}
	})
}

func TestSmartBuildManager_SettersAndGetters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewSmartBuildManager(tempDir, logger)

	// Test SetForceRebuild
	manager.SetForceRebuild(true)
	if !manager.forceRebuild {
		t.Error("SetForceRebuild(true) did not set forceRebuild to true")
	}

	manager.SetForceRebuild(false)
	if manager.forceRebuild {
		t.Error("SetForceRebuild(false) did not set forceRebuild to false")
	}

	// Test SetSkipCheck
	manager.SetSkipCheck(true)
	if !manager.skipCheck {
		t.Error("SetSkipCheck(true) did not set skipCheck to true")
	}

	manager.SetSkipCheck(false)
	if manager.skipCheck {
		t.Error("SetSkipCheck(false) did not set skipCheck to false")
	}
}

func TestSmartBuildManager_GetCacheStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildmanager_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Setup valid cache
	setupValidCache(t, tempDir)

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewSmartBuildManager(tempDir, logger)

	ctx := context.Background()
	status, err := manager.GetCacheStatus(ctx)
	if err != nil {
		t.Fatalf("GetCacheStatus() error = %v", err)
	}

	if !status.IsValid {
		t.Error("Expected cache status to be valid")
	}

	if status.Reason != "all artifacts are up-to-date" {
		t.Errorf("Expected reason to be 'all artifacts are up-to-date', got %s", status.Reason)
	}
}

// Helper functions for test setup

func setupValidCache(t *testing.T, tempDir string) {
	// Create src directory with source file first
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create foundry.toml
	foundryPath := filepath.Join(tempDir, "foundry.toml")
	if err := os.WriteFile(foundryPath, []byte("[profile.default]"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sleep to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create newer artifacts
	artifactsDir := filepath.Join(tempDir, "forge-artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(artifactsDir, "test.json")
	if err := os.WriteFile(artifactPath, []byte(`{"artifact": "data"}`), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupInvalidCache(t *testing.T, tempDir string) {
	// Create artifacts first (older)
	artifactsDir := filepath.Join(tempDir, "forge-artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(artifactsDir, "test.json")
	if err := os.WriteFile(artifactPath, []byte(`{"artifact": "data"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Sleep to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create src directory with newer source file
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupMissingArtifacts(t *testing.T, tempDir string) {
	// Create src directory with source file
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Don't create forge-artifacts directory
}
