package main

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/scripts/checks/common"
)

// Patterns to detect contract types and pragma
var (
	// Matches "pragma solidity X.Y.Z;" (strict) vs "pragma solidity ^X.Y.Z;" or ">=X.Y.Z" (non-strict)
	pragmaPattern = regexp.MustCompile(`pragma\s+solidity\s+([^;]+);`)

	// Matches "contract Name" but not "abstract contract Name"
	// Uses \s* to allow indentation at start of line
	contractPattern = regexp.MustCompile(`(?m)^\s*contract\s+\w+`)

	// Matches "abstract contract Name"
	abstractPattern = regexp.MustCompile(`(?m)^\s*abstract\s+contract\s+\w+`)

	// Matches "library Name"
	libraryPattern = regexp.MustCompile(`(?m)^\s*library\s+\w+`)

	// Matches "interface Name"
	interfacePattern = regexp.MustCompile(`(?m)^\s*interface\s+\w+`)
)

// Files that are grandfathered in (already have non-strict pragma)
// These should be fixed over time, but we don't want to block CI on them
var excludedFiles = []string{
	"src/integration/EventLogger.sol",
	"src/integration/GameHelper.sol",
	"src/libraries/TransientContext.sol",
	"src/periphery/AssetReceiver.sol",
	"src/periphery/Transactor.sol",
	"src/periphery/monitoring/DisputeMonitorHelper.sol",
	"src/universal/SafeSend.sol",
}

// rangeCheckedFiles are the sources that forge compiles with op-deployer's scripts, and the
// interfaces that they import.
var rangeCheckedFiles = []string{"src/**/*.sol", "scripts/**/*.sol", "interfaces/**/*.sol"}

func main() {
	_, strictErr := common.ProcessFilesGlob(
		[]string{"src/**/*.sol"},
		excludedFiles,
		processFile,
	)
	if err := errors.Join(strictErr, checkPragmaRanges(rangeCheckedFiles)); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// checkPragmaRanges checks that each pragma range in the files allows a compiler version that a
// file of them pins with an exact pragma. op-deployer runs scripts on contracts built in advance
// and refuses to compile them. With forge 1.8.3, a build that has a file whose range allows no
// pinned version runs only where the solc that compiled the file is installed. Elsewhere, forge
// compiles the file again.
func checkPragmaRanges(includes []string) error {
	pragmas, err := common.ProcessFilesGlob(includes, nil, readPragmas)
	if err != nil {
		return err
	}
	errs := rangeErrors(pragmas)
	if len(errs) == 0 {
		return nil
	}
	reporter := common.NewErrorReporter()
	for _, err := range errs {
		reporter.Fail("%v", err)
	}
	return fmt.Errorf("pragmas that fail the range check: %d", len(errs))
}

// readPragmas returns the versions of each pragma solidity in the file.
func readPragmas(filePath string) ([]string, []error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read file: %w", err)}
	}
	var pragmas []string
	for _, match := range pragmaPattern.FindAllStringSubmatch(removeComments(string(content)), -1) {
		pragmas = append(pragmas, strings.TrimSpace(match[1]))
	}
	return pragmas, nil
}

