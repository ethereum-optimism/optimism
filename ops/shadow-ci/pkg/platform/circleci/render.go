package circleci

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Renderer converts TestPlans into CircleCI continuation YAML.
type Renderer struct {
	runners map[string]string // abstract → CircleCI resource class
}

// NewRenderer creates a new CircleCI YAML renderer.
func NewRenderer(runners map[string]string) *Renderer {
	return &Renderer{runners: runners}
}

// Render produces CircleCI YAML from a TestPlan.
func (r *Renderer) Render(plan model.TestPlan) ([]byte, error) {
	data := templateData{
		Plan: plan,
		Jobs: make([]templateJob, 0, len(plan.Jobs)),
	}

	for _, job := range plan.Jobs {
		tj := templateJob{
			Name:        fmt.Sprintf("shadow-%s", job.Name),
			Runner:      r.mapRunner(job.Resources.Runner),
			Parallelism: job.Resources.Parallelism,
			Timeout:     formatTimeout(job.Resources.Timeout),
			Language:    job.Language,
			Targets:     targetIDs(job.Targets),
			Config:      job.Configurations[0].Name,
			Env:         mergeEnv(job.Configurations),
			Reason:      job.SelectionReason,
		}
		data.Jobs = append(data.Jobs, tj)
	}

	// Comparison job depends on all test jobs.
	data.ComparisonDeps = make([]string, len(data.Jobs))
	for i, j := range data.Jobs {
		data.ComparisonDeps[i] = j.Name
	}

	tmpl, err := template.New("workflow").Funcs(template.FuncMap{
		"yamlSafe": yamlSafe,
	}).Parse(workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *Renderer) mapRunner(abstract string) string {
	if mapped, ok := r.runners[abstract]; ok {
		return mapped
	}
	return "docker+large" // default fallback
}

type templateData struct {
	Plan           model.TestPlan
	Jobs           []templateJob
	ComparisonDeps []string
}

type templateJob struct {
	Name        string
	Runner      string
	Parallelism int
	Timeout     string
	Language    string
	Targets     string
	Config      string
	Env         map[string]string
	Reason      string
	Requires    []string
}

func targetIDs(targets []model.Target) string {
	ids := make([]string, len(targets))
	for i, t := range targets {
		ids[i] = t.ID
	}
	return strings.Join(ids, ",")
}

func mergeEnv(configs []model.Configuration) map[string]string {
	env := make(map[string]string)
	for _, c := range configs {
		for k, v := range c.Env {
			env[k] = v
		}
	}
	return env
}

func formatTimeout(d time.Duration) string {
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// yamlSafe quotes a string for safe inclusion in YAML. If the string contains
// characters that could be interpreted as YAML structure (colons, newlines,
// quotes, braces, etc.) it wraps in double quotes with internal escaping.
func yamlSafe(s string) string {
	if s == "" {
		return `""`
	}
	const dangerous = `:{}[]|>*&!%#@,"'\` + "\n\r\t\\"
	needsQuote := false
	for _, c := range s {
		if strings.ContainsRune(dangerous, c) {
			needsQuote = true
			break
		}
	}
	// Strings starting with quotes, dashes, or special YAML values also need quoting.
	if !needsQuote {
		first := s[0]
		if first == '"' || first == '\'' || first == '-' || first == ' ' {
			needsQuote = true
		}
	}
	if !needsQuote {
		lower := strings.ToLower(s)
		if lower == "true" || lower == "false" || lower == "null" || lower == "yes" || lower == "no" {
			needsQuote = true
		}
	}
	if !needsQuote {
		return s
	}
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\r", `\r`)
	escaped = strings.ReplaceAll(escaped, "\t", `\t`)
	return `"` + escaped + `"`
}

// RenderFromDecision produces CircleCI continuation YAML from a PipelineDecision.
// This is the dynamic pipeline generation path — the decision engine decides
// which categories run, and this method generates the minimal YAML for them.
// It resolves the dependency graph: when a test category depends on a build
// category (via depends_on in scoping config), the build job is automatically
// included and the test job gets a `requires` clause.
func (r *Renderer) RenderFromDecision(decision *model.PipelineDecision, plan *model.TestPlan, scoping model.ScopingConfig) ([]byte, error) {
	data := decisionTemplateData{
		Stage: string(decision.Stage),
	}

	// Collect all needed test/non-build category names.
	neededCategories := make(map[string]bool)
	for name, cd := range decision.Categories {
		if cd.Needed && !cd.Skipped {
			neededCategories[name] = true
		}
	}

	// Resolve build dependencies: for each needed category, include its
	// depends_on categories as build jobs.
	buildJobsNeeded := make(map[string]bool)
	categoryDeps := make(map[string][]string) // category → build job names it requires
	for name := range neededCategories {
		cat, ok := scoping.JobCategories[name]
		if !ok {
			continue
		}
		for _, dep := range cat.DependsOn {
			depCat, depOk := scoping.JobCategories[dep]
			if !depOk {
				continue
			}
			buildName := fmt.Sprintf("shadow-%s", dep)
			buildJobsNeeded[dep] = true

			// Also resolve transitive deps of build categories.
			for _, transDep := range depCat.DependsOn {
				buildJobsNeeded[transDep] = true
				// The build dep itself requires its own transitive dep.
				transBuildName := fmt.Sprintf("shadow-%s", transDep)
				categoryDeps[dep] = appendUnique(categoryDeps[dep], transBuildName)
			}

			categoryDeps[name] = appendUnique(categoryDeps[name], buildName)
		}
	}

	// Generate build jobs.
	for depName := range buildJobsNeeded {
		cat, ok := scoping.JobCategories[depName]
		if !ok {
			continue
		}
		runner := "large"
		if cat.RunnerClass != "" {
			runner = cat.RunnerClass
		}
		bj := buildTemplateJob{
			Name:           fmt.Sprintf("shadow-%s", depName),
			Runner:         r.mapRunner(runner),
			Timeout:        "30m",
			Command:        strings.TrimSpace(cat.Command),
			WorkspacePaths: cat.WorkspacePaths,
			Requires:       categoryDeps[depName],
		}
		if bj.Command == "" {
			bj.Command = fmt.Sprintf("echo 'Build step: %s'", depName)
		}
		data.BuildJobs = append(data.BuildJobs, bj)
	}

	// Add test jobs from the plan (graph-based categories).
	if plan != nil {
		for _, job := range plan.Jobs {
			tj := templateJob{
				Name:        fmt.Sprintf("shadow-%s", job.Name),
				Runner:      r.mapRunner(job.Resources.Runner),
				Parallelism: job.Resources.Parallelism,
				Timeout:     formatTimeout(job.Resources.Timeout),
				Language:    job.Language,
				Targets:     targetIDs(job.Targets),
				Config:      job.Configurations[0].Name,
				Env:         mergeEnv(job.Configurations),
				Reason:      job.SelectionReason,
			}
			// Resolve requires from the job's language to find its category.
			for catName, cat := range scoping.JobCategories {
				if cat.UseGraph && cat.Language == job.Language {
					tj.Requires = categoryDeps[catName]
					break
				}
			}
			data.TestJobs = append(data.TestJobs, tj)
		}
	}

	// Add non-graph categories as simple runner jobs.
	for name, cd := range decision.Categories {
		if !cd.Needed || cd.Skipped {
			continue
		}
		// Skip graph-based categories (handled via plan jobs) and build categories.
		if len(cd.Targets) > 0 || buildJobsNeeded[name] {
			continue
		}
		cat := scoping.JobCategories[name]
		runner := "large"
		if cat.RunnerClass != "" {
			runner = cat.RunnerClass
		}
		tj := templateJob{
			Name:        fmt.Sprintf("shadow-%s", name),
			Runner:      r.mapRunner(runner),
			Parallelism: 1,
			Timeout:     "20m",
			Language:    name,
			Targets:     strings.Join(cd.Packages, ","),
			Config:      "default",
			Env:         make(map[string]string),
			Reason:      cd.Reason,
			Requires:    categoryDeps[name],
		}
		if len(cd.Features) > 0 {
			tj.Targets = strings.Join(cd.Features, ",")
		}
		data.TestJobs = append(data.TestJobs, tj)
	}

	// Comparison job depends on all build + test jobs.
	for _, bj := range data.BuildJobs {
		data.ComparisonDeps = append(data.ComparisonDeps, bj.Name)
	}
	for _, tj := range data.TestJobs {
		data.ComparisonDeps = append(data.ComparisonDeps, tj.Name)
	}

	tmpl, err := template.New("decision-workflow").Funcs(template.FuncMap{
		"yamlSafe": yamlSafe,
	}).Parse(decisionWorkflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

type decisionTemplateData struct {
	Stage          string
	BuildJobs      []buildTemplateJob
	TestJobs       []templateJob
	ComparisonDeps []string
}

type buildTemplateJob struct {
	Name           string
	Runner         string
	Timeout        string
	Command        string
	WorkspacePaths []string
	Requires       []string
}

var decisionWorkflowTemplate = `# Auto-generated by shadow-ci decision engine. Do not edit.
# Stage: {{ .Stage }}

version: 2.1

jobs:
{{- range .BuildJobs }}
  {{ .Name }}:
    machine:
      image: default
    resource_class: {{ .Runner }}
    steps:
      - checkout
      - run:
          name: Build ({{ .Name }})
          no_output_timeout: {{ .Timeout }}
          command: |
            {{ .Command }}
{{- if .WorkspacePaths }}
      - persist_to_workspace:
          root: .
          paths:
{{- range .WorkspacePaths }}
            - {{ . }}
{{- end }}
{{- end }}
{{ end }}
{{- range .TestJobs }}
  {{ .Name }}:
    machine:
      image: default
    resource_class: {{ .Runner }}
    parallelism: {{ .Parallelism }}
    steps:
      - checkout
      - attach_workspace:
          at: .
      - run:
          name: Run {{ .Language }} tests ({{ .Config | yamlSafe }})
          no_output_timeout: {{ .Timeout }}
          command: |
            ops/shadow-ci/bin/execute \
              --group={{ .Language | yamlSafe }} \
              --config=ops/shadow-ci/config \
              --events-dir=/tmp/shadow-ci-events
{{- range $k, $v := .Env }}
          environment:
            {{ $k | yamlSafe }}: {{ $v | yamlSafe }}
{{- end }}
      - store_artifacts:
          path: /tmp/shadow-ci-results
      - store_test_results:
          path: /tmp/shadow-ci-results
{{ end }}
  shadow-comparison:
    docker:
      - image: cimg/base:2026.03
    steps:
      - checkout
      - attach_workspace:
          at: .
      - run:
          name: Compare shadow vs main CI
          command: |
            ops/shadow-ci/bin/compare \
              --pipeline-id="${CIRCLE_PIPELINE_ID}"
      - store_artifacts:
          path: /tmp/shadow-ci-comparison

workflows:
  shadow-ci:
    jobs:
{{- range .BuildJobs }}
{{- if .Requires }}
      - {{ .Name }}:
          requires:
{{- range .Requires }}
            - {{ . }}
{{- end }}
{{- else }}
      - {{ .Name }}
{{- end }}
{{- end }}
{{- range .TestJobs }}
{{- if .Requires }}
      - {{ .Name }}:
          requires:
{{- range .Requires }}
            - {{ . }}
{{- end }}
{{- else }}
      - {{ .Name }}
{{- end }}
{{- end }}
      - shadow-comparison:
          requires:
{{- range .ComparisonDeps }}
            - {{ . }}
{{- end }}
`

var workflowTemplate = `# Auto-generated by shadow-ci render. Do not edit.
# Plan: {{ .Plan.ID | yamlSafe }}
# Created: {{ .Plan.CreatedAt.Format "2006-01-02T15:04:05Z" }}
# Trigger: {{ .Plan.Trigger.Type }} branch={{ .Plan.Trigger.Branch | yamlSafe }}

version: 2.1

jobs:
{{- range .Jobs }}
  {{ .Name }}:
    machine:
      image: default
    resource_class: {{ .Runner }}
    parallelism: {{ .Parallelism }}
    steps:
      - checkout
      - attach_workspace:
          at: .
      - run:
          name: Run {{ .Language }} tests ({{ .Config | yamlSafe }})
          no_output_timeout: {{ .Timeout }}
          command: |
            ops/shadow-ci/bin/execute \
              --group={{ .Language | yamlSafe }} \
              --config=ops/shadow-ci/config \
              --events-dir=/tmp/shadow-ci-events
{{- range $k, $v := .Env }}
          environment:
            {{ $k | yamlSafe }}: {{ $v | yamlSafe }}
{{- end }}
      - store_artifacts:
          path: /tmp/shadow-ci-results
      - store_test_results:
          path: /tmp/shadow-ci-results
{{ end }}
  shadow-comparison:
    docker:
      - image: cimg/base:2026.03
    steps:
      - checkout
      - attach_workspace:
          at: .
      - run:
          name: Compare shadow vs main CI
          command: |
            ops/shadow-ci/bin/compare \
              --pipeline-id="${CIRCLE_PIPELINE_ID}" \
              --plan-id={{ .Plan.ID | yamlSafe }}
      - store_artifacts:
          path: /tmp/shadow-ci-comparison

workflows:
  shadow-ci:
    jobs:
{{- range .Jobs }}
      - {{ .Name }}
{{- end }}
      - shadow-comparison:
          requires:
{{- range .ComparisonDeps }}
            - {{ . }}
{{- end }}
`
