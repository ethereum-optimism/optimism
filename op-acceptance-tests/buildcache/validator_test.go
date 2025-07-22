package buildcache

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildCacheValidator_IsCacheValid_MissingArtifacts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildcache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := log.New(io.Discard, "[TEST] ", log.LstdFlags)
	validator := NewBuildCacheValidator(tempDir, logger)

	ctx := context.Background()
	valid, err := validator.IsCacheValid(ctx)
	if err != nil {
		t.Fatalf("IsCacheValid() error = %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid when artifacts directory is missing")
	}
}

func TestBuildCacheValidator_IsCacheValid_StaleArtifacts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildcache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	artifactsDir := filepath.Join(tempDir, "forge-artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(artifactsDir, "test.json")
	if err := os.WriteFile(artifactPath, []byte(`{"artifact": "data"}`), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := log.New(io.Discard, "[TEST] ", log.LstdFlags)
	validator := NewBuildCacheValidator(tempDir, logger)

	ctx := context.Background()
	valid, err := validator.IsCacheValid(ctx)
	if err != nil {
		t.Fatalf("IsCacheValid() error = %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid when source files are newer than artifacts")
	}
}

func TestBuildCacheValidator_NoSourceDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildcache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := log.New(io.Discard, "[TEST] ", log.LstdFlags)
	validator := NewBuildCacheValidator(tempDir, logger)

	_, err = validator.getNewestSourceTime()
	if err == nil {
		t.Error("getNewestSourceTime() should return error when src directory does not exist")
	}

	if !strings.Contains(err.Error(), "source directory does not exist") {
		t.Errorf("Expected error about source directory not existing, got: %v", err)
	}
}

func TestBuildCacheValidator_IsCacheValid_UpToDateArtifacts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildcache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "test.sol"), []byte("contract Test {}"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	artifactsDir := filepath.Join(tempDir, "forge-artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(artifactsDir, "test.json")
	if err := os.WriteFile(artifactPath, []byte(`{"artifact": "data"}`), 0644); err != nil {
		t.Fatal(err)
	}

	logger := log.New(io.Discard, "[TEST] ", log.LstdFlags)
	validator := NewBuildCacheValidator(tempDir, logger)

	ctx := context.Background()
	valid, err := validator.IsCacheValid(ctx)
	if err != nil {
		t.Fatalf("IsCacheValid() error = %v", err)
	}

	if !valid {
		t.Error("Expected cache to be valid when artifacts are newer than source files")
	}
}

func TestBuildCacheValidator_CrossPlatformTimestamps(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "buildcache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	artifactsDir := filepath.Join(tempDir, "forge-artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatal(err)
	}

	baseTime := time.Now().Truncate(time.Second)

	tests := []struct {
		name          string
		sourceTime    time.Time
		artifactTime  time.Time
		expectedValid bool
	}{
		{
			name:          "artifacts exactly 1 second newer",
			sourceTime:    baseTime,
			artifactTime:  baseTime.Add(1 * time.Second),
			expectedValid: true,
		},
		{
			name:          "source exactly 1 millisecond newer",
			sourceTime:    baseTime.Add(1 * time.Millisecond),
			artifactTime:  baseTime,
			expectedValid: false,
		},
		{
			name:          "same timestamp",
			sourceTime:    baseTime,
			artifactTime:  baseTime,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := filepath.Join(srcDir, "test.sol")
			if err := os.WriteFile(sourceFile, []byte("contract Test {}"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(sourceFile, tt.sourceTime, tt.sourceTime); err != nil {
				t.Fatal(err)
			}

			artifactFile := filepath.Join(artifactsDir, "test.json")
			if err := os.WriteFile(artifactFile, []byte(`{"artifact": "data"}`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(artifactFile, tt.artifactTime, tt.artifactTime); err != nil {
				t.Fatal(err)
			}

			logger := log.New(io.Discard, "[TEST] ", log.LstdFlags)
			validator := NewBuildCacheValidator(tempDir, logger)

			ctx := context.Background()
			valid, err := validator.IsCacheValid(ctx)
			if err != nil {
				t.Fatalf("IsCacheValid() error = %v", err)
			}

			if valid != tt.expectedValid {
				t.Errorf("IsCacheValid() = %v, want %v (source: %v, artifact: %v)",
					valid, tt.expectedValid, tt.sourceTime, tt.artifactTime)
			}

			os.Remove(sourceFile)
			os.Remove(artifactFile)
		})
	}
}
