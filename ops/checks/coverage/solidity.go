package coverage

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// Collect runs forge coverage for a single test file and parses the debug output.
//
// We use --report debug (not lcov) because forge's LCOV output has a bug with
// multi-solc compilation: when a file (e.g. L1Block.sol) is compiled by multiple
// solc versions, hits can land on one source ID while anchors exist for another.
// The LCOV reporter only emits hits for the anchored source ID, losing data.
//
// The debug output shows all coverage items grouped by file path, with hit counts
// per (source ID, line range). We merge hits by file path to get accurate coverage.
//
// The profile argument sets environment variables (e.g. SYS_FEATURE__CUSTOM_GAS_TOKEN=true)
// so we can collect coverage under different feature flag combinations.
func (c *SolidityCollector) Collect(rootDir string, testPath string, profile Profile) (*Report, error) {
	contractsDir := filepath.Join(rootDir, c.ContractsDir)

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("checks-coverage-%d.debug", os.Getpid()))
	defer os.Remove(tmpFile)

	// Run forge coverage with debug report. We capture stdout to a file
	// because forge writes the debug report to stdout (not the --report-file path).
	out, err := os.Create(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer out.Close()

	cmd := exec.Command("forge", "coverage",
		"--report", "debug",
		"--match-path", testPath,
	)
	cmd.Dir = contractsDir
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	// Set env vars for the profile (starts from current env so forge finds its tools)
	cmd.Env = os.Environ()
	for k, v := range profile.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	_ = cmd.Run() // tolerate test failures — coverage for passing tests is still valid

	out.Close()

	covers, err := parseDebugFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("parsing debug output (forge may have failed): %w", err)
	}

	return &Report{
		Test:     testPath,
		Language: "solidity",
		Profile:  profile.Name,
		Covers:   covers,
	}, nil
}

// debugLineRegex matches a "- Line (location: (source ID: N, lines: A..B, ...), hits: X) -> ..." entry.
// We only care about Line items — Statement/Branch/Function add noise and duplicate line-level info.
var debugLineRegex = regexp.MustCompile(`^- Line \(location: \(source ID: (\d+), lines: (\d+)\.\.(\d+), [^)]*\), hits: (\d+)\)`)

// debugFileHeaderRegex matches a file path header like "scripts/Artifacts.s.sol:" which
// appears at the start of each block of coverage items for a file.
var debugFileHeaderRegex = regexp.MustCompile(`^(\S+\.sol):$`)

// parseDebug parses forge coverage --report debug output into the common format.
// It groups hits by file path, merging across source IDs (fixing the multi-solc bug).
func parseDebug(data string) map[string][][2]int {
	// For each file, track which lines were hit (any source ID).
	fileLines := make(map[string]map[int]bool)

	var currentFile string

	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // forge output can be very large
	for scanner.Scan() {
		line := scanner.Text()

		// File header resets the current file context
		if m := debugFileHeaderRegex.FindStringSubmatch(line); m != nil {
			currentFile = m[1]
			if _, ok := fileLines[currentFile]; !ok {
				fileLines[currentFile] = make(map[int]bool)
			}
			continue
		}

		if currentFile == "" {
			continue
		}

		// Line entry within the current file
		if m := debugLineRegex.FindStringSubmatch(line); m != nil {
			startLine, err1 := strconv.Atoi(m[2])
			endLine, err2 := strconv.Atoi(m[3])
			hits, err3 := strconv.Atoi(m[4])
			if err1 != nil || err2 != nil || err3 != nil || hits == 0 {
				continue
			}
			lines := fileLines[currentFile]
			for l := startLine; l <= endLine; l++ {
				lines[l] = true
			}
		}
	}

	covers := make(map[string][][2]int)
	for file, lines := range fileLines {
		if len(lines) == 0 {
			continue
		}
		sortedLines := make([]int, 0, len(lines))
		for l := range lines {
			sortedLines = append(sortedLines, l)
		}
		covers[file] = compactRanges(sortedLines)
	}

	return covers
}

// parseDebugFile reads a forge debug output file from disk.
func parseDebugFile(path string) (map[string][][2]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseDebug(string(data)), nil
}

// parseLCOV parses LCOV format coverage data into the common format.
// Kept for backward compatibility with existing LCOV-format data (e.g. from Rust).
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
