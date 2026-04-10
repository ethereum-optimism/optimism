package coverage

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SolidityCollector collects coverage from Forge tests.
type SolidityCollector struct {
	// ContractsDir is the path to the contracts directory relative to rootDir.
	// Default: "packages/contracts-bedrock"
	ContractsDir string
}

func NewSolidityCollector() *SolidityCollector {
	return &SolidityCollector{ContractsDir: "packages/contracts-bedrock"}
}

func (c *SolidityCollector) Language() string { return "solidity" }

// Collect runs forge coverage for a single test file and parses LCOV output.
func (c *SolidityCollector) Collect(rootDir string, testPath string) (*Report, error) {
	contractsDir := filepath.Join(rootDir, c.ContractsDir)

	// Create temp file for LCOV output
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("checks-coverage-%d.lcov", os.Getpid()))
	defer os.Remove(tmpFile)

	// Run forge coverage for the specific test file, writing LCOV to file
	cmd := exec.Command("forge", "coverage",
		"--report", "lcov",
		"--report-file", tmpFile,
		"--match-path", testPath,
	)
	cmd.Dir = contractsDir
	cmd.Stderr = os.Stderr

	// Run forge coverage — tolerate test failures since coverage is still
	// produced for passing tests. The LCOV file is what we care about.
	_ = cmd.Run()

	// Parse LCOV file (may be partial if some tests failed)
	covers, err := parseLCOVFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("parsing LCOV output (forge may have failed): %w", err)
	}

	return &Report{
		Test:     testPath,
		Language: "solidity",
		Covers:   covers,
	}, nil
}

// parseLCOV parses LCOV format coverage data into the common format.
func parseLCOV(data string) map[string][][2]int {
	covers := make(map[string][][2]int)

	var currentFile string
	var lines []int

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "SF:") {
			currentFile = strings.TrimPrefix(line, "SF:")
			lines = nil
		} else if strings.HasPrefix(line, "DA:") {
			parts := strings.SplitN(strings.TrimPrefix(line, "DA:"), ",", 2)
			if len(parts) == 2 && parts[1] != "0" {
				lineNum, err := strconv.Atoi(parts[0])
				if err == nil {
					lines = append(lines, lineNum)
				}
			}
		} else if line == "end_of_record" && currentFile != "" {
			if len(lines) > 0 {
				covers[currentFile] = compactRanges(lines)
			}
			currentFile = ""
			lines = nil
		}
	}

	return covers
}

// parseLCOVFile reads an LCOV file from disk.
func parseLCOVFile(path string) (map[string][][2]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseLCOV(string(data)), nil
}

// compactRanges converts a sorted list of line numbers into contiguous ranges.
// [1, 2, 3, 5, 6, 10] → [[1,3], [5,6], [10,10]]
func compactRanges(lines []int) [][2]int {
	if len(lines) == 0 {
		return nil
	}

	// Sort
	sorted := make([]int, len(lines))
	copy(sorted, lines)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	var ranges [][2]int
	start, end := sorted[0], sorted[0]
	for _, l := range sorted[1:] {
		if l == end+1 {
			end = l
		} else {
			ranges = append(ranges, [2]int{start, end})
			start, end = l, l
		}
	}
	ranges = append(ranges, [2]int{start, end})
	return ranges
}
