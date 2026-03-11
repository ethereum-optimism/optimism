// Command import-history pulls historical test results from CircleCI
// and writes them as events to the local event store. This bootstraps
// the correlation matrix and stats aggregation.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform/circleci"
)

func main() {
	project := flag.String("project", "gh/ethereum-optimism/optimism", "CircleCI project slug")
	branch := flag.String("branch", "develop", "Branch to import from")
	limit := flag.Int("limit", 20, "Maximum number of pipelines to import")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci/events", "Events directory")
	flag.Parse()

	token := os.Getenv("CIRCLE_TOKEN")
	client := circleci.NewClient(token, *project)

	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, "import-history", 0, "", *branch)

	log.Printf("Importing history from %s (branch: %s, limit: %d)", *project, *branch, *limit)

	pipelines, err := client.ListPipelines(*branch, *limit)
	if err != nil {
		fatal("listing pipelines: %v", err)
	}

	log.Printf("Found %d pipelines", len(pipelines))

	totalResults := 0
	for _, pipeline := range pipelines {
		workflows, err := client.GetWorkflows(pipeline.ID)
		if err != nil {
			log.Printf("  WARN: failed to get workflows for pipeline %s: %v", pipeline.ID, err)
			continue
		}

		for _, wf := range workflows {
			jobs, err := client.GetWorkflowJobs(wf.ID)
			if err != nil {
				log.Printf("  WARN: failed to get jobs for workflow %s: %v", wf.ID, err)
				continue
			}

			for _, job := range jobs {
				artifacts, err := client.GetJobArtifacts(job.ProjectSlug, job.JobNumber)
				if err != nil {
					continue
				}

				for _, artifact := range artifacts {
					results := importArtifact(client, artifact, job.Name)
					for _, r := range results {
						eventType := model.EventTestPassed
						if r.Status == model.StatusFail {
							eventType = model.EventTestFailed
						}
						emitter.Emit(eventType, map[string]any{
							"test":     r.Test,
							"duration": r.Duration.Seconds(),
							"config":   r.Config,
							"job":      job.Name,
						})
						totalResults++
					}
				}
			}
		}
	}

	log.Printf("Imported %d test results", totalResults)
}

func importArtifact(client *circleci.Client, artifact circleci.ArtifactItem, jobName string) []model.TestResult {
	data, err := client.DownloadArtifact(artifact.URL)
	if err != nil {
		return nil
	}

	path := strings.ToLower(artifact.Path)

	switch {
	case strings.HasSuffix(path, ".json") && strings.Contains(path, "gotestsum"):
		results, err := circleci.ParseGotestsumJSON(data)
		if err != nil {
			log.Printf("  WARN: failed to parse gotestsum JSON from %s: %v", artifact.Path, err)
			return nil
		}
		return results

	case strings.HasSuffix(path, ".xml"):
		results, err := circleci.ParseJUnitXML(data)
		if err != nil {
			log.Printf("  WARN: failed to parse JUnit XML from %s: %v", artifact.Path, err)
			return nil
		}
		return results

	case strings.HasSuffix(path, ".json") && strings.Contains(path, "forge"):
		results, err := circleci.ParseForgeJSON(data)
		if err != nil {
			log.Printf("  WARN: failed to parse forge JSON from %s: %v", artifact.Path, err)
			return nil
		}
		return results
	}

	return nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
