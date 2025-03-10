package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
)

// standardPrestatesUrl is the URL to the TOML file in superchain registry that defines the list of standard prestates
// Note that this explicitly points to the main branch and is not pinned to a specific version. The verification check
// intends to
const standardPrestatesUrl = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/refs/heads/main/validation/standard/standard-prestates.toml"

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

	expected, err := loadReleases(expectedFile)
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
	slices.SortFunc(expected, sortFunc)

	differs := false
	report := ""
	for i := 0; i < max(len(actual), len(expected)); i++ {
		get := func(arr []Release, idx int) string {
			if i >= len(arr) {
				return "<missing>"
			} else {
				return formatRelease(arr[i])
			}
		}
		expectedStr := get(expected, i)
		actualStr := get(actual, i)
		releaseDiffers := expectedStr != actualStr
		marker := "✅"
		if releaseDiffers {
			marker = "❌"
		}
		report += fmt.Sprintf("%v %d\tExpected: %v\tActual: %v\n", marker, i, expectedStr, actualStr)
		differs = differs || releaseDiffers
	}
	fmt.Println(report)
	if differs {
		os.Exit(1)
	}
}

func formatRelease(release Release) string {
	return fmt.Sprintf("%-13v %s %-10v", release.Version, release.Hash, release.Type)
}

func loadReleases(overrideFile string) ([]Release, error) {
	var data []byte
	if overrideFile != "" {
		d, err := os.ReadFile(overrideFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read override file (%v): %w", overrideFile, err)
		}
		data = d
	} else {
		resp, err := http.Get(standardPrestatesUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to download standard prestates from %v: %w", standardPrestatesUrl, err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read standard prestates from %v: %w", standardPrestatesUrl, err)
		}
	}
	var standardPrestates Prestates
	err := toml.Unmarshal(data, &standardPrestates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standard prestates from %v: %w", standardPrestatesUrl, err)
	}
	var releases []Release
	for version, prestateList := range standardPrestates.Prestates {
		for _, prestate := range prestateList {
			releases = append(releases, Release{
				Version: version,
				Hash:    prestate.Hash.Hex(),
				Type:    prestate.Type,
			})
		}
	}
	return releases, nil
}

type Prestates struct {
	Prestates map[string][]Prestate `toml:"prestates"`
}

type Prestate struct {
	Type string      `toml:"type"`
	Hash common.Hash `toml:"hash"`
}

type Release struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
	Type    string `json:"type"`
}
