package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// generate-ci renders shadow-ci.yml from the scoping config.
// This is the "magic" — users configure shadow CI in its own language
// (scoping.yaml), and this tool generates the CircleCI config.
//
// Usage:
//   go run ./cmd/generate-ci --config ops/shadow-ci/config --output .circleci/continue/shadow-ci.yml
//   go run ./cmd/generate-ci --config ops/shadow-ci/config --check .circleci/continue/shadow-ci.yml
func main() {
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	output := flag.String("output", "", "Write generated YAML to this path")
	check := flag.String("check", "", "Check that file matches generated output (exits non-zero if stale)")
	flag.Parse()

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	yaml, err := renderShadowCIYAML(cfg)
	if err != nil {
		fatal("rendering: %v", err)
	}

	if *check != "" {
		existing, err := os.ReadFile(*check)
		if err != nil {
			fatal("reading %s: %v", *check, err)
		}
		existingHash := sha256.Sum256(existing)
		generatedHash := sha256.Sum256(yaml)
		if existingHash != generatedHash {
			fmt.Fprintf(os.Stderr, "ERROR: %s is stale — regenerate with:\n", *check)
			fmt.Fprintf(os.Stderr, "  go run ./ops/shadow-ci/cmd/generate-ci --config ops/shadow-ci/config --output %s\n", *check)
			os.Exit(1)
		}
		fmt.Printf("OK: %s is up to date\n", *check)
		return
	}

	if *output != "" {
		if err := os.WriteFile(*output, yaml, 0o644); err != nil {
			fatal("writing %s: %v", *output, err)
		}
		fmt.Printf("Wrote %d bytes to %s\n", len(yaml), *output)
		return
	}

	// Default: print to stdout.
	os.Stdout.Write(yaml)
}

// groupInfo describes an execution group for the CI template.
type groupInfo struct {
	Name         string
	DockerImage  string
	ResourceClass string
	SetupSteps   string // shell commands to install toolchain
	Categories   []string
}

