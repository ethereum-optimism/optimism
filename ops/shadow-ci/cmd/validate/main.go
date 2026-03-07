package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/validate"
)

func main() {
	path := flag.String("pipeline", ".circleci/continue/shadow-ci.yml", "Path to CircleCI pipeline YAML")
	strict := flag.Bool("strict", false, "Treat warnings as errors")
	flag.Parse()

	issues, err := validate.ValidatePipeline(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Println("pipeline validation passed — no issues found")
		os.Exit(0)
	}

	hasError := false
	for _, issue := range issues {
		fmt.Println(issue)
		if issue.Severity == "error" || (*strict && issue.Severity == "warning") {
			hasError = true
		}
	}

	if hasError {
		fmt.Fprintf(os.Stderr, "\npipeline validation failed: %d issue(s)\n", len(issues))
		os.Exit(1)
	}
	fmt.Printf("\npipeline validation passed with %d warning(s)\n", len(issues))
}
