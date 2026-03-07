package validate

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Issue represents a validation finding.
type Issue struct {
	Severity string // "error" or "warning"
	Location string // e.g. "workflows.shadow-ci.jobs[3]"
	Message  string
}

func (i Issue) String() string {
	return fmt.Sprintf("[%s] %s: %s", i.Severity, i.Location, i.Message)
}

// ValidatePipeline loads a CircleCI YAML file and checks for common mistakes.
func ValidatePipeline(path string) ([]Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	var issues []Issue
	issues = append(issues, checkWorkspaceRoots(doc)...)
	issues = append(issues, checkWorkflowRequires(doc)...)
	issues = append(issues, checkMatrixCommas(doc)...)
	issues = append(issues, checkPythonImports(data)...)
	return issues, nil
}

// checkWorkspaceRoots detects jobs that attach_workspace or persist_to_workspace
// at multiple different roots, which causes silent data loss.
func checkWorkspaceRoots(doc map[string]any) []Issue {
	jobs, _ := doc["jobs"].(map[string]any)
	if jobs == nil {
		return nil
	}

	var issues []Issue
	for name, jobRaw := range jobs {
		job, _ := jobRaw.(map[string]any)
		if job == nil {
			continue
		}
		steps, _ := job["steps"].([]any)

		for _, check := range []string{"persist_to_workspace", "attach_workspace"} {
			roots := collectWorkspaceRoots(steps, check)
			if len(roots) > 1 {
				issues = append(issues, Issue{
					Severity: "error",
					Location: fmt.Sprintf("jobs.%s", name),
					Message:  fmt.Sprintf("multiple %s roots: %v — only one root is supported per job", check, roots),
				})
			}
		}
	}
	return issues
}

func collectWorkspaceRoots(steps []any, key string) []string {
	var roots []string
	seen := map[string]bool{}
	for _, step := range steps {
		stepMap, _ := step.(map[string]any)
		if stepMap == nil {
			continue
		}
		ws, ok := stepMap[key].(map[string]any)
		if !ok {
			continue
		}
		root := fmt.Sprint(ws["root"])
		if key == "attach_workspace" {
			root = fmt.Sprint(ws["at"])
		}
		if root != "" && !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	return roots
}

// checkWorkflowRequires checks that every `requires` entry in a workflow
// references a job defined in the same workflow.
func checkWorkflowRequires(doc map[string]any) []Issue {
	workflows, _ := doc["workflows"].(map[string]any)
	if workflows == nil {
		return nil
	}

	jobs, _ := doc["jobs"].(map[string]any)

	var issues []Issue
	for wfName, wfRaw := range workflows {
		wf, _ := wfRaw.(map[string]any)
		if wf == nil {
			continue
		}
		wfJobs, _ := wf["jobs"].([]any)
		if wfJobs == nil {
			continue
		}

		// Collect all job names defined in this workflow (including matrix expansions).
		definedNames := collectDefinedJobNames(wfJobs)

		// Check each job's requires.
		for i, jobEntry := range wfJobs {
			jobMap, _ := jobEntry.(map[string]any)
			if jobMap == nil {
				continue
			}
			for jobType, configRaw := range jobMap {
				config, _ := configRaw.(map[string]any)
				if config == nil {
					continue
				}
				requires, _ := config["requires"].([]any)
				for _, req := range requires {
					reqName := fmt.Sprint(req)
					if !definedNames[reqName] {
						// Check if it's a job from another workflow (defined in jobs: but not in this workflow).
						_, definedGlobally := jobs[reqName]
						hint := ""
						if definedGlobally {
							hint = " (defined as a job but not in this workflow — requires only works within the same workflow)"
						}
						issues = append(issues, Issue{
							Severity: "error",
							Location: fmt.Sprintf("workflows.%s.jobs[%d].%s.requires", wfName, i, jobType),
							Message:  fmt.Sprintf("requires %q but no job with that name exists in this workflow%s", reqName, hint),
						})
					}
				}
			}
		}
	}
	return issues
}

func collectDefinedJobNames(wfJobs []any) map[string]bool {
	names := map[string]bool{}
	for _, jobEntry := range wfJobs {
		switch v := jobEntry.(type) {
		case string:
			names[v] = true
		case map[string]any:
			for jobType, configRaw := range v {
				config, _ := configRaw.(map[string]any)
				if config == nil {
					names[jobType] = true
					continue
				}
				name, _ := config["name"].(string)
				if name == "" {
					name = jobType
				}

				// Handle matrix expansion.
				matrix, _ := config["matrix"].(map[string]any)
				if matrix != nil && strings.Contains(name, "<<matrix.") {
					params, _ := matrix["parameters"].(map[string]any)
					expandMatrixNames(name, params, names)
				} else {
					names[name] = true
				}
			}
		}
	}
	return names
}

func expandMatrixNames(nameTemplate string, params map[string]any, out map[string]bool) {
	if len(params) == 0 {
		out[nameTemplate] = true
		return
	}
	// Single-parameter expansion (covers the common case).
	for paramKey, paramValues := range params {
		values, _ := paramValues.([]any)
		placeholder := fmt.Sprintf("<<matrix.%s>>", paramKey)
		for _, val := range values {
			expanded := strings.ReplaceAll(nameTemplate, placeholder, fmt.Sprint(val))
			out[expanded] = true
		}
		return // only first param dimension
	}
}

// checkMatrixCommas warns about commas in matrix parameter values,
// which can break CircleCI's parameter parsing.
func checkMatrixCommas(doc map[string]any) []Issue {
	workflows, _ := doc["workflows"].(map[string]any)
	if workflows == nil {
		return nil
	}

	var issues []Issue
	for wfName, wfRaw := range workflows {
		wf, _ := wfRaw.(map[string]any)
		if wf == nil {
			continue
		}
		wfJobs, _ := wf["jobs"].([]any)
		for i, jobEntry := range wfJobs {
			jobMap, _ := jobEntry.(map[string]any)
			if jobMap == nil {
				continue
			}
			for jobType, configRaw := range jobMap {
				config, _ := configRaw.(map[string]any)
				if config == nil {
					continue
				}
				matrix, _ := config["matrix"].(map[string]any)
				if matrix == nil {
					continue
				}
				params, _ := matrix["parameters"].(map[string]any)
				for pk, pvRaw := range params {
					values, _ := pvRaw.([]any)
					for _, val := range values {
						s := fmt.Sprint(val)
						if strings.Contains(s, ",") {
							issues = append(issues, Issue{
								Severity: "warning",
								Location: fmt.Sprintf("workflows.%s.jobs[%d].%s.matrix.parameters.%s", wfName, i, jobType, pk),
								Message:  fmt.Sprintf("matrix value %q contains a comma — this may break CircleCI parameter parsing; use explicit job entries instead", s),
							})
						}
					}
				}
			}
		}
	}
	return issues
}

// checkPythonImports scans for Python imports that aren't available on cimg/base.
var unsafePythonImports = []string{"import yaml", "import requests", "import numpy"}

func checkPythonImports(data []byte) []Issue {
	content := string(data)
	var issues []Issue
	for _, imp := range unsafePythonImports {
		if strings.Contains(content, imp) {
			issues = append(issues, Issue{
				Severity: "error",
				Location: "inline python",
				Message:  fmt.Sprintf("%q is not available on cimg/base — use stdlib only (json, sys, os)", imp),
			})
		}
	}
	return issues
}
