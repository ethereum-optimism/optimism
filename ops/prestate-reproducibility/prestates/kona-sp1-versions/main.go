package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/prestate-reproducibility/prestates"
)

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
var hashPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

func main() {
	output := flag.String("output", "", "Write the fetched registry snapshot here")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "must specify --output")
		os.Exit(2)
	}
	data, err := prestates.FetchStandardPrestates()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	versions, err := selectKonaSP1Versions(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, version := range versions {
		fmt.Println(version)
	}
}

func selectKonaSP1Versions(data []byte) ([]string, error) {
	registry, err := prestates.ParseReleases(data)
	if err != nil {
		return nil, err
	}
	var versions []string
	for version, entries := range registry.Prestates {
		seen := false
		for _, entry := range entries {
			if entry.Type != "kona-sp1" {
				continue
			}
			if seen {
				return nil, fmt.Errorf("duplicate kona-sp1 prestate for version %s", version)
			}
			seen = true
			if !versionPattern.MatchString(version) {
				return nil, fmt.Errorf("invalid kona-sp1 version %q", version)
			}
			if !hashPattern.MatchString(entry.Hash) || strings.EqualFold(entry.Hash, "0x"+strings.Repeat("0", 64)) {
				return nil, fmt.Errorf("invalid kona-sp1 hash for version %s: %q", version, entry.Hash)
			}
			versions = append(versions, version)
		}
	}
	sort.Strings(versions)
	return versions, nil
}
