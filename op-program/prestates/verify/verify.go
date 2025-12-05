package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/ethereum-optimism/optimism/op-program/prestates"
)

func main() {
	var inputFile string
	flag.StringVar(&inputFile, "input", "", "Releases JSON file to verify")
	var expectedFile string
	flag.StringVar(&expectedFile, "expected", "", "Override the expected TOML file")
	flag.Parse()
	if inputFile == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Must specify --input")
		os.Exit(2)
	}

	in, err := os.OpenFile(inputFile, os.O_RDONLY, 0o644)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open input file: %v\n", err.Error())
		os.Exit(2)
	}
	defer in.Close()

	input, err := os.ReadFile(inputFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to read input file: %v\n", err.Error())
		os.Exit(2)
	}
	var actual []Release
	err = json.Unmarshal(input, &actual)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err.Error())
		os.Exit(2)
	}

	expected, err := prestates.LoadReleases(expectedFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to load expected releases: %v\n", err.Error())
		os.Exit(2)
	}

	stringCompare := func(a, b string) int {
		if a > b {
			return 1
		} else if a == b {
			return 0
		}
		return -1
	}
	sortFunc := func(a, b Release) int {
		if a.Version > b.Version {
			return 1
		} else if a.Version == b.Version {
			return stringCompare(a.Type, b.Type)
		}
		return -1
	}
	slices.SortFunc(actual, sortFunc)

	differs := false
	report := ""
	for _, release := range actual {
		var expectedPrestate prestates.Prestate
		standardVersion := expected.Prestates[release.Version]
		for _, prestate := range standardVersion {
			if prestate.Type == release.Type {
				expectedPrestate = prestate
				break
			}
		}
		var expectedStr string
		if expectedPrestate == (prestates.Prestate{}) {
			expectedStr = "<missing>"
		} else {
			expectedStr = formatRelease(Release{
				Version: release.Version,
				Type:    expectedPrestate.Type,
				Hash:    expectedPrestate.Hash,
			})
		}
		actualStr := formatRelease(release)
		releaseDiffers := expectedStr != actualStr
		marker := "✅"
		if releaseDiffers {
			marker = "❌"
			differs = true
		}
		report += fmt.Sprintf("%v Expected: %v\tActual: %v\n", marker, expectedStr, actualStr)
	}
	// Verify there aren't any additional entries in expected
	totalExpected := 0
	for version, prestates := range expected.Prestates {
		for _, prestate := range prestates {
			totalExpected++
			// Try to find an actual release matching this expected one
			contains := slices.ContainsFunc(actual, func(release Release) bool {
				return release.Version == version && release.Type == prestate.Type
			})
			if contains {
				continue
			}
			expectedStr := formatRelease(Release{
				Version: version,
				Hash:    prestate.Hash,
				Type:    prestate.Type,
			})
			report += fmt.Sprintf("❌ Expected: %v\tActual: <missing>\n", expectedStr)
			differs = true
		}
	}
	// Sanity check entries are not duplicated in the standard prestates
	if totalExpected != len(actual) {
		report += fmt.Sprintf("❌ Found %v expected prestates but %v actual\n", totalExpected, len(actual))
		differs = true
	}
	fmt.Println(report)
	if differs {
		os.Exit(1)
	}
}

func formatRelease(release Release) string {
	return fmt.Sprintf("%-13v %s %-10v", release.Version, release.Hash, release.Type)
}

type Release struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
	Type    string `json:"type"`
}