// rangeErrors returns an error for each pragma range, by file, that allows none of the versions
// pinned by an exact pragma in any of the files. It checks each file on its own: a file that a
// pinned file imports compiles with that pin, so this covers the files that only ranges import.
func rangeErrors(pragmas map[string][]string) []error {
	files := slices.Sorted(maps.Keys(pragmas))
	var errs []error
	var pinned []*semver.Version
	for _, filePath := range files {
		for _, pragma := range pragmas[filePath] {
			if !isStrictPragma(pragma) {
				continue
			}
			version, err := semver.NewVersion(pragma)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: cannot parse pragma solidity %s: %w", filePath, pragma, err))
				continue
			}
			pinned = append(pinned, version)
		}
	}
	slices.SortFunc(pinned, func(a, b *semver.Version) int { return a.Compare(b) })
	pinned = slices.CompactFunc(pinned, func(a, b *semver.Version) bool { return a.Equal(b) })
	pinnedNames := make([]string, len(pinned))
	for i, v := range pinned {
		pinnedNames[i] = v.String()
	}

	for _, filePath := range files {
		for _, pragma := range pragmas[filePath] {
			if isStrictPragma(pragma) {
				continue
			}
			constraint, err := semver.NewConstraint(pragma)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: cannot parse pragma solidity %s: %w", filePath, pragma, err))
				continue
			}
			if !slices.ContainsFunc(pinned, constraint.Check) {
				errs = append(errs, fmt.Errorf("%s: pragma solidity %s allows none of the pinned compiler versions (%s). forge builds the file with the newest installed solc, and op-deployer then cannot run the built contracts where that solc is missing. Allow a pinned version", filePath, pragma, strings.Join(pinnedNames, ", ")))
			}
		}
	}
	return errs
}

func processFile(filePath string) (*common.Void, []error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read file: %w", err)}
	}

	contentStr := string(content)

	// Check if file contains a concrete contract (not abstract, not library, not interface)
	if !hasConcreteContract(contentStr) {
		return nil, nil
	}

	// Check if pragma is strict
	pragma := extractPragma(contentStr)
	if pragma == "" {
		return nil, []error{fmt.Errorf("no pragma found")}
	}

	if !isStrictPragma(pragma) {
		return nil, []error{fmt.Errorf("non-strict pragma '%s' - contracts must use exact version (e.g., '0.8.15' not '^0.8.15')", pragma)}
	}

	return nil, nil
}

// hasConcreteContract returns true if the file contains at least one concrete contract
// (not abstract, not library, not interface)
func hasConcreteContract(content string) bool {
	// Remove comments to avoid false positives
	content = removeComments(content)

	// Check for concrete contract definition
	hasContract := contractPattern.MatchString(content)
	if !hasContract {
		return false
	}

	// Make sure it's not just abstract contracts, libraries, or interfaces
	// by checking if we have a "contract X" that isn't preceded by "abstract"
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip if it's an abstract contract, library, or interface
		if abstractPattern.MatchString(trimmed) ||
			libraryPattern.MatchString(trimmed) ||
			interfacePattern.MatchString(trimmed) {
			continue
		}
		// Check for concrete contract
		if contractPattern.MatchString(trimmed) {
			return true
		}
	}

	return false
}

// extractPragma extracts the pragma version string from the content
func extractPragma(content string) string {
	matches := pragmaPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// isStrictPragma returns true if the pragma is a strict version (no ^ or >= or other operators)
func isStrictPragma(pragma string) bool {
	// Strict pragma should be just a version number like "0.8.15"
	// Non-strict examples: "^0.8.0", ">=0.8.0", ">=0.8.0 <0.9.0", "0.8.x"

	// Check for common non-strict indicators
	nonStrictIndicators := []string{"^", ">=", "<=", ">", "<", "~", "x", "X", "*", " "}
	for _, indicator := range nonStrictIndicators {
		if strings.Contains(pragma, indicator) {
			return false
		}
	}

	// Should match a simple version pattern like "0.8.15"
	strictPattern := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	return strictPattern.MatchString(pragma)
}

// removeComments removes single-line and multi-line comments from Solidity code
func removeComments(content string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	inMultiLineComment := false

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multi-line comments
		if inMultiLineComment {
			if idx := strings.Index(line, "*/"); idx != -1 {
				line = line[idx+2:]
				inMultiLineComment = false
			} else {
				continue
			}
		}

		// Remove multi-line comment starts
		for {
			startIdx := strings.Index(line, "/*")
			if startIdx == -1 {
				break
			}
			endIdx := strings.Index(line[startIdx:], "*/")
			if endIdx == -1 {
				line = line[:startIdx]
				inMultiLineComment = true
				break
			}
			line = line[:startIdx] + line[startIdx+endIdx+2:]
		}

		// Remove single-line comments
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}
