package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// mainlinePRJobs lists the jobs that mainline CI runs on PR branches.
// This is the ground truth — extracted from .circleci/continue/main.yml.
var mainlinePRJobs = map[string]string{
	"contracts_build":     "contracts-bedrock-build",
	"sol_tests":           "contracts-bedrock-tests",
	"sol_upgrade":         "contracts-bedrock-tests-upgrade",
	"sol_checks":          "contracts-bedrock-checks, contracts-bedrock-checks-fast",
	"sol_coverage":        "contracts-bedrock-coverage",
	"semgrep":             "semgrep-scan-local, semgrep-test",
	"go_lint":             "go-lint",
	"go_tests":            "go-tests-short",
	"go_fuzz":             "fuzz-golang-*",
	"cannon_prestate":     "cannon-prestate",
	"cannon_tests":        "cannon-go-lint-and-test",
	"go_binaries_for_sysgo": "go-binaries-for-sysgo",
	"acceptance_tests":    "memory-all (op-acceptance-tests)",
	"generated_mocks":     "check-generated-mocks-*",
	"shell_check":         "shell-check",
	"todo_issues":         "todo-issues-check",
	"docker_build":        "op-deployer-docker-build",
	"rust_build":          "kona-build-release",
	"rust_submodule_build": "rust-build-op-rbuilder, rust-build-rollup-boost",
	"rust_ci":             "rust-ci workflow (fmt, clippy, tests, etc.)",
	"rust_e2e":            "rust-e2e-ci workflow",
}

// mainlineDevelopOnlyJobs are jobs that only run on develop.
var mainlineDevelopOnlyJobs = map[string]string{
	"go_fault_proofs":         "fault-proofs e2e (develop only)",
	"kontrol":                 "kontrol formal verification (develop only)",
	"publish_cannon_prestates": "publish cannon prestates (develop only)",
}

// mainlineScheduleOnlyJobs only run on scheduled pipelines.
var mainlineScheduleOnlyJobs = map[string]string{
	"sol_heavy_fuzz": "heavy fuzz nightly",
	"sync_tests":     "sync tests (daily)",
	"flake_shake":    "flake shake (daily)",
	"docker_publish": "docker publish (scheduled)",
}

