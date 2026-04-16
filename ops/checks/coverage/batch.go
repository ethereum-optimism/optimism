package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// BatchResult summarizes a batch run.
type BatchResult struct {
	Total     int
	Completed int
	Skipped   int
	Failed    int
	Elapsed   time.Duration
}

// RunBatch executes a list of coverage collection jobs sequentially.
// Existing reports (by output filename) are skipped. Individual job failures
// are tolerated — the batch continues and counts failures in the result.
//
// Collectors are instantiated once per language and reused across jobs
// so stateful caches (e.g. RustCollector's rustmeta.Loader) don't get
// thrown away between invocations.
func RunBatch(rootDir string, jobs []Job, outputDir string, cat *catalog.Catalog) (*BatchResult, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	collectors := map[string]Collector{
		"solidity": NewSolidityCollector(),
		"go":       NewGoCollector(),
		"rust":     NewRustCollector(),
	}

	res := &BatchResult{Total: len(jobs)}
	start := time.Now()
	var ranStart time.Time
	var ran int

	for i, job := range jobs {
		outPath := filepath.Join(outputDir, job.OutputName())
		if _, err := os.Stat(outPath); err == nil {
			res.Skipped++
			fmt.Printf("[%d/%d] SKIP %s %s\n", i+1, res.Total, job.Profile, job.Test)
			continue
		}

		// ETA estimate based only on jobs actually run this invocation
		label := ""
		if ran > 0 {
			avg := time.Since(ranStart) / time.Duration(ran)
			remaining := time.Duration(res.Total-i) * avg
			label = fmt.Sprintf("(~%dm remaining) ", int(remaining.Minutes()))
		}
		fmt.Printf("[%d/%d] %s%s %s\n", i+1, res.Total, label, job.Profile, job.Test)

		if ran == 0 {
			ranStart = time.Now()
		}
		ran++

		collector, ok := collectors[job.Language]
		if !ok {
			res.Failed++
			fmt.Printf("  FAIL: unknown language: %s\n", job.Language)
			continue
		}

		profile := Profile{}
		if p := cat.ProfileByName(job.Profile); p != nil {
			profile = Profile{Name: p.Name, Env: p.Env}
		}

		report, err := collector.Collect(rootDir, job.Test, profile)
		if err != nil {
			res.Failed++
			fmt.Printf("  FAIL: %v\n", err)
			continue
		}

		if err := SaveReport(report, outPath); err != nil {
			res.Failed++
			fmt.Printf("  FAIL: saving report: %v\n", err)
			continue
		}

		res.Completed++
	}

	res.Elapsed = time.Since(start)
	return res, nil
}
