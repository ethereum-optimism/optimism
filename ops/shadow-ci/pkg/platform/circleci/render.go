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
func (r *Renderer) RenderFromDecision(decision *model.PipelineDecision, plan *model.TestPlan) ([]byte, error) {
	data := decisionTemplateData{
		Stage: string(decision.Stage),
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
			data.Jobs = append(data.Jobs, tj)
		}
	}

	// Add non-graph categories as simple runner jobs.
	for name, cd := range decision.Categories {
		if !cd.Needed || cd.Skipped {
			continue
		}
		// Skip graph-based categories — they're handled via plan jobs above.
		if len(cd.Targets) > 0 {
			continue
		}
		tj := templateJob{
			Name:        fmt.Sprintf("shadow-%s", name),
			Runner:      r.mapRunner("large"),
			Parallelism: 1,
			Timeout:     "20m",
			Language:    name,
			Targets:     strings.Join(cd.Packages, ","),
			Config:      "default",
			Env:         make(map[string]string),
			Reason:      cd.Reason,
		}
		if len(cd.Features) > 0 {
			tj.Targets = strings.Join(cd.Features, ",")
		}
		data.Jobs = append(data.Jobs, tj)
	}

	// Comparison job depends on all test jobs.
	data.ComparisonDeps = make([]string, len(data.Jobs))
	for i, j := range data.Jobs {
		data.ComparisonDeps[i] = j.Name
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

type decisionTemplateData struct {
	Stage          string
	Jobs           []templateJob
	ComparisonDeps []string
}

var decisionWorkflowTemplate = `# Auto-generated by shadow-ci decision engine. Do not edit.
# Stage: {{ .Stage }}

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
            ops/shadow-ci/bin/runner \
              --language={{ .Language | yamlSafe }} \
              --targets={{ .Targets | yamlSafe }} \
              --config={{ .Config | yamlSafe }} \
              --reason={{ .Reason | yamlSafe }}
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
{{- range .Jobs }}
      - {{ .Name }}
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
            ops/shadow-ci/bin/runner \
              --language={{ .Language | yamlSafe }} \
              --targets={{ .Targets | yamlSafe }} \
              --config={{ .Config | yamlSafe }} \
              --reason={{ .Reason | yamlSafe }}
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