func main() {
	decisionPath := flag.String("decision", "/tmp/shadow-ci/decision.json", "Path to decision JSON")
	flag.Parse()

	data, err := os.ReadFile(*decisionPath)
	if err != nil {
		fatal("reading decision: %v", err)
	}

	var decision model.PipelineDecision
	if err := json.Unmarshal(data, &decision); err != nil {
		fatal("parsing decision: %v", err)
	}

	fmt.Printf("=== Shadow CI Coherence Report ===\n")
	fmt.Printf("Stage: %s | Branch: %s | ForceAll: %v\n\n", decision.Stage, decision.Branch, decision.ForceAll)

	errors := 0
	warnings := 0

	// Check: every mainline PR job should be represented.
	fmt.Println("--- PR Stage Coherence ---")
	if decision.Stage == model.StagePR || decision.Stage == model.StageMergeQueue {
		for cat, mainJob := range mainlinePRJobs {
			cd, ok := decision.Categories[cat]
			if !ok {
				fmt.Printf("  ERROR  %-30s missing from decision (mainline: %s)\n", cat, mainJob)
				errors++
				continue
			}
			if cd.Needed && !cd.Skipped {
				fmt.Printf("  OK     %-30s → RUN  (mainline: %s)\n", cat, mainJob)
			} else if cd.StageSkipped {
				fmt.Printf("  DEFER  %-30s → deferred to %s (mainline: %s)\n", cat, cd.PlacedAt, mainJob)
			} else {
				fmt.Printf("  WARN   %-30s → SKIP (%s) but mainline runs %s\n", cat, cd.SkipWhy, mainJob)
				warnings++
			}
		}
	}

	// Check: develop-only jobs should be skipped on PR.
	fmt.Println("\n--- Develop-Only Jobs ---")
	for cat, desc := range mainlineDevelopOnlyJobs {
		cd, ok := decision.Categories[cat]
		if !ok {
			fmt.Printf("  ERROR  %-30s missing from decision (%s)\n", cat, desc)
			errors++
			continue
		}
		if decision.Stage == model.StagePR {
			if cd.Skipped {
				fmt.Printf("  OK     %-30s correctly skipped on PR (%s)\n", cat, desc)
			} else {
				fmt.Printf("  ERROR  %-30s running on PR but should be develop-only (%s)\n", cat, desc)
				errors++
			}
		} else if decision.IsDevelop {
			if cd.Needed && !cd.Skipped {
				fmt.Printf("  OK     %-30s running on develop (%s)\n", cat, desc)
			} else {
				fmt.Printf("  WARN   %-30s skipped on develop but should run (%s)\n", cat, desc)
				warnings++
			}
		}
	}

	// Check: schedule-only jobs should be skipped on non-schedule.
	fmt.Println("\n--- Schedule-Only Jobs ---")
	for cat, desc := range mainlineScheduleOnlyJobs {
		cd, ok := decision.Categories[cat]
		if !ok {
			fmt.Printf("  ERROR  %-30s missing from decision (%s)\n", cat, desc)
			errors++
			continue
		}
		if !decision.IsSchedule {
			if cd.Skipped {
				fmt.Printf("  OK     %-30s correctly skipped (not scheduled) (%s)\n", cat, desc)
			} else {
				fmt.Printf("  ERROR  %-30s running but should be schedule-only (%s)\n", cat, desc)
				errors++
			}
		}
	}

	// Check: tag-only jobs should be skipped.
	fmt.Println("\n--- Tag-Only Jobs ---")
	if cd, ok := decision.Categories["publish_contract_artifacts"]; ok {
		if cd.Skipped {
			fmt.Printf("  OK     %-30s correctly skipped (not a tag build)\n", "publish_contract_artifacts")
		} else {
			fmt.Printf("  ERROR  %-30s running but should be tag-only\n", "publish_contract_artifacts")
			errors++
		}
	}

	// Summary of all categories.
	fmt.Println("\n--- Full Category Summary ---")
	var cats []string
	for name := range decision.Categories {
		cats = append(cats, name)
	}
	sort.Strings(cats)
	runCount, skipCount := 0, 0
	for _, name := range cats {
		cd := decision.Categories[name]
		status := "SKIP"
		detail := cd.SkipWhy
		if cd.Needed && !cd.Skipped {
			status = "RUN"
			detail = cd.Reason
			runCount++
		} else {
			skipCount++
		}
		if cd.StageSkipped {
			status = "DEFR"
		}
		fmt.Printf("  %-5s  %-30s %s\n", status, name, detail)
	}

	fmt.Printf("\n--- Result ---\n")
	fmt.Printf("Categories: %d run, %d skip, %d total\n", runCount, skipCount, runCount+skipCount)
	fmt.Printf("Errors:     %d\n", errors)
	fmt.Printf("Warnings:   %d\n", warnings)

	if errors > 0 {
		fmt.Printf("\nFAIL: %d coherence errors found\n", errors)
		os.Exit(1)
	}

	// Check category coverage — every scoping.yaml category should be in the decision.
	uncovered := findUncoveredCategories(decision.Categories)
	if len(uncovered) > 0 {
		fmt.Printf("\nWARN: %d categories in mainline not tracked by shadow CI: %s\n",
			len(uncovered), strings.Join(uncovered, ", "))
	}

	fmt.Println("\nPASS: Shadow CI decision is coherent with mainline CI")
}

func findUncoveredCategories(categories map[string]*model.CategoryDecision) []string {
	// All known mainline categories.
	known := make(map[string]bool)
	for k := range mainlinePRJobs {
		known[k] = true
	}
	for k := range mainlineDevelopOnlyJobs {
		known[k] = true
	}
	for k := range mainlineScheduleOnlyJobs {
		known[k] = true
	}
	known["publish_contract_artifacts"] = true

	var uncovered []string
	for k := range known {
		if _, ok := categories[k]; !ok {
			uncovered = append(uncovered, k)
		}
	}
	sort.Strings(uncovered)
	return uncovered
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
