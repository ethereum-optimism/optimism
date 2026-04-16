package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// Job is a single coverage collection unit: run this test under this profile.
type Job struct {
	Language string // "solidity"
	Profile  string // catalog profile name
	Test     string // test identifier relative to the language's root (e.g. "test/L2/L1Block.t.sol")
}

// OutputName returns the canonical report filename for a job.
// Format: test_<underscored-test-path>__<profile>.json
func (j Job) OutputName() string {
	name := j.Test
	name = strings.TrimPrefix(name, "test/")
	name = strings.TrimSuffix(name, ".t.sol")
	name = strings.ReplaceAll(name, "/", "_")
	return "test_" + name + "__" + j.Profile + ".json"
}

// ComputeSolidityJobs returns every (.t.sol file × profile) combination.
// No filtering: the coverage data itself is the ground truth about which
// tests behave differently under which profiles.
func ComputeSolidityJobs(cat *catalog.Catalog, contractsDir string) ([]Job, error) {
	testDir := filepath.Join(contractsDir, "test")

	var tests []string
	err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".t.sol") {
			return nil
		}
		rel, err := filepath.Rel(contractsDir, path)
		if err != nil {
			return err
		}
		tests = append(tests, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking test dir %q: %w", testDir, err)
	}

	sort.Strings(tests)

	var jobs []Job
	for _, p := range cat.Profiles {
		for _, t := range tests {
			jobs = append(jobs, Job{Language: "solidity", Profile: p.Name, Test: t})
		}
	}
	return jobs, nil
}