func renderShadowCIYAML(cfg *model.Config) ([]byte, error) {
	// Discover which groups have categories.
	groupCats := make(map[string][]string)
	for name, cat := range cfg.Scoping.JobCategories {
		if cat.Group == "" {
			continue
		}
		groupCats[cat.Group] = append(groupCats[cat.Group], name)
	}
	for _, cats := range groupCats {
		sort.Strings(cats)
	}

	// Build group definitions.
	groups := []groupInfo{
		{
			Name:          "build",
			DockerImage:   "<< pipeline.parameters.c-default_docker_image >>",
			ResourceClass: "2xlarge",
			SetupSteps: `curl -sSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
            echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/.foundry/bin' >> $BASH_ENV
            source $BASH_ENV
            curl -L https://foundry.paradigm.xyz | bash
            ~/.foundry/bin/foundryup`,
			Categories: groupCats["build"],
		},
		{
			Name:          "go",
			DockerImage:   "<< pipeline.parameters.c-default_docker_image >>",
			ResourceClass: "2xlarge",
			SetupSteps: `curl -sSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
            echo 'export PATH=$PATH:/usr/local/go/bin:$(go env GOPATH)/bin' >> $BASH_ENV
            source $BASH_ENV
            go install gotest.tools/gotestsum@latest
            curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.61.0`,
			Categories: groupCats["go"],
		},
		{
			Name:          "sol",
			DockerImage:   "<< pipeline.parameters.c-default_docker_image >>",
			ResourceClass: "2xlarge",
			SetupSteps: `curl -L https://foundry.paradigm.xyz | bash
            echo 'export PATH=$PATH:$HOME/.foundry/bin' >> $BASH_ENV
            source $BASH_ENV
            ~/.foundry/bin/foundryup`,
			Categories: groupCats["sol"],
		},
		{
			Name:          "rust",
			DockerImage:   "<< pipeline.parameters.c-default_docker_image >>",
			ResourceClass: "2xlarge",
			SetupSteps: `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
            echo 'export PATH=$PATH:$HOME/.cargo/bin' >> $BASH_ENV`,
			Categories: groupCats["rust"],
		},
		{
			Name:          "misc",
			DockerImage:   "<< pipeline.parameters.c-default_docker_image >>",
			ResourceClass: "medium",
			SetupSteps:    `sudo apt-get update -qq && sudo apt-get install -y -qq shellcheck`,
			Categories:    groupCats["misc"],
		},
	}

	// Filter out groups with no categories.
	var activeGroups []groupInfo
	for _, g := range groups {
		if len(g.Categories) > 0 {
			activeGroups = append(activeGroups, g)
		}
	}

	// Resolve dependencies: which groups depend on which.
	groupDeps := make(map[string]map[string]bool)
	for name, cat := range cfg.Scoping.JobCategories {
		if cat.Group == "" {
			continue
		}
		for _, dep := range cat.DependsOn {
			depCat, ok := cfg.Scoping.JobCategories[dep]
			if !ok {
				continue
			}
			if depCat.Group != "" && depCat.Group != cat.Group {
				if groupDeps[cat.Group] == nil {
					groupDeps[cat.Group] = make(map[string]bool)
				}
				groupDeps[name] = nil // suppress unused
				groupDeps[cat.Group][depCat.Group] = true
			}
		}
	}

	data := templateData{
		Groups:    activeGroups,
		GroupDeps: groupDeps,
	}

	tmpl, err := template.New("shadow-ci").Parse(shadowCITemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return []byte(buf.String()), nil
}

type templateData struct {
	Groups    []groupInfo
	GroupDeps map[string]map[string]bool
}

var shadowCITemplate = `# AUTO-GENERATED by ops/shadow-ci/cmd/generate-ci — DO NOT EDIT.
# Regenerate: go run ./ops/shadow-ci/cmd/generate-ci --config ops/shadow-ci/config --output .circleci/continue/shadow-ci.yml
#
# Shadow CI — adaptive test placement engine.
# Configure in ops/shadow-ci/config/scoping.yaml, then regenerate this file.
# The shadow-ci-tests job verifies this file is not stale.

version: 2.1

parameters:
  c-default_docker_image:
    type: string
    default: cimg/base:2026.03
  c-go-cache-version:
    type: string
    default: "v0.0"

jobs:
  shadow-ci-setup:
    docker:
      - image: << pipeline.parameters.c-default_docker_image >>
    resource_class: large
    steps:
      - checkout
      - run:
          name: Install Go
          command: |
            curl -sSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
            echo 'export PATH=$PATH:/usr/local/go/bin' >> $BASH_ENV
      - run:
          name: Build shadow CI binaries
          command: |
            cd ops/shadow-ci
            mkdir -p bin
            CGO_ENABLED=0 go build -o bin/affected ./cmd/affected
            CGO_ENABLED=0 go build -o bin/flake-reactor ./cmd/flake-reactor
            CGO_ENABLED=0 go build -o bin/runner ./cmd/runner
            CGO_ENABLED=0 go build -o bin/compare ./cmd/compare
            CGO_ENABLED=0 go build -o bin/aggregate ./cmd/aggregate
            CGO_ENABLED=0 go build -o bin/validate ./cmd/validate
            CGO_ENABLED=0 go build -o bin/optimize ./cmd/optimize
            CGO_ENABLED=0 go build -o bin/auto-revert ./cmd/auto-revert
            CGO_ENABLED=0 go build -o bin/coherence ./cmd/coherence
            CGO_ENABLED=0 go build -o bin/execute ./cmd/execute
            CGO_ENABLED=0 go build -o bin/generate-ci ./cmd/generate-ci
      - run:
          name: Update flake state
          command: |
            ops/shadow-ci/bin/flake-reactor \
              --config ops/shadow-ci/config \
              --events-dir /tmp/shadow-ci/events \
              --branch "$CIRCLE_BRANCH"
      - run:
          name: Compute pipeline decision
          command: |
            ops/shadow-ci/bin/affected \
              --config ops/shadow-ci/config \
              --repo . \
              --branch "$CIRCLE_BRANCH" \
              --events-dir /tmp/shadow-ci/events \
              --pipeline-id "$CIRCLE_PIPELINE_ID"
      - run:
          name: Stage artifacts for workspace
          command: |
            mkdir -p /tmp/shadow-ci-workspace
            cp -r ops/shadow-ci/bin /tmp/shadow-ci-workspace/bin
            cp /tmp/shadow-ci/decision.json /tmp/shadow-ci-workspace/decision.json
            cp /tmp/shadow-ci/affected.json /tmp/shadow-ci-workspace/affected.json
      - persist_to_workspace:
          root: /tmp/shadow-ci-workspace
          paths:
            - bin
            - decision.json
            - affected.json
      - store_artifacts:
          path: /tmp/shadow-ci/decision.json
          destination: shadow-ci/decision.json
      - store_artifacts:
          path: /tmp/shadow-ci/affected.json
          destination: shadow-ci/affected.json

  shadow-ci-verify:
    docker:
      - image: << pipeline.parameters.c-default_docker_image >>
    resource_class: medium
    steps:
      - attach_workspace:
          at: /tmp/shadow-ci-workspace
      - run:
          name: Verify decision coherence with mainline CI
          command: |
            /tmp/shadow-ci-workspace/bin/coherence \
              --decision /tmp/shadow-ci-workspace/decision.json
      - store_artifacts:
          path: /tmp/shadow-ci-workspace/decision.json
          destination: shadow-ci/decision.json

  shadow-ci-tests:
    docker:
      - image: << pipeline.parameters.c-default_docker_image >>
    resource_class: large
    steps:
      - checkout
      - run:
          name: Install Go
          command: |
            curl -sSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
            echo 'export PATH=$PATH:/usr/local/go/bin' >> $BASH_ENV
      - run:
          name: Check shadow-ci.yml is not stale
          command: |
            cd ops/shadow-ci
            go run ./cmd/generate-ci \
              --config config \
              --check ../../.circleci/continue/shadow-ci.yml
      - run:
          name: Run shadow CI test suite
          command: |
            cd ops/shadow-ci
            go test -v -count=1 ./...
      - store_test_results:
          path: /tmp/shadow-ci-test-results
{{ range .Groups }}
  shadow-ci-{{ .Name }}:
    docker:
      - image: {{ .DockerImage }}
    resource_class: {{ .ResourceClass }}
    steps:
      - checkout
      - attach_workspace:
          at: /tmp/shadow-ci-workspace
      - run:
          name: Install toolchain ({{ .Name }})
          command: |
            {{ .SetupSteps }}
      - run:
          name: Execute {{ .Name }} categories
          no_output_timeout: 45m
          command: |
            /tmp/shadow-ci-workspace/bin/execute \
              --group {{ .Name }} \
              --decision /tmp/shadow-ci-workspace/decision.json \
              --config ops/shadow-ci/config \
              --results-dir /tmp/shadow-ci-test-results
      - store_test_results:
          path: /tmp/shadow-ci-test-results
      - store_artifacts:
          path: /tmp/shadow-ci-test-results
          destination: shadow-ci/{{ .Name }}
{{ end }}
workflows:
  shadow-ci:
    jobs:
      - shadow-ci-setup:
          context:
            - shadow-ci-test-context
      - shadow-ci-verify:
          requires:
            - shadow-ci-setup
      - shadow-ci-tests
{{ range .Groups }}      - shadow-ci-{{ .Name }}:
          requires:
            - shadow-ci-setup{{ range $dep, $_ := (index $.GroupDeps .Name) }}
            - shadow-ci-{{ $dep }}{{ end }}
{{ end }}`

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
