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

// GoCollector collects coverage from Go tests.
type GoCollector struct{}

func NewGoCollector() *GoCollector { return &GoCollector{} }

func (c *GoCollector) Language() string { return "go" }

// Collect runs go test with coverage for a single package and parses the profile.
// testPath is a Go package path like "./op-node/rollup/derive/..."
func (c *GoCollector) Collect(rootDir string, testPath string) (*Report, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("checks-go-coverage-%d.prof", os.Getpid()))
	defer os.Remove(tmpFile)

	cmd := exec.Command("go", "test",
		"-coverprofile", tmpFile,
		"-covermode", "set",
		"-short",
		testPath,
	)
	cmd.Dir = rootDir
	cmd.Stderr = os.Stderr

	// Tolerate test failures — coverage is still produced for passing tests
	_ = cmd.Run()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("reading coverprofile (go test may have failed): %w", err)
	}

	covers := parseGoCoverprofile(string(data))

	return &Report{
		Test:     testPath,
		Language: "go",
		Covers:   covers,
	}, nil
}

// parseGoCoverprofile parses Go coverage profile format.
// Format: mode: set
//
//	file:startLine.startCol,endLine.endCol statements count
func parseGoCoverprofile(data string) map[string][][2]int {
	covers := make(map[string][][2]int)
	fileLines := make(map[string][]int)

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || line == "" {
			continue
		}

		colonIdx := strings.LastIndex(line, ":")
		if colonIdx < 0 {
			continue
		}
		filePath := line[:colonIdx]
		rest := line[colonIdx+1:]

		fields := strings.Fields(rest)
		if len(fields) < 3 {
			continue
		}

		// Only include lines where count > 0 (actually executed)
		if fields[2] == "0" {
			continue
		}

		rangeParts := strings.SplitN(fields[0], ",", 2)
		if len(rangeParts) != 2 {
			continue
		}

		startLine := parseLineNum(rangeParts[0])
		endLine := parseLineNum(rangeParts[1])

		if startLine > 0 && endLine > 0 {
			for l := startLine; l <= endLine; l++ {
				fileLines[filePath] = append(fileLines[filePath], l)
			}
		}
	}

	for file, lines := range fileLines {
		if len(lines) > 0 {
			covers[file] = compactRanges(lines)
		}
	}

	return covers
}

func parseLineNum(s string) int {
	dotIdx := strings.Index(s, ".")
	if dotIdx >= 0 {
		s = s[:dotIdx]
	}
	n, _ := strconv.Atoi(s)
	return n
}
