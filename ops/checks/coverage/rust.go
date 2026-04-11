package coverage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RustCollector collects coverage from Rust tests using cargo-llvm-cov.
type RustCollector struct {
	// WorkspaceDir is the path to the Rust workspace relative to rootDir.
	// Default: "rust"
	WorkspaceDir string
}

func NewRustCollector() *RustCollector {
	return &RustCollector{WorkspaceDir: "rust"}
}

func (c *RustCollector) Language() string { return "rust" }

// Collect runs cargo llvm-cov for a specific crate/test and parses LCOV output.
// testPath is a crate name (e.g. "kona-derive") or test binary name.
func (c *RustCollector) Collect(rootDir string, testPath string) (*Report, error) {
	workDir := filepath.Join(rootDir, c.WorkspaceDir)

	// Check if cargo-llvm-cov is available
	if _, err := exec.LookPath("cargo-llvm-cov"); err != nil {
		return nil, fmt.Errorf("cargo-llvm-cov not found: install with 'cargo install cargo-llvm-cov'")
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("checks-rust-coverage-%d.lcov", os.Getpid()))
	defer os.Remove(tmpFile)

	// Run cargo llvm-cov for the specific package, output as LCOV
	cmd := exec.Command("cargo", "llvm-cov",
		"--lcov",
		"--output-path", tmpFile,
		"--package", testPath,
	)
	cmd.Dir = workDir
	cmd.Stderr = os.Stderr

	// Tolerate test failures
	_ = cmd.Run()

	covers, err := parseLCOVFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("parsing LCOV output (cargo may have failed): %w", err)
	}

	return &Report{
		Test:     testPath,
		Language: "rust",
		Covers:   covers,
	}, nil
}
